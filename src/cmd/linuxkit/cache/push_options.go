package cache

import (
	"fmt"
	"io"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"golang.org/x/term"
)

type PushProgressMode string

const (
	PushProgressAuto  PushProgressMode = "auto"
	PushProgressTTY   PushProgressMode = "tty"
	PushProgressPlain PushProgressMode = "plain"
)

func (mode PushProgressMode) Validate() error {
	switch mode {
	case "", PushProgressAuto, PushProgressTTY, PushProgressPlain:
		return nil
	}
	return fmt.Errorf("invalid push progress mode %q: choose auto, tty, or plain", mode)
}

// PushOptions controls presentation for one push. Zero values select automatic
// output detection. The caller supplies the timeout through the context.
type PushOptions struct {
	Progress      PushProgressMode
	TraceUpload   bool
	TraceInterval time.Duration
	// TraceWriter defaults to stderr. It is separate from ordinary progress logs.
	TraceWriter io.Writer
}

func (settings PushOptions) Validate() error {
	if err := settings.Progress.Validate(); err != nil {
		return err
	}
	if settings.TraceInterval < 0 {
		return fmt.Errorf("upload trace interval must not be negative")
	}
	if !settings.TraceUpload {
		return nil
	}
	if settings.Progress == PushProgressTTY {
		return fmt.Errorf("upload tracing requires --progress=plain or non-terminal output; --progress=tty is incompatible")
	}
	if settings.Progress == "" || settings.Progress == PushProgressAuto {
		pushDisplayMu.Lock()
		interactive := pushOutputIsTerminal(log.StandardLogger().Out)
		pushDisplayMu.Unlock()
		if interactive {
			return fmt.Errorf("upload tracing in a terminal requires --progress=plain")
		}
	}
	return nil
}

func (o *pushObserver) progressState(state pushProgress) pushProgress {
	rows, _ := o.treeSnapshot()
	state.complete, state.total = pushTreeTotals(rows)
	o.mu.Lock()
	lastRead := o.lastRead
	o.mu.Unlock()
	if lastRead.After(state.lastChange) {
		state.lastChange = lastRead
	}
	if !state.finished.IsZero() && state.lastChange.After(state.finished) {
		state.lastChange = state.finished
	}
	return state
}

func pushOutputIsTerminal(output io.Writer) bool {
	switch writer := output.(type) {
	case *os.File:
		return term.IsTerminal(int(writer.Fd()))
	case *pushTerminal:
		return pushOutputIsTerminal(writer.out)
	}
	return false
}
