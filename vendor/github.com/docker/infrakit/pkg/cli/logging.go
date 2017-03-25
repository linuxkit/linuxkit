package cli

import log "github.com/Sirupsen/logrus"

// DefaultLogLevel is the default log level value.
var DefaultLogLevel = len(log.AllLevels) - 2

// SetLogLevel adjusts the logrus level.
func SetLogLevel(level int) {
	if level > len(log.AllLevels)-1 {
		level = len(log.AllLevels) - 1
	} else if level < 0 {
		level = 0
	}
	log.SetLevel(log.AllLevels[level])
}
