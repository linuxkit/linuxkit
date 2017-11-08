package p9p

import "fmt"

// Overflow will return a positive number, indicating there was an overflow for
// the error.
func Overflow(err error) int {
	if of, ok := err.(overflow); ok {
		return of.Size()
	}

	// traverse cause, if above fails.
	if causal, ok := err.(interface {
		Cause() error
	}); ok {
		return Overflow(causal.Cause())
	}

	return 0
}

// overflow is a resolvable error type that can help callers negotiate
// session msize. If this error is encountered, no message was sent.
//
// The return value of `Size()` represents the number of bytes that would have
// been truncated if the message were sent. This IS NOT the optimal buffer size
// for operations like read and write.
//
// In the case of `Twrite`, the caller can Size() from the local size to get an
// optimally size buffer or the write can simply be truncated to `len(buf) -
// err.Size()`.
//
// For the most part, no users of this package should see this error in
// practice. If this escapes the Session interface, it is a bug.
type overflow interface {
	Size() int // number of bytes overflowed.
}

type overflowErr struct {
	size int // number of bytes overflowed
}

func (o overflowErr) Error() string {
	return fmt.Sprintf("message overflowed %d bytes", o.size)
}

func (o overflowErr) Size() int {
	return o.size
}
