package progress

import (
	"context"
	"os"
	"sync"

	"github.com/containerd/console"
	"github.com/docker/buildx/util/logutil"
	"github.com/moby/buildkit/client"
	"github.com/moby/buildkit/util/progress/progressui"
	"github.com/opencontainers/go-digest"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Printer struct {
	status chan *client.SolveStatus

	ready  chan struct{}
	done   chan struct{}
	paused chan struct{}

	err          error
	warnings     []client.VertexWarning
	logMu        sync.Mutex
	logSourceMap map[digest.Digest]interface{}
	metrics      *metricWriter

	// TODO: remove once we can use result context to pass build ref
	//  see https://github.com/docker/buildx/pull/1861
	buildRefsMu sync.Mutex
	buildRefs   map[string]string
}

func (p *Printer) Wait() error {
	close(p.status)
	<-p.done
	return p.err
}

func (p *Printer) Pause() error {
	p.paused = make(chan struct{})
	return p.Wait()
}

func (p *Printer) Unpause() {
	close(p.paused)
	<-p.ready
}

func (p *Printer) Write(s *client.SolveStatus) {
	p.status <- s
	if p.metrics != nil {
		p.metrics.Write(s)
	}
}

func (p *Printer) Warnings() []client.VertexWarning {
	return p.warnings
}

func (p *Printer) ValidateLogSource(dgst digest.Digest, v interface{}) bool {
	p.logMu.Lock()
	defer p.logMu.Unlock()
	src, ok := p.logSourceMap[dgst]
	if ok {
		if src == v {
			return true
		}
	} else {
		p.logSourceMap[dgst] = v
		return true
	}
	return false
}

func (p *Printer) ClearLogSource(v interface{}) {
	p.logMu.Lock()
	defer p.logMu.Unlock()
	for d := range p.logSourceMap {
		if p.logSourceMap[d] == v {
			delete(p.logSourceMap, d)
		}
	}
}

func NewPrinter(ctx context.Context, out console.File, mode progressui.DisplayMode, opts ...PrinterOpt) (*Printer, error) {
	opt := &printerOpts{}
	for _, o := range opts {
		o(opt)
	}

	if v := os.Getenv("BUILDKIT_PROGRESS"); v != "" && mode == progressui.AutoMode {
		mode = progressui.DisplayMode(v)
	}

	d, err := progressui.NewDisplay(out, mode, opt.displayOpts...)
	if err != nil {
		return nil, err
	}

	pw := &Printer{
		ready:   make(chan struct{}),
		metrics: opt.mw,
	}
	go func() {
		for {
			pw.status = make(chan *client.SolveStatus)
			pw.done = make(chan struct{})

			pw.logMu.Lock()
			pw.logSourceMap = map[digest.Digest]interface{}{}
			pw.logMu.Unlock()

			resumeLogs := logutil.Pause(logrus.StandardLogger())
			close(pw.ready)
			// not using shared context to not disrupt display but let is finish reporting errors
			pw.warnings, pw.err = d.UpdateFrom(ctx, pw.status)
			resumeLogs()
			close(pw.done)

			if opt.onclose != nil {
				opt.onclose()
			}
			if pw.paused == nil {
				break
			}

			pw.ready = make(chan struct{})
			<-pw.paused
			pw.paused = nil

			d, _ = progressui.NewDisplay(out, mode, opt.displayOpts...)
		}
	}()
	<-pw.ready
	return pw, nil
}

func (p *Printer) WriteBuildRef(target string, ref string) {
	p.buildRefsMu.Lock()
	defer p.buildRefsMu.Unlock()
	if p.buildRefs == nil {
		p.buildRefs = map[string]string{}
	}
	p.buildRefs[target] = ref
}

func (p *Printer) BuildRefs() map[string]string {
	return p.buildRefs
}

type printerOpts struct {
	displayOpts []progressui.DisplayOpt
	mw          *metricWriter

	onclose func()
}

type PrinterOpt func(b *printerOpts)

func WithPhase(phase string) PrinterOpt {
	return func(opt *printerOpts) {
		opt.displayOpts = append(opt.displayOpts, progressui.WithPhase(phase))
	}
}

func WithDesc(text string, console string) PrinterOpt {
	return func(opt *printerOpts) {
		opt.displayOpts = append(opt.displayOpts, progressui.WithDesc(text, console))
	}
}

func WithMetrics(mp metric.MeterProvider, attrs attribute.Set) PrinterOpt {
	return func(opt *printerOpts) {
		opt.mw = newMetrics(mp, attrs)
	}
}

func WithOnClose(onclose func()) PrinterOpt {
	return func(opt *printerOpts) {
		opt.onclose = onclose
	}
}
