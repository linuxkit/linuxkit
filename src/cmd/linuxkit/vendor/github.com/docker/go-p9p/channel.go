package p9p

import (
	"bufio"
	"context"
	"encoding/binary"
	"io"
	"io/ioutil"
	"log"
	"net"
	"time"
)

const (
	// channelMessageHeaderSize is the overhead for sending the size of a
	// message on the wire.
	channelMessageHeaderSize = 4
)

// Channel defines the operations necessary to implement a 9p message channel
// interface. Typically, message channels do no protocol processing except to
// send and receive message frames.
type Channel interface {
	// ReadFcall reads one fcall frame into the provided fcall structure. The
	// Fcall may be cleared whether there is an error or not. If the operation
	// is successful, the contents of the fcall will be populated in the
	// argument. ReadFcall cannot be called concurrently with other calls to
	// ReadFcall. This both to preserve message ordering and to allow lockless
	// buffer reusage.
	ReadFcall(ctx context.Context, fcall *Fcall) error

	// WriteFcall writes the provided fcall to the channel. WriteFcall cannot
	// be called concurrently with other calls to WriteFcall.
	WriteFcall(ctx context.Context, fcall *Fcall) error

	// MSize returns the current msize for the channel.
	MSize() int

	// SetMSize sets the maximum message size for the channel. This must never
	// be called currently with ReadFcall or WriteFcall.
	SetMSize(msize int)
}

// NewChannel returns a new channel to read and write Fcalls with the provided
// connection and message size.
func NewChannel(conn net.Conn, msize int) Channel {
	return newChannel(conn, codec9p{}, msize)
}

const (
	defaultRWTimeout = 30 * time.Second // default read/write timeout if not set in context
)

// channel provides bidirectional protocol framing for 9p over net.Conn.
// Operations are not thread-safe but reads and writes may be carried out
// concurrently, supporting separate read and write loops.
//
// Lifecyle
//
// A connection, or message channel abstraction, has a lifecycle delineated by
// Tversion/Rversion request response cycles. For now, this is part of the
// channel itself but doesn't necessarily influence the channels state, except
// the msize. Visually, it might look something like this:
//
// 	[Established] -> [Version] -> [Session] -> [Version]---+
//	                     ^                                 |
// 	                     |_________________________________|
//
// The connection is established, then we negotiate a version, run a session,
// then negotiate a version and so on. For most purposes, we are likely going
// to terminate the connection after the session but we may want to support
// connection pooling. Pooling may result in possible security leaks if the
// connections are shared among contexts, since the version is negotiated at
// the start of the session. To avoid this, we can actually use a "tombstone"
// version message which clears the server's session state without starting a
// new session. The next version message would then prepare the session
// without leaking any Fid's.
type channel struct {
	conn   net.Conn
	codec  Codec
	brd    *bufio.Reader
	bwr    *bufio.Writer
	closed chan struct{}
	msize  int
	rdbuf  []byte
}

func newChannel(conn net.Conn, codec Codec, msize int) *channel {
	return &channel{
		conn:   conn,
		codec:  codec,
		brd:    bufio.NewReaderSize(conn, msize), // msize may not be optimal buffer size
		bwr:    bufio.NewWriterSize(conn, msize),
		closed: make(chan struct{}),
		msize:  msize,
		rdbuf:  make([]byte, msize),
	}
}

func (ch *channel) MSize() int {
	return ch.msize
}

// setmsize resizes the buffers for use with a separate msize. This call must
// be protected by a mutex or made before passing to other goroutines.
func (ch *channel) SetMSize(msize int) {
	// NOTE(stevvooe): We cannot safely resize the buffered reader and writer.
	// Proceed assuming that original size is sufficient.

	ch.msize = msize
	if msize < len(ch.rdbuf) {
		// just change the cap
		ch.rdbuf = ch.rdbuf[:msize]
		return
	}

	ch.rdbuf = make([]byte, msize)
}

// ReadFcall reads the next message from the channel into fcall.
//
// If the incoming message overflows the msize, Overflow(err) will return
// nonzero with the number of bytes overflowed.
func (ch *channel) ReadFcall(ctx context.Context, fcall *Fcall) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch.closed:
		return ErrClosed
	default:
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(defaultRWTimeout)
	}

	if err := ch.conn.SetReadDeadline(deadline); err != nil {
		log.Printf("transport: error setting read deadline on %v: %v", ch.conn.RemoteAddr(), err)
	}

	n, err := readmsg(ch.brd, ch.rdbuf)
	if err != nil {
		// TODO(stevvooe): There may be more we can do here to detect partial
		// reads. For now, we just propagate the error untouched.
		return err
	}

	if n > len(ch.rdbuf) {
		return overflowErr{size: n - len(ch.rdbuf)}
	}

	// clear out the fcall
	*fcall = Fcall{}
	if err := ch.codec.Unmarshal(ch.rdbuf[:n], fcall); err != nil {
		return err
	}

	if err := ch.maybeTruncate(fcall); err != nil {
		return err
	}

	return nil
}

// WriteFcall writes the message to the connection.
//
// If a message destined for the wire will overflow MSize, an Overflow error
// may be returned. For Twrite calls, the buffer will simply be truncated to
// the optimal msize, with the caller detecting this condition with
// Rwrite.Count.
func (ch *channel) WriteFcall(ctx context.Context, fcall *Fcall) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-ch.closed:
		return ErrClosed
	default:
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(defaultRWTimeout)
	}

	if err := ch.conn.SetWriteDeadline(deadline); err != nil {
		log.Printf("transport: error setting read deadline on %v: %v", ch.conn.RemoteAddr(), err)
	}

	if err := ch.maybeTruncate(fcall); err != nil {
		return err
	}

	p, err := ch.codec.Marshal(fcall)
	if err != nil {
		return err
	}

	if err := sendmsg(ch.bwr, p); err != nil {
		return err
	}

	return ch.bwr.Flush()
}

// maybeTruncate will truncate the message to fit into msize on the wire, if
// possible, or modify the message to ensure the response won't overflow.
//
// If the message cannot be truncated, an error will be returned and the
// message should not be sent.
//
// A nil return value means the message can be sent without
func (ch *channel) maybeTruncate(fcall *Fcall) error {

	// for certain message types, just remove the extra bytes from the data portion.
	switch msg := fcall.Message.(type) {
	// TODO(stevvooe): There is one more problematic message type:
	//
	// Rread: while we can employ the same truncation fix as Twrite, we
	// need to make it observable to upstream handlers.

	case MessageTread:
		// We can rewrite msg.Count so that a return message will be under
		// msize.  This is more defensive than anything but will ensure that
		// calls don't fail on sloppy servers.

		// first, craft the shape of the response message
		resp := newFcall(fcall.Tag, MessageRread{})
		overflow := uint32(ch.msgmsize(resp)) + msg.Count - uint32(ch.msize)

		if msg.Count < overflow {
			// Let the bad thing happen; msize too small to even support valid
			// rewrite. This will result in a Terror from the server-side or
			// just work.
			return nil
		}

		msg.Count -= overflow
		fcall.Message = msg

		return nil
	case MessageTwrite:
		// If we are going to overflow the msize, we need to truncate the write to
		// appropriate size or throw an error in all other conditions.
		size := ch.msgmsize(fcall)
		if size <= ch.msize {
			return nil
		}

		// overflow the msize, including the channel message size fields.
		overflow := size - ch.msize

		if len(msg.Data) < overflow {
			// paranoid: if msg.Data is not big enough to handle the
			// overflow, we should get an overflow error. MSize would have
			// to be way too small to be realistic.
			return overflowErr{size: overflow}
		}

		// The truncation is reflected in the return message (Rwrite) by
		// the server, so we don't need a return value or error condition
		// to communicate it.
		msg.Data = msg.Data[:len(msg.Data)-overflow]
		fcall.Message = msg // since we have a local copy

		return nil
	default:
		size := ch.msgmsize(fcall)
		if size > ch.msize {
			// overflow the msize, including the channel message size fields.
			return overflowErr{size: size - ch.msize}
		}

		return nil
	}

}

// msgmsize returns the on-wire msize of the Fcall, including the size header.
// Typically, this can be used to detect whether or not the message overflows
// the msize buffer.
func (ch *channel) msgmsize(fcall *Fcall) int {
	return channelMessageHeaderSize + ch.codec.Size(fcall)
}

// readmsg reads a 9p message into p from rd, ensuring that all bytes are
// consumed from the size header. If the size header indicates the message is
// larger than p, the entire message will be discarded, leaving a truncated
// portion in p. Any error should be treated as a framing error unless n is
// zero. The caller must check that n is less than or equal to len(p) to
// ensure that a valid message has been read.
func readmsg(rd io.Reader, p []byte) (n int, err error) {
	var msize uint32

	if err := binary.Read(rd, binary.LittleEndian, &msize); err != nil {
		return 0, err
	}

	n += binary.Size(msize)
	mbody := int(msize) - 4

	if mbody < len(p) {
		p = p[:mbody]
	}

	np, err := io.ReadFull(rd, p)
	if err != nil {
		return np + n, err
	}
	n += np

	if mbody > len(p) {
		// message has been read up to len(p) but we must consume the entire
		// message. This is an error condition but is non-fatal if we can
		// consume msize bytes.
		nn, err := io.CopyN(ioutil.Discard, rd, int64(mbody-len(p)))
		n += int(nn)
		if err != nil {
			return n, err
		}
	}

	return n, nil
}

// sendmsg writes a message of len(p) to wr with a 9p size header. All errors
// should be considered terminal.
func sendmsg(wr io.Writer, p []byte) error {
	size := uint32(len(p) + 4) // message size plus 4-bytes for size.
	if err := binary.Write(wr, binary.LittleEndian, size); err != nil {
		return err
	}

	// This assume partial writes to wr aren't possible. Not sure if this
	// valid. Matters during timeout retries.
	if n, err := wr.Write(p); err != nil {
		return err
	} else if n < len(p) {
		return io.ErrShortWrite
	}

	return nil
}
