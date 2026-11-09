package util

import (
	"bytes"
	"testing"

	ggcrlog "github.com/google/go-containerregistry/pkg/logs"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestGGCRProgressIsSynchronousAndKeepsPlainMessages(t *testing.T) {
	logger := log.StandardLogger()
	output, formatter, level := logger.Out, logger.Formatter, logger.GetLevel()
	progress := ggcrlog.Progress
	defer func() {
		logger.SetOutput(output)
		logger.SetFormatter(formatter)
		logger.SetLevel(level)
		ggcrlog.Progress = progress
	}()
	var buffer bytes.Buffer
	require.NoError(t, SetupLogging(false, 1, false))
	logger.SetOutput(&buffer)
	ggcrlog.Progress.Print("existing blob: sha256:example")
	// A pipe-based Logrus WriterLevel bridge could return before formatting.
	// Completion must be delivered before the scoped terminal formatter closes.
	require.Equal(t, "existing blob: sha256:example\n", buffer.String())
	buffer.Reset()
	require.NoError(t, SetupLogging(true, 0, false))
	ggcrlog.Progress.Print("quiet completion")
	require.Empty(t, buffer.String())
}
