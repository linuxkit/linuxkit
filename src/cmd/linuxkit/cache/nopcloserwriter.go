package cache

import (
	"io"
)

type nopCloserWriter struct {
	writer io.Writer
}

func (n nopCloserWriter) Write(b []byte) (int, error) {
	return n.writer.Write(b)
}

func (n nopCloserWriter) Close() error {
	return nil
}

// NopCloserWriter wrap an io.Writer with a no-op Closer
func NopCloserWriter(writer io.Writer) io.WriteCloser {
	return nopCloserWriter{writer}
}
