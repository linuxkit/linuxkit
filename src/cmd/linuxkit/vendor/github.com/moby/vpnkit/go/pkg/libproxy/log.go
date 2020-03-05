package libproxy

import (
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

// SetLogger sets a new default logger
func SetLogger(l *logrus.Logger) {
	log = l
}
