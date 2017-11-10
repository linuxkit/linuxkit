package p9p

import (
	"io"

	"context"
)

// ReaddirAll reads all the directory entries for the resource fid.
func ReaddirAll(session Session, fid Fid) ([]Dir, error) {
	panic("not implemented")
}

// Readdir helps one to implement the server-side of Session.Read on
// directories.
type Readdir struct {
	nextfn func() (Dir, error)
	buf    *Dir // one-item buffer
	codec  Codec
	offset int64
}

// NewReaddir returns a new Readdir to assist implementing server-side Readdir.
// The codec will be used to decode messages with Dir entries. The provided
// function next will be called until io.EOF is returned.
func NewReaddir(codec Codec, next func() (Dir, error)) *Readdir {
	return &Readdir{
		nextfn: next,
		codec:  codec,
	}
}

// NewFixedReaddir returns a Readdir that will returned a fixed set of
// directory entries.
func NewFixedReaddir(codec Codec, dir []Dir) *Readdir {
	dirs := make([]Dir, len(dir))
	copy(dirs, dir) // make our own copy!

	return NewReaddir(codec,
		func() (Dir, error) {
			if len(dirs) == 0 {
				return Dir{}, io.EOF
			}

			d := dirs[0]
			dirs = dirs[1:]
			return d, nil
		})
}

func (rd *Readdir) Read(ctx context.Context, p []byte, offset int64) (n int, err error) {
	if rd.offset != offset {
		return 0, ErrBadoffset
	}

	p = p[:0:len(p)]
	for len(p) < cap(p) {
		var d Dir
		if rd.buf != nil {
			d = *rd.buf
			rd.buf = nil
		} else {
			d, err = rd.nextfn()
			if err != nil {
				goto done
			}
		}

		var dp []byte
		dp, err = rd.codec.Marshal(d)
		if err != nil {
			goto done
		}

		if len(p)+len(dp) > cap(p) {
			// will over fill buffer. save item and exit.
			rd.buf = &d
			goto done
		}

		p = append(p, dp...)
	}

done:
	if err == io.EOF && len(p) > 0 {
		// Don't let io.EOF escape if we've written to p. 9p doesn't handle
		// this like Go.
		err = nil
	}

	rd.offset += int64(len(p))
	return len(p), err
}
