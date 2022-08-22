package util

import (
	"flag"
	"fmt"
	stdlog "log"
	"os"

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

var (
	flagQuiet, flagVerbose *bool
)

// AddLoggingFlags add the logging flags to a flagset, or, if none given,
// the default flag package
func AddLoggingFlags(fs *flag.FlagSet) {
	// if we have no flagset, add it to the default flag package
	if fs == nil {
		flagQuiet = flag.Bool("q", false, "Quiet execution")
		flagVerbose = flag.Bool("v", false, "Verbose execution")
	} else {
		flagQuiet = fs.Bool("q", false, "Quiet execution")
		flagVerbose = fs.Bool("v", false, "Verbose execution")
	}
}

// SetupLogging once the flags have been parsed, setup the logging
func SetupLogging() {
	// Set up logging
	log.SetFormatter(new(infoFormatter))
	log.SetLevel(log.InfoLevel)
	flag.Parse()
	if *flagQuiet && *flagVerbose {
		fmt.Printf("Can't set quiet and verbose flag at the same time\n")
		os.Exit(1)
	}
	if *flagQuiet {
		log.SetLevel(log.ErrorLevel)
	}
	if *flagVerbose {
		// Switch back to the standard formatter
		log.SetFormatter(defaultLogFormatter)
		log.SetLevel(log.DebugLevel)
		// set go-containerregistry logging as well
		ggcrlog.Warn = stdlog.New(log.StandardLogger().WriterLevel(log.WarnLevel), "", 0)
		ggcrlog.Debug = stdlog.New(log.StandardLogger().WriterLevel(log.DebugLevel), "", 0)
	}
	ggcrlog.Progress = stdlog.New(log.StandardLogger().WriterLevel(log.InfoLevel), "", 0)
}
