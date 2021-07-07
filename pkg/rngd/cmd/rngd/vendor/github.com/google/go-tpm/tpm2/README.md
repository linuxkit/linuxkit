# TPM 2.0 client library

## Integration tests

`tpm2_test.go` contains integration tests that run against a real TPM device
(emulated or hardware).

By default, running `go test` will skip them. To run the tests on a host with
available TPM device, run

```
go test --tpm_path=/dev/tpm0
```

where `/dev/tpm0` is path to a TPM character device or a Unix socket.

Tip: if your TPM host is remote and you don't want to install Go on it, use `go
test -c` to compile a test binary. It can be copied to remote host and run
without extra dependencies.
