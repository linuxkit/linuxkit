package p9p

import "fmt"

// FcallType encodes the message type for the target Fcall.
type FcallType uint8

// Definitions for Fcall's used in 9P2000.
const (
	Tversion FcallType = iota + 100
	Rversion
	Tauth
	Rauth
	Tattach
	Rattach
	Terror
	Rerror
	Tflush
	Rflush
	Twalk
	Rwalk
	Topen
	Ropen
	Tcreate
	Rcreate
	Tread
	Rread
	Twrite
	Rwrite
	Tclunk
	Rclunk
	Tremove
	Rremove
	Tstat
	Rstat
	Twstat
	Rwstat
	Tmax
)

func (fct FcallType) String() string {
	switch fct {
	case Tversion:
		return "Tversion"
	case Rversion:
		return "Rversion"
	case Tauth:
		return "Tauth"
	case Rauth:
		return "Rauth"
	case Tattach:
		return "Tattach"
	case Rattach:
		return "Rattach"
	case Terror:
		// invalid.
		return "Terror"
	case Rerror:
		return "Rerror"
	case Tflush:
		return "Tflush"
	case Rflush:
		return "Rflush"
	case Twalk:
		return "Twalk"
	case Rwalk:
		return "Rwalk"
	case Topen:
		return "Topen"
	case Ropen:
		return "Ropen"
	case Tcreate:
		return "Tcreate"
	case Rcreate:
		return "Rcreate"
	case Tread:
		return "Tread"
	case Rread:
		return "Rread"
	case Twrite:
		return "Twrite"
	case Rwrite:
		return "Rwrite"
	case Tclunk:
		return "Tclunk"
	case Rclunk:
		return "Rclunk"
	case Tremove:
		return "Tremove"
	case Rremove:
		return "Rremove"
	case Tstat:
		return "Tstat"
	case Rstat:
		return "Rstat"
	case Twstat:
		return "Twstat"
	case Rwstat:
		return "Rwstat"
	default:
		return "Tunknown"
	}
}

// Fcall defines the fields for sending a 9p formatted message. The type will
// be introspected from the Message implementation.
type Fcall struct {
	Type    FcallType
	Tag     Tag
	Message Message
}

func newFcall(tag Tag, msg Message) *Fcall {
	return &Fcall{
		Type:    msg.Type(),
		Tag:     tag,
		Message: msg,
	}
}

func newErrorFcall(tag Tag, err error) *Fcall {
	var msg Message

	switch v := err.(type) {
	case MessageRerror:
		msg = v
	case *MessageRerror:
		msg = *v
	default:
		msg = MessageRerror{Ename: v.Error()}
	}

	return &Fcall{
		Type:    Rerror,
		Tag:     tag,
		Message: msg,
	}
}

func (fc *Fcall) String() string {
	return fmt.Sprintf("%v(%v) %v", fc.Type, fc.Tag, string9p(fc.Message))
}
