package log

import (
	"flag"
	"os"

	"github.com/Sirupsen/logrus"
	"gopkg.in/inconshreveable/log15.v2"
)

// DefaultLogLevel is the default log level value.
var DefaultLogLevel = len(logrus.AllLevels) - 2

// SetLogLevel adjusts the logrus level.
func SetLogLevel(level int) {
	if level > len(logrus.AllLevels)-1 {
		level = len(logrus.AllLevels) - 1
	} else if level < 0 {
		level = 0
	}
	logrus.SetLevel(logrus.AllLevels[level])
}

// Options capture the logging configuration
type Options struct {
	Level     int
	Stdout    bool
	Format    string
	CallFunc  bool
	CallStack bool
}

// DevDefaults is the default options for development
var DevDefaults = Options{
	Level:     5,
	Stdout:    false,
	Format:    "json",
	CallStack: true,
}

// ProdDefaults is the default options for production
var ProdDefaults = Options{
	Level:    4,
	Stdout:   false,
	Format:   "term",
	CallFunc: true,
}

func init() {
	Configure(&DevDefaults)
}

// New returns a logger of given context
func New(ctx ...interface{}) log15.Logger {
	return log15.Root().New(ctx...)
}

// Root returns the process's root logger
func Root() log15.Logger {
	return log15.Root()
}

// Configure configures the logging
func Configure(options *Options) {

	SetLogLevel(options.Level)

	var f log15.Format
	switch options.Format {
	case "term":
		f = log15.TerminalFormat()
	case "json":
		f = log15.JsonFormatEx(true, true)
	case "logfmt":
		fallthrough
	default:
		f = log15.LogfmtFormat()
	}

	var h log15.Handler
	if options.Stdout {
		h = log15.StreamHandler(os.Stdout, f)
	} else {
		h = log15.StreamHandler(os.Stderr, f)
	}

	if options.CallFunc {
		h = log15.CallerFuncHandler(h)
	}
	if options.CallStack {
		h = log15.CallerStackHandler("%+v", h)
	}

	switch options.Level {
	case 0:
		h = log15.DiscardHandler() // no output
	case 1:
		h = log15.LvlFilterHandler(log15.LvlCrit, h)
	case 2:
		h = log15.LvlFilterHandler(log15.LvlError, h)
	case 3:
		h = log15.LvlFilterHandler(log15.LvlWarn, h)
	case 4:
		h = log15.LvlFilterHandler(log15.LvlInfo, h)
	case 5:
		h = log15.LvlFilterHandler(log15.LvlDebug, h)
	default:
		h = log15.LvlFilterHandler(log15.LvlInfo, h)
	}
	log15.Root().SetHandler(h)

	// Necessary to stop glog from complaining / noisy logs
	flag.CommandLine.Parse([]string{})
}
