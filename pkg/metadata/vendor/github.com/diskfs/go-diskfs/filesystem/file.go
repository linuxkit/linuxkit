package filesystem

import "io"

// File a reference to a single file on disk
type File interface {
	io.ReadWriteSeeker
	//io.ReaderAt
	//io.WriterAt
}
