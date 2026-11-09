# LinuxKit upload diagnostics

Sampling is off by default. Neither plain output nor redirection enables it.
After rebuilding LinuxKit, enable it explicitly:

```sh
linuxkit cache push --progress=plain --trace-upload IMAGE 2>upload.log
```

The default sample interval is five seconds; add `--trace-interval=2s` to change
it. The existing `--timeout` still controls the push deadline. A final sample
is written even when the push finishes before the first periodic sample.

Tracing requires plain output. `--progress=plain` works in a terminal; with
non-terminal stderr, automatic progress detection also permits tracing.
`--progress=tty --trace-upload` is rejected. Ordinary pushes do not create
the diagnostic sampler, recorder, or HTTP tracing hooks.

## Reading the output

Diagnostics are JSON lines on stderr, mixed with ordinary progress logs. Each
has `trace: "linuxkit-upload"`, `schema_version: 1`, a `push_id`, and elapsed
milliseconds. Extract just the diagnostic records with:

```sh
jq -R 'fromjson? | select(.trace == "linuxkit-upload")' upload.log
```

Instrumentation lives in LinuxKit's observed artifact readers, HTTP transport
(using Go's `httptrace`), and progress consumer. No dependency is modified.

- `artifact_start`, `artifact_open`, and `artifact_reader_done` identify each
  digest and reader attempt. Samples include their byte counts and rates.
  `recent_bps` measures approximately the last ten seconds (the actual window
  is `recent_window_ms`); `average_bps` measures the whole attempt. Retries
  count bytes again, so diagnostic byte counts measure work, not unique size.
- Large `source_read_ms` / `max_source_read_ms` points to slow local source
  reads. Large `between_reads_ms` / `max_between_reads_ms` means the consumer
  is waiting between reads, consistent with downstream backpressure. These
  alone cannot distinguish a slow connection from a slow registry.
- `request_start`, `request_body_done`, `request_headers`, and `request_done`
  describe each HTTP request. Samples include active requests and their phase.
  Compare connection wait, DNS, connect, TLS, body send, and response wait
  timings to locate the delay. Reused connections omit DNS/connect/TLS timings
  when no new work occurred. Connection IDs and reuse flags help correlate them.
- Request IDs join events; opaque session IDs join upload initialization,
  PATCH, and commit. `session_bind` associates a session with its digest when
  that becomes known. A failed PATCH can lack a digest if the library never
  reaches the commit; retain session records rather than assuming a match.
- `status`, `error_kind`, protocol, response bytes, and early-response flags
  help explain HTTP failures. Operation start/done records also cover time
  around registry lookup and upload operations outside individual requests.
- `progress_queue_depth`, capacity/high-water, and progress report times show
  whether progress reporting is backing up. Trace writer times and
  `dropped_records` identify slow diagnostic output. The bounded asynchronous
  trace queue drops events instead of blocking the upload; final sample and
  push-end records are preserved unless the output writer fails. Writer errors
  are returned to the caller.

Counters describe reads and writes on the client. A body read or completed
request write is not a registry acknowledgment. Distinguishing network limits
from registry throttling may still require comparing another connection or
registry after identifying the slow phase.

Authorization headers, query values, upload URLs, request/response bodies,
and raw error strings are not recorded. Session query state is hashed for
correlation. Records do include image references, registry hosts, digests,
sizes, timings, and HTTP status codes.

Validation uses local HTTP registries, cancellation, HTTP errors, redaction,
blocked output, CLI policy checks, and race tests; no public registry push is
performed by these tests.

## Patch

The change is applied to the working tree. An incremental copy is saved as
`/tmp/lkt/upload-trace.patch`, relative to the repository root. For another
checkout at the same starting revision:

```sh
git apply --check /tmp/lkt/upload-trace.patch
git apply /tmp/lkt/upload-trace.patch
```
