package p9p

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"context"
)

// TODO(stevvooe): Add net/http.Server-like type here to manage connections.
// Coupled with Handler mux, we can get a very http-like experience for 9p
// servers.

// ServeConn the 9p handler over the provided network connection.
func ServeConn(ctx context.Context, cn net.Conn, handler Handler) error {

	// TODO(stevvooe): It would be nice if the handler could declare the
	// supported version. Before we had handler, we used the session to get
	// the version (msize, version := session.Version()). We must decided if
	// we want to proxy version and message size decisions all the back to the
	// origin server or make those decisions at each link of a proxy chain.

	ch := newChannel(cn, codec9p{}, DefaultMSize)
	negctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	// TODO(stevvooe): For now, we negotiate here. It probably makes sense to
	// do this outside of this function and then pass in a ready made channel.
	// We are not really ready to export the channel type yet.

	if err := servernegotiate(negctx, ch, DefaultVersion); err != nil {
		// TODO(stevvooe): Need better error handling and retry support here.
		return fmt.Errorf("error negotiating version: %s", err)
	}

	ctx = withVersion(ctx, DefaultVersion)

	c := &conn{
		ctx:     ctx,
		ch:      ch,
		handler: handler,
		closed:  make(chan struct{}),
	}

	return c.serve()
}

// conn plays role of session dispatch for handler in a server.
type conn struct {
	ctx     context.Context
	session Session
	ch      Channel
	handler Handler

	once   sync.Once
	closed chan struct{}
	err    error // terminal error for the conn
}

// activeRequest includes information about the active request.
type activeRequest struct {
	ctx     context.Context
	request *Fcall
	cancel  context.CancelFunc
}

// serve messages on the connection until an error is encountered.
func (c *conn) serve() error {
	tags := map[Tag]*activeRequest{} // active requests

	requests := make(chan *Fcall)  // sync, read-limited
	responses := make(chan *Fcall) // sync, goroutine consumed
	completed := make(chan *Fcall) // sync, send in goroutine per request

	// read loop
	go c.read(requests)
	go c.write(responses)

	log.Println("server.run()")
	for {
		select {
		case req := <-requests:
			if _, ok := tags[req.Tag]; ok {
				select {
				case responses <- newErrorFcall(req.Tag, ErrDuptag):
					// Send to responses, bypass tag management.
				case <-c.ctx.Done():
					return c.ctx.Err()
				case <-c.closed:
					return c.err
				}
				continue
			}

			switch msg := req.Message.(type) {
			case MessageTflush:
				log.Println("server: flushing message", msg.Oldtag)

				var resp *Fcall
				// check if we have actually know about the requested flush
				active, ok := tags[msg.Oldtag]
				if ok {
					active.cancel() // propagate cancellation to callees
					delete(tags, msg.Oldtag)
					resp = newFcall(req.Tag, MessageRflush{})
				} else {
					resp = newErrorFcall(req.Tag, ErrUnknownTag)
				}

				select {
				case responses <- resp:
					// bypass tag management in completed.
				case <-c.ctx.Done():
					return c.ctx.Err()
				case <-c.closed:
					return c.err
				}
			default:
				// Allows us to session handlers to cancel processing of the fcall
				// through context.
				ctx, cancel := context.WithCancel(c.ctx)

				// The contents of these instances are only writable in the main
				// server loop. The value of tag will not change.
				tags[req.Tag] = &activeRequest{
					ctx:     ctx,
					request: req,
					cancel:  cancel,
				}

				go func(ctx context.Context, req *Fcall) {
					// TODO(stevvooe): Re-write incoming Treads so that handler
					// can always respond with a message of the correct msize.

					var resp *Fcall
					msg, err := c.handler.Handle(ctx, req.Message)
					if err != nil {
						// all handler errors are forwarded as protocol errors.
						resp = newErrorFcall(req.Tag, err)
					} else {
						resp = newFcall(req.Tag, msg)
					}

					select {
					case completed <- resp:
					case <-ctx.Done():
						return
					case <-c.closed:
						return
					}
				}(ctx, req)
			}
		case resp := <-completed:
			// only responses that flip the tag state traverse this section.
			active, ok := tags[resp.Tag]
			if !ok {
				// The tag is no longer active. Likely a flushed message.
				continue
			}

			select {
			case responses <- resp:
			case <-active.ctx.Done():
				// the context was canceled for some reason, perhaps timeout or
				// due to a flush call. We treat this as a condition where a
				// response should not be sent.
				log.Println("canceled", resp, active.ctx.Err())
			}
			delete(tags, resp.Tag)
		case <-c.ctx.Done():
			return c.ctx.Err()
		case <-c.closed:
			return c.err
		}
	}
}

// read takes requests off the channel and sends them on requests.
func (c *conn) read(requests chan *Fcall) {
	for {
		req := new(Fcall)
		if err := c.ch.ReadFcall(c.ctx, req); err != nil {
			if err, ok := err.(net.Error); ok {
				if err.Timeout() || err.Temporary() {
					// TODO(stevvooe): A full idle timeout on the connection
					// should be enforced here. No logging because it is quite
					// chatty.
					continue
				}
			}

			c.CloseWithError(fmt.Errorf("error reading fcall: %v", err))
			return
		}

		select {
		case requests <- req:
		case <-c.ctx.Done():
			c.CloseWithError(c.ctx.Err())
			return
		case <-c.closed:
			return
		}
	}
}

func (c *conn) write(responses chan *Fcall) {
	for {
		select {
		case resp := <-responses:
			// TODO(stevvooe): Correctly protect againt overflowing msize from
			// handler. This can be done above, in the main message handler
			// loop, by adjusting incoming Tread calls to have a Count that
			// won't overflow the msize.

			if err := c.ch.WriteFcall(c.ctx, resp); err != nil {
				if err, ok := err.(net.Error); ok {
					if err.Timeout() || err.Temporary() {
						// TODO(stevvooe): A full idle timeout on the
						// connection should be enforced here. We log here,
						// since this is less common.
						log.Printf("9p server: temporary error writing fcall: %v", err)
						continue
					}
				}

				c.CloseWithError(fmt.Errorf("error writing fcall: %v", err))
				return
			}
		case <-c.ctx.Done():
			c.CloseWithError(c.ctx.Err())
			return
		case <-c.closed:
			return
		}
	}
}

func (c *conn) Close() error {
	return c.CloseWithError(nil)
}

func (c *conn) CloseWithError(err error) error {
	c.once.Do(func() {
		if err == nil {
			err = ErrClosed
		}

		c.err = err
		close(c.closed)
	})

	return c.err
}
