package progress

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/docker/buildx/util/metricutil"
	"github.com/moby/buildkit/client"
	"github.com/opencontainers/go-digest"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type rePatterns struct {
	LocalSourceType *regexp.Regexp
	ImageSourceType *regexp.Regexp
	ExecType        *regexp.Regexp
	ExportImageType *regexp.Regexp
	LintMessage     *regexp.Regexp
}

var re = sync.OnceValue(func() *rePatterns {
	return &rePatterns{
		LocalSourceType: regexp.MustCompile(
			strings.Join([]string{
				`(?P<context>\[internal] load build context)`,
				`(?P<dockerfile>load build definition)`,
				`(?P<dockerignore>load \.dockerignore)`,
				`(?P<namedcontext>\[context .+] load from client)`,
			}, "|"),
		),
		ImageSourceType: regexp.MustCompile(`^\[.*] FROM `),
		ExecType:        regexp.MustCompile(`^\[.*] RUN `),
		ExportImageType: regexp.MustCompile(`^exporting to (image|(?P<format>\w+) image format)$`),
		LintMessage:     regexp.MustCompile(`^https://docs\.docker\.com/go/dockerfile/rule/([\w|-]+)/`),
	}
})

type metricWriter struct {
	recorders []metricRecorder
	attrs     attribute.Set
	mu        sync.Mutex
}

func newMetrics(mp metric.MeterProvider, attrs attribute.Set) *metricWriter {
	meter := metricutil.Meter(mp)
	return &metricWriter{
		recorders: []metricRecorder{
			newLocalSourceTransferMetricRecorder(meter, attrs),
			newImageSourceTransferMetricRecorder(meter, attrs),
			newExecMetricRecorder(meter, attrs),
			newExportImageMetricRecorder(meter, attrs),
			newIdleMetricRecorder(meter, attrs),
			newLintMetricRecorder(meter, attrs),
		},
		attrs: attrs,
	}
}

func (mw *metricWriter) Write(ss *client.SolveStatus) {
	mw.mu.Lock()
	defer mw.mu.Unlock()

	for _, recorder := range mw.recorders {
		recorder.Record(ss)
	}
}

type metricRecorder interface {
	Record(ss *client.SolveStatus)
}

type (
	localSourceTransferState struct {
		// Attributes holds the attributes specific to this context transfer.
		Attributes attribute.Set

		// LastTransferSize contains the last byte count for the transfer.
		LastTransferSize int64
	}
	localSourceTransferMetricRecorder struct {
		// BaseAttributes holds the set of base attributes for all metrics produced.
		BaseAttributes attribute.Set

		// State contains the state for individual digests that are being processed.
		State map[digest.Digest]*localSourceTransferState

		// TransferSize holds the metric for the number of bytes transferred.
		TransferSize metric.Int64Counter

		// Duration holds the metric for the total time taken to perform the transfer.
		Duration metric.Float64Counter
	}
)

func newLocalSourceTransferMetricRecorder(meter metric.Meter, attrs attribute.Set) *localSourceTransferMetricRecorder {
	mr := &localSourceTransferMetricRecorder{
		BaseAttributes: attrs,
		State:          make(map[digest.Digest]*localSourceTransferState),
	}
	mr.TransferSize, _ = meter.Int64Counter("source.local.transfer.io",
		metric.WithDescription("Measures the number of bytes transferred between the client and server for the context."),
		metric.WithUnit("By"))

	mr.Duration, _ = meter.Float64Counter("source.local.transfer.time",
		metric.WithDescription("Measures the length of time spent transferring the context."),
		metric.WithUnit("ms"))
	return mr
}

func (mr *localSourceTransferMetricRecorder) Record(ss *client.SolveStatus) {
	for _, v := range ss.Vertexes {
		state, ok := mr.State[v.Digest]
		if !ok {
			attr := detectLocalSourceType(v.Name)
			if !attr.Valid() {
				// Not a context transfer operation so just ignore.
				continue
			}

			state = &localSourceTransferState{
				Attributes: attribute.NewSet(attr),
			}
			mr.State[v.Digest] = state
		}

		if v.Started != nil && v.Completed != nil {
			dur := float64(v.Completed.Sub(*v.Started)) / float64(time.Millisecond)
			mr.Duration.Add(context.Background(), dur,
				metric.WithAttributeSet(mr.BaseAttributes),
				metric.WithAttributeSet(state.Attributes),
			)
		}
	}

	for _, status := range ss.Statuses {
		state, ok := mr.State[status.Vertex]
		if !ok {
			continue
		}

		if strings.HasPrefix(status.Name, "transferring") {
			diff := status.Current - state.LastTransferSize
			if diff > 0 {
				mr.TransferSize.Add(context.Background(), diff,
					metric.WithAttributeSet(mr.BaseAttributes),
					metric.WithAttributeSet(state.Attributes),
				)
			}
		}
	}
}

func detectLocalSourceType(vertexName string) attribute.KeyValue {
	match := re().LocalSourceType.FindStringSubmatch(vertexName)
	if match == nil {
		return attribute.KeyValue{}
	}

	for i, source := range re().LocalSourceType.SubexpNames() {
		if len(source) == 0 {
			// Not a subexpression.
			continue
		}

		// Did we find a match for this subexpression?
		if len(match[i]) > 0 {
			// Use the match name which corresponds to the name of the source.
			return attribute.String("source.local.type", source)
		}
	}
	// No matches found.
	return attribute.KeyValue{}
}

type (
	imageSourceMetricRecorder struct {
		// BaseAttributes holds the set of base attributes for all metrics produced.
		BaseAttributes attribute.Set

		// State holds the state for an individual digest. It is mostly used to check
		// if a status belongs to an image source since this recorder doesn't maintain
		// individual digest state.
		State map[digest.Digest]struct{}

		// TransferSize holds the counter for the transfer size.
		TransferSize metric.Int64Counter

		// TransferDuration holds the counter for the transfer duration.
		TransferDuration metric.Float64Counter

		// ExtractDuration holds the counter for the duration of image extraction.
		ExtractDuration metric.Float64Counter
	}
)

func newImageSourceTransferMetricRecorder(meter metric.Meter, attrs attribute.Set) *imageSourceMetricRecorder {
	mr := &imageSourceMetricRecorder{
		BaseAttributes: attrs,
		State:          make(map[digest.Digest]struct{}),
	}
	mr.TransferSize, _ = meter.Int64Counter("source.image.transfer.io",
		metric.WithDescription("Measures the number of bytes transferred for image content."),
		metric.WithUnit("By"))

	mr.TransferDuration, _ = meter.Float64Counter("source.image.transfer.time",
		metric.WithDescription("Measures the length of time spent transferring image content."),
		metric.WithUnit("ms"))

	mr.ExtractDuration, _ = meter.Float64Counter("source.image.extract.time",
		metric.WithDescription("Measures the length of time spent extracting image content."),
		metric.WithUnit("ms"))
	return mr
}

func (mr *imageSourceMetricRecorder) Record(ss *client.SolveStatus) {
	for _, v := range ss.Vertexes {
		if _, ok := mr.State[v.Digest]; !ok {
			if !detectImageSourceType(v.Name) {
				continue
			}
			mr.State[v.Digest] = struct{}{}
		}
	}

	for _, status := range ss.Statuses {
		// For this image type, we're only interested in completed statuses.
		if status.Completed == nil {
			continue
		}

		if status.Name == "extracting" {
			dur := float64(status.Completed.Sub(*status.Started)) / float64(time.Millisecond)
			mr.ExtractDuration.Add(context.Background(), dur,
				metric.WithAttributeSet(mr.BaseAttributes),
			)
			continue
		}

		// Remaining statuses will be associated with the from node.
		if _, ok := mr.State[status.Vertex]; !ok {
			continue
		}

		if strings.HasPrefix(status.ID, "sha256:") {
			// Signals a transfer. Record the duration and the size.
			dur := float64(status.Completed.Sub(*status.Started)) / float64(time.Millisecond)
			mr.TransferDuration.Add(context.Background(), dur,
				metric.WithAttributeSet(mr.BaseAttributes),
			)
			mr.TransferSize.Add(context.Background(), status.Total,
				metric.WithAttributeSet(mr.BaseAttributes),
			)
		}
	}
}

func detectImageSourceType(vertexName string) bool {
	return re().ImageSourceType.MatchString(vertexName)
}

type (
	execMetricRecorder struct {
		// Attributes holds the attributes for this metric recorder.
		Attributes attribute.Set

		// Duration tracks the duration of exec statements.
		Duration metric.Float64Counter
	}
)

func newExecMetricRecorder(meter metric.Meter, attrs attribute.Set) *execMetricRecorder {
	mr := &execMetricRecorder{
		Attributes: attrs,
	}
	mr.Duration, _ = meter.Float64Counter("exec.command.time",
		metric.WithDescription("Measures the length of time spent executing run statements."),
		metric.WithUnit("ms"))
	return mr
}

func (mr *execMetricRecorder) Record(ss *client.SolveStatus) {
	for _, v := range ss.Vertexes {
		if v.Started == nil || v.Completed == nil || !detectExecType(v.Name) {
			continue
		}

		dur := float64(v.Completed.Sub(*v.Started)) / float64(time.Millisecond)
		mr.Duration.Add(context.Background(), dur, metric.WithAttributeSet(mr.Attributes))
	}
}

func detectExecType(vertexName string) bool {
	return re().ExecType.MatchString(vertexName)
}

type (
	exportImageMetricRecorder struct {
		// Attributes holds the attributes for the export image metric.
		Attributes attribute.Set

		// Duration tracks the duration of image exporting.
		Duration metric.Float64Counter
	}
)

func newExportImageMetricRecorder(meter metric.Meter, attrs attribute.Set) *exportImageMetricRecorder {
	mr := &exportImageMetricRecorder{
		Attributes: attrs,
	}
	mr.Duration, _ = meter.Float64Counter("export.image.time",
		metric.WithDescription("Measures the length of time spent exporting the image."),
		metric.WithUnit("ms"))
	return mr
}

func (mr *exportImageMetricRecorder) Record(ss *client.SolveStatus) {
	for _, v := range ss.Vertexes {
		if v.Started == nil || v.Completed == nil {
			continue
		}

		format := detectExportImageType(v.Name)
		if format == "" {
			continue
		}

		dur := float64(v.Completed.Sub(*v.Started)) / float64(time.Millisecond)
		mr.Duration.Add(context.Background(), dur,
			metric.WithAttributeSet(mr.Attributes),
			metric.WithAttributes(
				attribute.String("image.format", format),
			),
		)
	}
}

func detectExportImageType(vertexName string) string {
	m := re().ExportImageType.FindStringSubmatch(vertexName)
	if m == nil {
		return ""
	}

	format := "docker"
	if m[2] != "" {
		format = m[2]
	}
	return format
}

type idleMetricRecorder struct {
	// Attributes holds the set of base attributes for all metrics produced.
	Attributes attribute.Set

	// Duration tracks the amount of time spent idle during this build.
	Duration metric.Float64ObservableGauge

	// Started stores the set of times when tasks were started.
	Started []time.Time

	// Completed stores the set of times when tasks were completed.
	Completed []time.Time

	mu sync.Mutex
}

func newIdleMetricRecorder(meter metric.Meter, attrs attribute.Set) *idleMetricRecorder {
	mr := &idleMetricRecorder{
		Attributes: attrs,
	}
	mr.Duration, _ = meter.Float64ObservableGauge("builder.idle.time",
		metric.WithDescription("Measures the length of time the builder spends idle."),
		metric.WithUnit("ms"),
		metric.WithFloat64Callback(mr.calculateIdleTime))
	return mr
}

func (mr *idleMetricRecorder) Record(ss *client.SolveStatus) {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	for _, v := range ss.Vertexes {
		if v.Started == nil || v.Completed == nil {
			continue
		}
		mr.Started = append(mr.Started, *v.Started)
		mr.Completed = append(mr.Completed, *v.Completed)
	}
}

// calculateIdleTime will use the recorded vertices that have been completed to determine the
// amount of time spent idle.
//
// This calculation isn't accurate until the build itself is completed. At the moment,
// metrics are only ever sent when a build is completed. If that changes, this calculation
// will likely be inaccurate.
func (mr *idleMetricRecorder) calculateIdleTime(_ context.Context, o metric.Float64Observer) error {
	mr.mu.Lock()
	defer mr.mu.Unlock()

	dur := calculateIdleTime(mr.Started, mr.Completed)
	o.Observe(float64(dur)/float64(time.Millisecond), metric.WithAttributeSet(mr.Attributes))
	return nil
}

func calculateIdleTime(started, completed []time.Time) time.Duration {
	sort.Slice(started, func(i, j int) bool {
		return started[i].Before(started[j])
	})
	sort.Slice(completed, func(i, j int) bool {
		return completed[i].Before(completed[j])
	})

	if len(started) == 0 {
		return 0
	}

	var (
		idleStart time.Time
		elapsed   time.Duration
	)
	for active := 0; len(started) > 0 && len(completed) > 0; {
		if started[0].Before(completed[0]) {
			if active == 0 && !idleStart.IsZero() {
				elapsed += started[0].Sub(idleStart)
			}
			active++
			started = started[1:]
			continue
		}

		active--
		if active == 0 {
			idleStart = completed[0]
		}
		completed = completed[1:]
	}
	return elapsed
}

type lintMetricRecorder struct {
	// Attributes holds the set of attributes for all metrics produced.
	Attributes attribute.Set

	// Count holds the metric for the number of times a lint rule has been triggered
	// within the current build.
	Count metric.Int64Counter
}

func newLintMetricRecorder(meter metric.Meter, attrs attribute.Set) *lintMetricRecorder {
	mr := &lintMetricRecorder{
		Attributes: attrs,
	}
	mr.Count, _ = meter.Int64Counter("lint.trigger.count",
		metric.WithDescription("Measures the number of times a lint rule has been triggered."))
	return mr
}

func kebabToCamel(s string) string {
	words := strings.Split(s, "-")
	for i, word := range words {
		words[i] = cases.Title(language.English).String(word)
	}
	return strings.Join(words, "")
}

var lintRuleNameProperty = attribute.Key("lint.rule.name")

func (mr *lintMetricRecorder) Record(ss *client.SolveStatus) {
	reLintMessage := re().LintMessage
	for _, warning := range ss.Warnings {
		m := reLintMessage.FindSubmatch([]byte(warning.URL))
		if len(m) < 2 {
			continue
		}

		ruleName := kebabToCamel(string(m[1]))
		mr.Count.Add(context.Background(), 1,
			metric.WithAttributeSet(mr.Attributes),
			metric.WithAttributes(
				lintRuleNameProperty.String(ruleName),
			),
		)
	}
}
