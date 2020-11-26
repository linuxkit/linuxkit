package testhelper

import "fmt"

type reader func(b []byte, offset int64) (int, error)
type writer func(b []byte, offset int64) (int, error)

// FileImpl implement github.com/diskfs/go-diskfs/util/File
// used for testing to enable stubbing out files
type FileImpl struct {
	Reader reader
	Writer writer
}

// ReadAt read at a particular offset
func (f *FileImpl) ReadAt(b []byte, offset int64) (int, error) {
	return f.Reader(b, offset)
}

// WriteAt write at a particular offset
func (f *FileImpl) WriteAt(b []byte, offset int64) (int, error) {
	return f.Writer(b, offset)
}

// Seek seek a particular offset - does not actually work
func (f *FileImpl) Seek(offset int64, whence int) (int64, error) {
	return 0, fmt.Errorf("FileImpl does not implement Seek()")
}
