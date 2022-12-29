package util

import (
	"errors"
	stdlog "log"

	ggcrlog "github.com/google/go-containerregistry/pkg/logs"
	log "github.com/sirupsen/logrus"
)

var (
	defaultLogFormatter = &log.TextFormatter{}
)

// infoFormatter overrides the default format for Info() log events to
// provide an easier to read output
type infoFormatter struct {
}

func (f *infoFormatter) Format(entry *log.Entry) ([]byte, error) {
	if entry.Level == log.InfoLevel {
		return append([]byte(entry.Message), '\n'), nil
	}
	return defaultLogFormatter.Format(entry)
}

// SetupLogging once the flags have been parsed, setup the logging
func SetupLogging(quiet, verbose bool) error {
	// Set up logging
	log.SetFormatter(new(infoFormatter))
	log.SetLevel(log.InfoLevel)
	if quiet && verbose {
		return errors.New("can't set quiet and verbose flag at the same time")
	}
	if quiet {
		log.SetLevel(log.ErrorLevel)
	}
	if verbose {
		// Switch back to the standard formatter
		log.SetFormatter(defaultLogFormatter)
		log.SetLevel(log.DebugLevel)
		// set go-containerregistry logging as well
		ggcrlog.Warn = stdlog.New(log.StandardLogger().WriterLevel(log.WarnLevel), "", 0)
		ggcrlog.Debug = stdlog.New(log.StandardLogger().WriterLevel(log.DebugLevel), "", 0)
	}
	ggcrlog.Progress = stdlog.New(log.StandardLogger().WriterLevel(log.InfoLevel), "", 0)
	return nil
}
