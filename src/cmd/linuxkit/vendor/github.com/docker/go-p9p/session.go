package p9p

import "context"

// Session provides the central abstraction for a 9p connection. Clients
// implement sessions and servers serve sessions. Sessions can be proxied by
// serving up a client session.
//
// The interface is also wired up with full context support to manage timeouts
// and resource clean up.
//
// Session represents the operations covered in section 5 of the plan 9 manual
// (http://man.cat-v.org/plan_9/5/). Requests are managed internally, so the
// Flush method is handled by the internal implementation. Consider preceeding
// these all with context to control request timeout.
type Session interface {
	Auth(ctx context.Context, afid Fid, uname, aname string) (Qid, error)
	Attach(ctx context.Context, fid, afid Fid, uname, aname string) (Qid, error)
	Clunk(ctx context.Context, fid Fid) error
	Remove(ctx context.Context, fid Fid) error
	Walk(ctx context.Context, fid Fid, newfid Fid, names ...string) ([]Qid, error)

	// Read follows the semantics of io.ReaderAt.ReadAtt method except it takes
	// a contxt and Fid.
	Read(ctx context.Context, fid Fid, p []byte, offset int64) (n int, err error)

	// Write follows the semantics of io.WriterAt.WriteAt except takes a context and an Fid.
	//
	// If n == len(p), no error is returned.
	// If n < len(p), io.ErrShortWrite will be returned.
	Write(ctx context.Context, fid Fid, p []byte, offset int64) (n int, err error)

	Open(ctx context.Context, fid Fid, mode Flag) (Qid, uint32, error)
	Create(ctx context.Context, parent Fid, name string, perm uint32, mode Flag) (Qid, uint32, error)
	Stat(ctx context.Context, fid Fid) (Dir, error)
	WStat(ctx context.Context, fid Fid, dir Dir) error

	// Version returns the supported version and msize of the session. This
	// can be affected by negotiating or the level of support provided by the
	// session implementation.
	Version() (msize int, version string)
}
