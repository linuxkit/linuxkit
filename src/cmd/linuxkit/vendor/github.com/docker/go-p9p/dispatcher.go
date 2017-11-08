package p9p

import "context"

// Handler defines an interface for 9p message handlers. A handler
// implementation could be used to intercept calls of all types before sending
// them to the next handler.
type Handler interface {
	Handle(ctx context.Context, msg Message) (Message, error)

	// TODO(stevvooe): Right now, this interface is functianally identical to
	// roundtripper. If we find that this is sufficient on the server-side, we
	// may unify the types. For now, we leave them separated to differentiate
	// between them.
}

// HandlerFunc is a convenience type for defining inline handlers.
type HandlerFunc func(ctx context.Context, msg Message) (Message, error)

// Handle implements the requirements for the Handler interface.
func (fn HandlerFunc) Handle(ctx context.Context, msg Message) (Message, error) {
	return fn(ctx, msg)
}

// Dispatch returns a handler that dispatches messages to the target session.
// No concurrency is managed by the returned handler. It simply turns messages
// into function calls on the session.
func Dispatch(session Session) Handler {
	return HandlerFunc(func(ctx context.Context, msg Message) (Message, error) {
		switch msg := msg.(type) {
		case MessageTauth:
			qid, err := session.Auth(ctx, msg.Afid, msg.Uname, msg.Aname)
			if err != nil {
				return nil, err
			}

			return MessageRauth{Qid: qid}, nil
		case MessageTattach:
			qid, err := session.Attach(ctx, msg.Fid, msg.Afid, msg.Uname, msg.Aname)
			if err != nil {
				return nil, err
			}

			return MessageRattach{
				Qid: qid,
			}, nil
		case MessageTwalk:
			// TODO(stevvooe): This is one of the places where we need to manage
			// fid allocation lifecycle. We need to reserve the fid, then, if this
			// call succeeds, we should alloc the fid for future uses. Also need
			// to interact correctly with concurrent clunk and the flush of this
			// walk message.
			qids, err := session.Walk(ctx, msg.Fid, msg.Newfid, msg.Wnames...)
			if err != nil {
				return nil, err
			}

			return MessageRwalk{
				Qids: qids,
			}, nil
		case MessageTopen:
			qid, iounit, err := session.Open(ctx, msg.Fid, msg.Mode)
			if err != nil {
				return nil, err
			}

			return MessageRopen{
				Qid:    qid,
				IOUnit: iounit,
			}, nil
		case MessageTcreate:
			qid, iounit, err := session.Create(ctx, msg.Fid, msg.Name, msg.Perm, msg.Mode)
			if err != nil {
				return nil, err
			}

			return MessageRcreate{
				Qid:    qid,
				IOUnit: iounit,
			}, nil
		case MessageTread:
			p := make([]byte, int(msg.Count))
			n, err := session.Read(ctx, msg.Fid, p, int64(msg.Offset))
			if err != nil {
				return nil, err
			}

			return MessageRread{
				Data: p[:n],
			}, nil
		case MessageTwrite:
			n, err := session.Write(ctx, msg.Fid, msg.Data, int64(msg.Offset))
			if err != nil {
				return nil, err
			}

			return MessageRwrite{
				Count: uint32(n),
			}, nil
		case MessageTclunk:
			// TODO(stevvooe): Manage the clunking of file descriptors based on
			// walk and attach call progression.
			if err := session.Clunk(ctx, msg.Fid); err != nil {
				return nil, err
			}

			return MessageRclunk{}, nil
		case MessageTremove:
			if err := session.Remove(ctx, msg.Fid); err != nil {
				return nil, err
			}

			return MessageRremove{}, nil
		case MessageTstat:
			dir, err := session.Stat(ctx, msg.Fid)
			if err != nil {
				return nil, err
			}

			return MessageRstat{
				Stat: dir,
			}, nil
		case MessageTwstat:
			if err := session.WStat(ctx, msg.Fid, msg.Stat); err != nil {
				return nil, err
			}

			return MessageRwstat{}, nil
		default:
			return nil, ErrUnknownMsg
		}
	})
}
