package p9p

import "fmt"

// Message represents the target of an fcall.
type Message interface {
	// Type returns the type of call for the target message.
	Type() FcallType
}

// newMessage returns a new instance of the message based on the Fcall type.
func newMessage(typ FcallType) (Message, error) {
	switch typ {
	case Tversion:
		return MessageTversion{}, nil
	case Rversion:
		return MessageRversion{}, nil
	case Tauth:
		return MessageTauth{}, nil
	case Rauth:
		return MessageRauth{}, nil
	case Tattach:
		return MessageTattach{}, nil
	case Rattach:
		return MessageRattach{}, nil
	case Rerror:
		return MessageRerror{}, nil
	case Tflush:
		return MessageTflush{}, nil
	case Rflush:
		return MessageRflush{}, nil // No message body for this response.
	case Twalk:
		return MessageTwalk{}, nil
	case Rwalk:
		return MessageRwalk{}, nil
	case Topen:
		return MessageTopen{}, nil
	case Ropen:
		return MessageRopen{}, nil
	case Tcreate:
		return MessageTcreate{}, nil
	case Rcreate:
		return MessageRcreate{}, nil
	case Tread:
		return MessageTread{}, nil
	case Rread:
		return MessageRread{}, nil
	case Twrite:
		return MessageTwrite{}, nil
	case Rwrite:
		return MessageRwrite{}, nil
	case Tclunk:
		return MessageTclunk{}, nil
	case Rclunk:
		return MessageRclunk{}, nil // no response body
	case Tremove:
		return MessageTremove{}, nil
	case Rremove:
		return MessageRremove{}, nil
	case Tstat:
		return MessageTstat{}, nil
	case Rstat:
		return MessageRstat{}, nil
	case Twstat:
		return MessageTwstat{}, nil
	case Rwstat:
		return MessageRwstat{}, nil
	}

	return nil, fmt.Errorf("unknown message type")
}

// MessageVersion encodes the message body for Tversion and Rversion RPC
// calls. The body is identical in both directions.
type MessageTversion struct {
	MSize   uint32
	Version string
}

type MessageRversion struct {
	MSize   uint32
	Version string
}

type MessageTauth struct {
	Afid  Fid
	Uname string
	Aname string
}

type MessageRauth struct {
	Qid Qid
}

type MessageTflush struct {
	Oldtag Tag
}

type MessageRflush struct{}

type MessageTattach struct {
	Fid   Fid
	Afid  Fid
	Uname string
	Aname string
}

type MessageRattach struct {
	Qid Qid
}

type MessageTwalk struct {
	Fid    Fid
	Newfid Fid
	Wnames []string
}

type MessageRwalk struct {
	Qids []Qid
}

type MessageTopen struct {
	Fid  Fid
	Mode Flag
}

type MessageRopen struct {
	Qid    Qid
	IOUnit uint32
}

type MessageTcreate struct {
	Fid  Fid
	Name string
	Perm uint32
	Mode Flag
}

type MessageRcreate struct {
	Qid    Qid
	IOUnit uint32
}

type MessageTread struct {
	Fid    Fid
	Offset uint64
	Count  uint32
}

type MessageRread struct {
	Data []byte
}

type MessageTwrite struct {
	Fid    Fid
	Offset uint64
	Data   []byte
}

type MessageRwrite struct {
	Count uint32
}

type MessageTclunk struct {
	Fid Fid
}

type MessageRclunk struct{}

type MessageTremove struct {
	Fid Fid
}

type MessageRremove struct{}

type MessageTstat struct {
	Fid Fid
}

type MessageRstat struct {
	Stat Dir
}

type MessageTwstat struct {
	Fid  Fid
	Stat Dir
}

type MessageRwstat struct{}

func (MessageTversion) Type() FcallType { return Tversion }
func (MessageRversion) Type() FcallType { return Rversion }
func (MessageTauth) Type() FcallType    { return Tauth }
func (MessageRauth) Type() FcallType    { return Rauth }
func (MessageTflush) Type() FcallType   { return Tflush }
func (MessageRflush) Type() FcallType   { return Rflush }
func (MessageTattach) Type() FcallType  { return Tattach }
func (MessageRattach) Type() FcallType  { return Rattach }
func (MessageTwalk) Type() FcallType    { return Twalk }
func (MessageRwalk) Type() FcallType    { return Rwalk }
func (MessageTopen) Type() FcallType    { return Topen }
func (MessageRopen) Type() FcallType    { return Ropen }
func (MessageTcreate) Type() FcallType  { return Tcreate }
func (MessageRcreate) Type() FcallType  { return Rcreate }
func (MessageTread) Type() FcallType    { return Tread }
func (MessageRread) Type() FcallType    { return Rread }
func (MessageTwrite) Type() FcallType   { return Twrite }
func (MessageRwrite) Type() FcallType   { return Rwrite }
func (MessageTclunk) Type() FcallType   { return Tclunk }
func (MessageRclunk) Type() FcallType   { return Rclunk }
func (MessageTremove) Type() FcallType  { return Tremove }
func (MessageRremove) Type() FcallType  { return Rremove }
func (MessageTstat) Type() FcallType    { return Tstat }
func (MessageRstat) Type() FcallType    { return Rstat }
func (MessageTwstat) Type() FcallType   { return Twstat }
func (MessageRwstat) Type() FcallType   { return Rwstat }
