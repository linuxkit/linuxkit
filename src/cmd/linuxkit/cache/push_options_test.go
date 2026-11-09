package cache

import (
	"bytes"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestProgressModeValidation(t *testing.T) {
	for _, mode := range []PushProgressMode{"", PushProgressAuto, PushProgressTTY, PushProgressPlain} {
		require.NoError(t, mode.Validate())
	}
	require.ErrorContains(t, PushProgressMode("bad").Validate(), "choose auto, tty, or plain")
}

func TestForcedProgressModes(t *testing.T) {
	for _, test := range []struct {
		mode  PushProgressMode
		level log.Level
		tty   bool
	}{{PushProgressAuto, log.InfoLevel, false}, {PushProgressPlain, log.InfoLevel, false}, {PushProgressTTY, log.InfoLevel, true}, {PushProgressTTY, log.ErrorLevel, false}} {
		t.Run(string(test.mode)+"/"+test.level.String(), func(t *testing.T) {
			logger := log.StandardLogger()
			out, formatter, level := logger.Out, logger.Formatter, logger.GetLevel()
			defer func() { logger.SetOutput(out); logger.SetFormatter(formatter); logger.SetLevel(level) }()
			var output bytes.Buffer
			logger.SetOutput(&output)
			logger.SetLevel(test.level)
			o := newPushObserver()
			o.progressMode = test.mode
			d := newPushDisplay(o, "progress mode test")
			defer d.Close()
			require.Equal(t, test.tty, d.interactive())
			now := time.Now()
			d.Report(pushProgress{started: now, lastChange: now})
			d.Finish(pushProgress{started: now, lastChange: now}, true)
			require.Equal(t, test.tty, bytes.Contains(output.Bytes(), []byte("\x1b[")))
			require.Same(t, &output, logger.Out)
			require.Same(t, formatter, logger.Formatter)
		})
	}
}
func TestForcedTTYDoesNotNestDisplays(t *testing.T) {
	logger := log.StandardLogger()
	out, formatter, level := logger.Out, logger.Formatter, logger.GetLevel()
	defer func() { logger.SetOutput(out); logger.SetFormatter(formatter); logger.SetLevel(level) }()
	var output bytes.Buffer
	logger.SetOutput(&output)
	logger.SetLevel(log.InfoLevel)
	o := newPushObserver()
	o.progressMode = PushProgressTTY
	first := newPushDisplay(o, "first")
	defer first.Close()
	require.True(t, first.interactive())
	other := newPushObserver()
	other.progressMode = PushProgressTTY
	second := newPushDisplay(other, "second")
	defer second.Close()
	require.False(t, second.interactive())
	second.Close()
	first.Close()
	require.Same(t, &output, logger.Out)
	require.Same(t, formatter, logger.Formatter)
}
