package p9p

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	"context"
)

// roundTripper manages the request and response from the client-side. A
// roundTripper must abide by similar rules to the http.RoundTripper.
// Typically, the roundTripper will manage tag assignment and message
// serialization.
type roundTripper interface {
	send(ctx context.Context, msg Message) (Message, error)
}

// transport plays the role of being a client channel manager. It multiplexes
// function calls onto the wire and dispatches responses to blocking calls to
// send. On the whole, transport is thread-safe for calling send
type transport struct {
	ctx      context.Context
	ch       Channel
	requests chan *fcallRequest

	shutdown chan struct{}
	once     sync.Once // protect closure of shutdown
	closed   chan struct{}

	tags uint16
}

var _ roundTripper = &transport{}

func newTransport(ctx context.Context, ch Channel) roundTripper {
	t := &transport{
		ctx:      ctx,
		ch:       ch,
		requests: make(chan *fcallRequest),
		shutdown: make(chan struct{}),
		closed:   make(chan struct{}),
	}

	go t.handle()

	return t
}

// fcallRequest encompasses the request to send a message via fcall.
type fcallRequest struct {
	ctx      context.Context
	message  Message
	response chan *Fcall
	err      chan error
}

func newFcallRequest(ctx context.Context, msg Message) *fcallRequest {
	return &fcallRequest{
		ctx:      ctx,
		message:  msg,
		response: make(chan *Fcall, 1),
		err:      make(chan error, 1),
	}
}

func (t *transport) send(ctx context.Context, msg Message) (Message, error) {
	req := newFcallRequest(ctx, msg)

	// dispatch the request.
	select {
	case <-t.closed:
		return nil, ErrClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	case t.requests <- req:
	}

	// wait for the response.
	select {
	case <-t.closed:
		return nil, ErrClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-req.err:
		return nil, err
	case resp := <-req.response:
		if resp.Type == Rerror {
			// pack the error into something useful
			respmesg, ok := resp.Message.(MessageRerror)
			if !ok {
				return nil, fmt.Errorf("invalid error response: %v", resp)
			}

			return nil, respmesg
		}

		return resp.Message, nil
	}
}

// allocateTag returns a valid tag given a tag pool map. It receives a hint as
// to where to start the tag search. It returns an error if the allocation is
// not possible. The provided map must not contain NOTAG as a key.
func allocateTag(r *fcallRequest, m map[Tag]*fcallRequest, hint Tag) (Tag, error) {
	// The tag pool is depleted if 65535 (0xFFFF) tags are taken.
	if len(m) >= 0xFFFF {
		return 0, errors.New("tag pool depleted")
	}

	// Look for the first tag that doesn't exist in the map and return it.
	for i := 0; i < 0xFFFF; i++ {
		hint++
		if hint == NOTAG {
			hint = 0
		}

		if _, exists := m[hint]; !exists {
			return hint, nil
		}
	}

	return 0, errors.New("allocateTag: unexpected error")
}

// handle takes messages off the wire and wakes up the waiting tag call.
func (t *transport) handle() {
	defer func() {
		log.Println("exited handle loop")
		close(t.closed)
	}()

	// the following variable block are protected components owned by this thread.
	var (
		responses = make(chan *Fcall)
		// outstanding provides a map of tags to outstanding requests.
		outstanding = map[Tag]*fcallRequest{}
		selected    Tag
	)

	// loop to read messages off of the connection
	go func() {
		defer func() {
			log.Println("exited read loop")
			t.close() // single main loop
		}()
	loop:
		for {
			fcall := new(Fcall)
			if err := t.ch.ReadFcall(t.ctx, fcall); err != nil {
				switch err := err.(type) {
				case net.Error:
					if err.Timeout() || err.Temporary() {
						// BUG(stevvooe): There may be partial reads under
						// timeout errors where this is actually fatal.

						// can only retry if we haven't offset the frame.
						continue loop
					}
				}

				log.Println("fatal error reading msg:", err)
				return
			}

			select {
			case <-t.ctx.Done():
				return
			case <-t.closed:
				log.Println("transport closed")
				return
			case responses <- fcall:
			}
		}
	}()

	for {
		select {
		case req := <-t.requests:
			var err error

			selected, err = allocateTag(req, outstanding, selected)
			if err != nil {
				req.err <- err
				continue
			}

			outstanding[selected] = req
			fcall := newFcall(selected, req.message)

			// TODO(stevvooe): Consider the case of requests that never
			// receive a response. We need to remove the fcall context from
			// the tag map and dealloc the tag. We may also want to send a
			// flush for the tag.
			if err := t.ch.WriteFcall(req.ctx, fcall); err != nil {
				delete(outstanding, fcall.Tag)
				req.err <- err
			}
		case b := <-responses:
			req, ok := outstanding[b.Tag]
			if !ok {
				// BUG(stevvooe): The exact handling of an unknown tag is
				// unclear at this point. These may not necessarily fatal to
				// the session, since they could be messages that the client no
				// longer cares for. When we figure this out, replace this
				// panic with something more sensible.
				panic(fmt.Sprintf("unknown tag received: %v", b))
			}

			// BUG(stevvooe): Must detect duplicate tag and ensure that we are
			// waking up the right caller. If a duplicate is received, the
			// entry should not be deleted.
			delete(outstanding, b.Tag)

			req.response <- b

			// TODO(stevvooe): Reclaim tag id.
		case <-t.shutdown:
			return
		case <-t.ctx.Done():
			return
		}
	}
}

func (t *transport) flush(ctx context.Context, tag Tag) error {
	// TODO(stevvooe): We need to fire and forget flush messages when a call
	// context gets cancelled.
	panic("not implemented")
}

func (t *transport) Close() error {
	t.close()

	select {
	case <-t.closed:
		return nil
	case <-t.ctx.Done():
		return t.ctx.Err()
	}
}

// close starts the shutdown process.
func (t *transport) close() {
	t.once.Do(func() {
		close(t.shutdown)
	})
}
