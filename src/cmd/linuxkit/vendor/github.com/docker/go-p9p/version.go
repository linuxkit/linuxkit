package p9p

import (
	"fmt"

	"context"
)

// NOTE(stevvooe): This file contains functions for negotiating version on the
// client and server. There are some nasty details to get right for
// downgrading the connection on the server-side that are not present yet.
// Really, these should be refactored into some sort of channel type that can
// support resets through version messages during the protocol exchange.

// clientnegotiate negiotiates the protocol version using channel, blocking
// until a response is received. The received value will be the version
// implemented by the server.
func clientnegotiate(ctx context.Context, ch Channel, version string) (string, error) {
	req := newFcall(NOTAG, MessageTversion{
		MSize:   uint32(ch.MSize()),
		Version: version,
	})

	if err := ch.WriteFcall(ctx, req); err != nil {
		return "", err
	}

	resp := new(Fcall)
	if err := ch.ReadFcall(ctx, resp); err != nil {
		return "", err
	}

	switch v := resp.Message.(type) {
	case MessageRversion:

		if v.Version != version {
			// TODO(stevvooe): A stubborn client indeed!
			return "", fmt.Errorf("unsupported server version: %v", version)
		}

		if int(v.MSize) < ch.MSize() {
			// upgrade msize if server differs.
			ch.SetMSize(int(v.MSize))
		}

		return v.Version, nil
	case error:
		return "", v
	default:
		return "", ErrUnexpectedMsg
	}
}

// servernegotiate blocks until a version message is received or a timeout
// occurs. The msize for the tranport will be set from the negotiation. If
// negotiate returns nil, a server may proceed with the connection.
//
// In the future, it might be better to handle the version messages in a
// separate object that manages the session. Each set of version requests
// effectively "reset" a connection, meaning all fids get clunked and all
// outstanding IO is aborted. This is probably slightly racy, in practice with
// a misbehaved client. The main issue is that we cannot tell which session
// messages belong to.
func servernegotiate(ctx context.Context, ch Channel, version string) error {
	// wait for the version message over the transport.
	req := new(Fcall)
	if err := ch.ReadFcall(ctx, req); err != nil {
		return err
	}

	mv, ok := req.Message.(MessageTversion)
	if !ok {
		return fmt.Errorf("expected version message: %v", mv)
	}

	respmsg := MessageRversion{
		Version: version,
	}

	if mv.Version != version {
		// TODO(stevvooe): Not the best place to do version handling. We need
		// to have a way to pass supported versions into this method then have
		// it return the actual version. For now, respond with 9P2000 for
		// anything that doesn't match the provided version string.
		//
		// version(9) says "The server may respond with the client’s
		// version string, or a version string identifying an earlier
		// defined protocol version. Currently, the only defined
		// version is the 6 characters 9P2000." Therefore, it is always
		// OK to respond with this.
		respmsg.Version = "9P2000"
	}

	if int(mv.MSize) < ch.MSize() {
		// if the server msize is too large, use the client's suggested msize.
		ch.SetMSize(int(mv.MSize))
		respmsg.MSize = mv.MSize
	} else {
		respmsg.MSize = uint32(ch.MSize())
	}

	resp := newFcall(NOTAG, respmsg)
	if err := ch.WriteFcall(ctx, resp); err != nil {
		return err
	}

	if respmsg.Version == "unknown" {
		return fmt.Errorf("bad version negotiation")
	}

	return nil
}
