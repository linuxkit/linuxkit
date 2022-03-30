package pkglib

import (
	"io"
)

type readerCtx struct {
	reader io.Reader
}

// Copy just copies from reader to writer
func (c *readerCtx) Copy(w io.WriteCloser) error {
	_, err := io.Copy(w, c.reader)
	if err != nil {
		return err
	}
	return w.Close()
}
