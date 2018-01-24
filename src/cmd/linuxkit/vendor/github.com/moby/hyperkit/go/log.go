package hyperkit

import (
	"fmt"
	golog "log"
)

// Logger is an interface for logging.
type Logger interface {
	// Debugf logs a message with "debug" severity (very verbose).
	Debugf(format string, v ...interface{})
	// Infof logs a message with "info" severity (less verbose).
	Infof(format string, v ...interface{})
	// Warnf logs a message with "warn" (non-fatal) severity.
	Warnf(format string, v ...interface{})
	// Errorf logs an (non-fatal) error.
	Errorf(format string, v ...interface{})
	// Fatalf logs a fatal error message, and exits 1.
	Fatalf(format string, v ...interface{})
}

// StandardLogger makes the go standard logger comply to our Logger interface.
type StandardLogger struct{}

// Debugf logs a message with "debug" severity.
func (*StandardLogger) Debugf(f string, v ...interface{}) {
	golog.Printf("DEBUG: %v", fmt.Sprintf(f, v...))
}

// Infof logs a message with "info" severity.
func (*StandardLogger) Infof(f string, v ...interface{}) {
	golog.Printf("INFO : %v", fmt.Sprintf(f, v...))
}

// Warnf logs a message with "warn" (non-fatal) severity.
func (*StandardLogger) Warnf(f string, v ...interface{}) {
	golog.Printf("WARN : %v", fmt.Sprintf(f, v...))
}

// Errorf logs an (non-fatal) error.
func (*StandardLogger) Errorf(f string, v ...interface{}) {
	golog.Printf("ERROR: %v", fmt.Sprintf(f, v...))
}

// Fatalf logs a fatal error message, and exits 1.
func (*StandardLogger) Fatalf(f string, v ...interface{}) {
	golog.Fatalf("FATAL: %v", fmt.Sprintf(f, v...))
}

// Log receives stdout/stderr of the hyperkit process itself, if set.
// It defaults to the go standard logger.
var log Logger = &StandardLogger{}

// SetLogger sets the logger to use.
func SetLogger(l Logger) {
	log = l
}
