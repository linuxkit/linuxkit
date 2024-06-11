package logutil

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

type Formatter struct {
	logrus.TextFormatter
}

func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	msg := bytes.NewBuffer(nil)
	fmt.Fprintf(msg, "%s: %s", strings.ToUpper(entry.Level.String()), entry.Message)
	if v, ok := entry.Data[logrus.ErrorKey]; ok {
		fmt.Fprintf(msg, ": %v", v)
	}
	fmt.Fprintf(msg, "\n")
	return msg.Bytes(), nil
}
