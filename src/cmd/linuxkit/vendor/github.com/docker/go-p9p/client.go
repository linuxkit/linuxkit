package p9p

import (
	"io"
	"net"

	"context"
)

type client struct {
	version   string
	msize     int
	ctx       context.Context
	transport roundTripper
}

// NewSession returns a session using the connection. The Context ctx provides
// a context for out of bad messages, such as flushes, that may be sent by the
// session. The session can effectively shutdown with this context.
func NewSession(ctx context.Context, conn net.Conn) (Session, error) {
	ch := newChannel(conn, codec9p{}, DefaultMSize) // sets msize, effectively.

	// negotiate the protocol version
	version, err := clientnegotiate(ctx, ch, DefaultVersion)
	if err != nil {
		return nil, err
	}

	return &client{
		version:   version,
		msize:     ch.MSize(),
		ctx:       ctx,
		transport: newTransport(ctx, ch),
	}, nil
}

var _ Session = &client{}

func (c *client) Version() (int, string) {
	return c.msize, c.version
}

func (c *client) Auth(ctx context.Context, afid Fid, uname, aname string) (Qid, error) {
	m := MessageTauth{
		Afid:  afid,
		Uname: uname,
		Aname: aname,
	}

	resp, err := c.transport.send(ctx, m)
	if err != nil {
		return Qid{}, err
	}

	rauth, ok := resp.(MessageRauth)
	if !ok {
		return Qid{}, ErrUnexpectedMsg
	}

	return rauth.Qid, nil
}

func (c *client) Attach(ctx context.Context, fid, afid Fid, uname, aname string) (Qid, error) {
	m := MessageTattach{
		Fid:   fid,
		Afid:  afid,
		Uname: uname,
		Aname: aname,
	}

	resp, err := c.transport.send(ctx, m)
	if err != nil {
		return Qid{}, err
	}

	rattach, ok := resp.(MessageRattach)
	if !ok {
		return Qid{}, ErrUnexpectedMsg
	}

	return rattach.Qid, nil
}

func (c *client) Clunk(ctx context.Context, fid Fid) error {
	resp, err := c.transport.send(ctx, MessageTclunk{
		Fid: fid,
	})
	if err != nil {
		return err
	}

	_, ok := resp.(MessageRclunk)
	if !ok {
		return ErrUnexpectedMsg
	}

	return nil
}

func (c *client) Remove(ctx context.Context, fid Fid) error {
	resp, err := c.transport.send(ctx, MessageTremove{
		Fid: fid,
	})
	if err != nil {
		return err
	}

	_, ok := resp.(MessageRremove)
	if !ok {
		return ErrUnexpectedMsg
	}

	return nil
}

func (c *client) Walk(ctx context.Context, fid Fid, newfid Fid, names ...string) ([]Qid, error) {
	if len(names) > 16 {
		return nil, ErrWalkLimit
	}

	resp, err := c.transport.send(ctx, MessageTwalk{
		Fid:    fid,
		Newfid: newfid,
		Wnames: names,
	})
	if err != nil {
		return nil, err
	}

	rwalk, ok := resp.(MessageRwalk)
	if !ok {
		return nil, ErrUnexpectedMsg
	}

	return rwalk.Qids, nil
}

func (c *client) Read(ctx context.Context, fid Fid, p []byte, offset int64) (n int, err error) {
	resp, err := c.transport.send(ctx, MessageTread{
		Fid:    fid,
		Offset: uint64(offset),
		Count:  uint32(len(p)),
	})
	if err != nil {
		return 0, err
	}

	rread, ok := resp.(MessageRread)
	if !ok {
		return 0, ErrUnexpectedMsg
	}

	n = copy(p, rread.Data)
	switch {
	case len(rread.Data) == 0:
		err = io.EOF
	case n < len(p):
		// TODO(stevvooe): Technically, we should treat this as an io.EOF.
		// However, we cannot tell if the short read was due to EOF or due to
		// truncation.
	}

	return n, err
}

func (c *client) Write(ctx context.Context, fid Fid, p []byte, offset int64) (n int, err error) {
	resp, err := c.transport.send(ctx, MessageTwrite{
		Fid:    fid,
		Offset: uint64(offset),
		Data:   p,
	})
	if err != nil {
		return 0, err
	}

	rwrite, ok := resp.(MessageRwrite)
	if !ok {
		return 0, ErrUnexpectedMsg
	}

	if int(rwrite.Count) < len(p) {
		err = io.ErrShortWrite
	}

	return int(rwrite.Count), err
}

func (c *client) Open(ctx context.Context, fid Fid, mode Flag) (Qid, uint32, error) {
	resp, err := c.transport.send(ctx, MessageTopen{
		Fid:  fid,
		Mode: mode,
	})
	if err != nil {
		return Qid{}, 0, err
	}

	ropen, ok := resp.(MessageRopen)
	if !ok {
		return Qid{}, 0, ErrUnexpectedMsg
	}

	return ropen.Qid, ropen.IOUnit, nil
}

func (c *client) Create(ctx context.Context, parent Fid, name string, perm uint32, mode Flag) (Qid, uint32, error) {
	resp, err := c.transport.send(ctx, MessageTcreate{
		Fid:  parent,
		Name: name,
		Perm: perm,
		Mode: mode,
	})
	if err != nil {
		return Qid{}, 0, err
	}

	rcreate, ok := resp.(MessageRcreate)
	if !ok {
		return Qid{}, 0, ErrUnexpectedMsg
	}

	return rcreate.Qid, rcreate.IOUnit, nil
}

func (c *client) Stat(ctx context.Context, fid Fid) (Dir, error) {
	resp, err := c.transport.send(ctx, MessageTstat{Fid: fid})
	if err != nil {
		return Dir{}, err
	}

	rstat, ok := resp.(MessageRstat)
	if !ok {
		return Dir{}, ErrUnexpectedMsg
	}

	return rstat.Stat, nil
}

func (c *client) WStat(ctx context.Context, fid Fid, dir Dir) error {
	resp, err := c.transport.send(ctx, MessageTwstat{
		Fid:  fid,
		Stat: dir,
	})
	if err != nil {
		return err
	}

	_, ok := resp.(MessageRwstat)
	if !ok {
		return ErrUnexpectedMsg
	}

	return nil
}
