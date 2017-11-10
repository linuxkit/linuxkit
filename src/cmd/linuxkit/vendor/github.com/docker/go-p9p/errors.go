package p9p

import (
	"errors"
	"fmt"
)

// MessageRerror provides both a Go error type and message type.
type MessageRerror struct {
	Ename string
}

// 9p wire errors returned by Session interface methods
var (
	ErrBadattach    = new9pError("unknown specifier in attach")
	ErrBadoffset    = new9pError("bad offset")
	ErrBadcount     = new9pError("bad count")
	ErrBotch        = new9pError("9P protocol botch")
	ErrCreatenondir = new9pError("create in non-directory")
	ErrDupfid       = new9pError("duplicate fid")
	ErrDuptag       = new9pError("duplicate tag")
	ErrIsdir        = new9pError("is a directory")
	ErrNocreate     = new9pError("create prohibited")
	ErrNomem        = new9pError("out of memory")
	ErrNoremove     = new9pError("remove prohibited")
	ErrNostat       = new9pError("stat prohibited")
	ErrNotfound     = new9pError("file not found")
	ErrNowrite      = new9pError("write prohibited")
	ErrNowstat      = new9pError("wstat prohibited")
	ErrPerm         = new9pError("permission denied")
	ErrUnknownfid   = new9pError("unknown fid")
	ErrBaddir       = new9pError("bad directory in wstat")
	ErrWalknodir    = new9pError("walk in non-directory")

	// extra errors not part of the normal protocol

	ErrTimeout       = new9pError("fcall timeout") // returned when timing out on the fcall
	ErrUnknownTag    = new9pError("unknown tag")
	ErrUnknownMsg    = new9pError("unknown message")    // returned when encountering unknown message type
	ErrUnexpectedMsg = new9pError("unexpected message") // returned when an unexpected message is encountered
	ErrWalkLimit     = new9pError("too many wnames in walk")
	ErrClosed        = errors.New("closed")
)

// new9pError returns a new 9p error ready for the wire.
func new9pError(s string) error {
	return MessageRerror{Ename: s}
}

// Type ensures that 9p errors can be transparently used as a 9p message in an
// Fcall.
func (MessageRerror) Type() FcallType {
	return Rerror
}

func (e MessageRerror) Error() string {
	return fmt.Sprintf("9p: %v", e.Ename)
}
