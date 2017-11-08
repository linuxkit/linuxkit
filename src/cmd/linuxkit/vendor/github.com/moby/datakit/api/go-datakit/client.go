package datakit

import (
	"bytes"
	"io"
	"log"
	"net"
	"sync"

	p9p "github.com/docker/go-p9p"
	"context"
)

type Client struct {
	conn     net.Conn
	session  p9p.Session
	m        *sync.Mutex
	c        *sync.Cond
	usedFids map[p9p.Fid]bool
	freeFids []p9p.Fid
	root     p9p.Fid
}

var badFid = p9p.Fid(0)

var rwx = p9p.DMREAD | p9p.DMWRITE | p9p.DMEXEC
var rx = p9p.DMREAD | p9p.DMEXEC
var rw = p9p.DMREAD | p9p.DMWRITE
var r = p9p.DMREAD
var dirperm = uint32(rwx<<6 | rx<<3 | rx | p9p.DMDIR)
var fileperm = uint32(rw<<6 | r<<3 | r)

// Dial opens a connection to a 9P server
func Dial(ctx context.Context, network, address string) (*Client, error) {
	log.Println("Dialling", network, address)
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return NewClient(ctx, conn)
}

// NewClient creates opens a connection with the p9p server
func NewClient(ctx context.Context, conn net.Conn) (*Client, error) {
	session, err := p9p.NewSession(ctx, conn)
	if err != nil {
		log.Println("Failed to establish 9P session to", err)
		return nil, err
	}
	root := p9p.Fid(1)
	if _, err := session.Attach(ctx, root, p9p.NOFID, "anyone", "/"); err != nil {
		log.Println("Failed to Attach to filesystem", err)
		return nil, err
	}
	usedFids := make(map[p9p.Fid]bool, 0)
	freeFids := make([]p9p.Fid, 0)
	for i := 0; i < 128; i++ {
		fid := p9p.Fid(i)
		if fid == root {
			usedFids[fid] = true
		} else {
			freeFids = append(freeFids, fid)
			usedFids[fid] = false
		}
	}
	var m sync.Mutex
	c := sync.NewCond(&m)
	return &Client{conn, session, &m, c, usedFids, freeFids, root}, nil
}

func (c *Client) Close(ctx context.Context) {
	if err := c.session.Clunk(ctx, c.root); err != nil {
		log.Println("Failed to Clunk root fid", err)
	} else {
		c.usedFids[c.root] = false
	}
	c.m.Lock()
	defer c.m.Unlock()
	for fid, inuse := range c.usedFids {
		if inuse {
			log.Println("I don't know how to flush: leaking", fid)
		}
	}
	c.conn.Close()
}

// allocFid returns a fresh fid, bound to a clone of from
func (c *Client) allocFid(ctx context.Context, from p9p.Fid) (p9p.Fid, error) {
	c.m.Lock()
	defer c.m.Unlock()
	for len(c.freeFids) == 0 {
		c.c.Wait()
	}
	fid := c.freeFids[len(c.freeFids)-1]
	c.freeFids = c.freeFids[0 : len(c.freeFids)-1]
	c.usedFids[fid] = true
	_, err := c.session.Walk(ctx, from, fid)
	if err != nil {
		log.Println("Failed to clone root fid", err)
		return badFid, err
	}
	return fid, nil
}

// freeFid removes resources associated with the given fid
func (c *Client) freeFid(ctx context.Context, fid p9p.Fid) {
	c.m.Lock()
	defer c.m.Unlock()
	c.freeFids = append(c.freeFids, fid)
	c.usedFids[fid] = false
	if err := c.session.Clunk(ctx, fid); err != nil {
		log.Println("Failed to clunk fid", fid)
	}
	c.c.Signal()
}

// Mkdir acts like 'mkdir -p'
func (c *Client) Mkdir(ctx context.Context, path ...string) error {
	fid, err := c.allocFid(ctx, c.root)
	if err != nil {
		return nil
	}
	defer c.freeFid(ctx, fid)
	// mkdir -p
	for _, dir := range path {
		dirfid, err := c.allocFid(ctx, fid)
		if err != nil {
			return err
		}
		// dir may or may not exist
		_, _, _ = c.session.Create(ctx, dirfid, dir, dirperm, p9p.OREAD)
		c.freeFid(ctx, dirfid)
		// dir should definitely exist
		if _, err := c.session.Walk(ctx, fid, fid, dir); err != nil {
			log.Println("Failed to Walk to", dir, err)
			return err
		}
	}
	return nil
}

var enoent = p9p.MessageRerror{Ename: "No such file or directory"}
var enotdir = p9p.MessageRerror{Ename: "Can't walk from a file"}

// Remove acts like 'rm -f'
func (c *Client) Remove(ctx context.Context, path ...string) error {
	fid, err := c.allocFid(ctx, c.root)
	if err != nil {
		return err
	}
	if _, err := c.session.Walk(ctx, fid, fid, path...); err != nil {
		if err == enoent || err == enotdir {
			c.freeFid(ctx, fid)
			return nil
		}
		log.Println("Failed to walk to", path, err)
		c.freeFid(ctx, fid)
		return err
	}
	// Remove will cluck the fid, even if it fails
	if err := c.session.Remove(ctx, fid); err != nil {
		if err == enoent {
			return nil
		}
		log.Println("Failed to Remove", path, err)
		return err
	}
	return nil
}

type File struct {
	fid  p9p.Fid
	c    *Client
	m    *sync.Mutex
	open bool
}

// Create creates a file
func (c *Client) Create(ctx context.Context, path ...string) (*File, error) {
	fid, err := c.allocFid(ctx, c.root)
	if err != nil {
		return nil, err
	}
	dir := path[0 : len(path)-1]
	_, err = c.session.Walk(ctx, fid, fid, dir...)
	if err != nil {
		if err != enoent {
			// This is a common error
			log.Println("Failed to Walk to", path, err)
		}
		c.freeFid(ctx, fid)
		return nil, err
	}
	_, _, err = c.session.Create(ctx, fid, path[len(path)-1], fileperm, p9p.ORDWR)
	if err != nil {
		log.Println("Failed to Create", path, err)
		return nil, err
	}
	var m sync.Mutex
	return &File{fid: fid, c: c, m: &m, open: true}, nil
}

// Open opens a file
func (c *Client) Open(ctx context.Context, mode p9p.Flag, path ...string) (*File, error) {
	fid, err := c.allocFid(ctx, c.root)
	if err != nil {
		return nil, err
	}
	_, err = c.session.Walk(ctx, fid, fid, path...)
	if err != nil {
		if err != enoent {
			// This is a common error
			log.Println("Failed to Walk to", path, err)
		}
		c.freeFid(ctx, fid)
		return nil, err
	}
	_, _, err = c.session.Open(ctx, fid, mode)
	if err != nil {
		log.Println("Failed to Open", path, err)
		c.freeFid(ctx, fid)
		return nil, err
	}
	var m sync.Mutex
	return &File{fid: fid, c: c, m: &m, open: true}, nil
}

// List a directory
func (c *Client) List(ctx context.Context, path []string) ([]string, error) {
	file, err := c.Open(ctx, p9p.OREAD, path...)
	if err != nil {
		return nil, err
	}
	defer file.Close(ctx)

	msize, _ := c.session.Version()
	iounit := uint32(msize - 24) // size of message max minus fcall io header (Rread)

	p := make([]byte, iounit)

	n, err := c.session.Read(ctx, file.fid, p, 0)
	if err != nil {
		return nil, err
	}

	files := []string{}

	rd := bytes.NewReader(p[:n])
	codec := p9p.NewCodec() // TODO(stevvooe): Need way to resolve codec based on session.
	for {
		var d p9p.Dir
		if err := p9p.DecodeDir(codec, rd, &d); err != nil {
			if err == io.EOF {
				break
			}
			return files, err
		}
		files = append(files, d.Name)
	}
	return files, nil
}

// Close closes a file
func (f *File) Close(ctx context.Context) {
	f.m.Lock()
	defer f.m.Unlock()
	if f.open {
		f.c.freeFid(ctx, f.fid)
	}
	f.open = false
}

// Read reads a value
func (f *File) Read(ctx context.Context, p []byte, offset int64) (int, error) {
	f.m.Lock()
	defer f.m.Unlock()
	if !f.open {
		return 0, io.EOF
	}
	return f.c.session.Read(ctx, f.fid, p, offset)
}

// Write writes a value
func (f *File) Write(ctx context.Context, p []byte, offset int64) (int, error) {
	f.m.Lock()
	defer f.m.Unlock()
	if !f.open {
		return 0, io.EOF
	}
	return f.c.session.Write(ctx, f.fid, p, offset)
}

type FileReader struct {
	file   *File
	offset int64
	ctx    context.Context
}

func (f *File) NewFileReader(ctx context.Context) *FileReader {
	offset := int64(0)
	return &FileReader{file: f, offset: offset, ctx: ctx}
}

func (f *FileReader) Read(p []byte) (int, error) {
	n, err := f.file.Read(f.ctx, p, f.offset)
	f.offset = f.offset + int64(n)
	if n == 0 {
		return 0, io.EOF
	}
	return n, err
}

type ioFileReaderWriter struct {
	f      *File
	ctx    context.Context
	offset int64
}

// NewIOReader creates a standard io.Reader at a given position in the file
func (f *File) NewIOReader(ctx context.Context, offset int64) io.Reader {
	return &ioFileReaderWriter{f, ctx, offset}
}

// NewIOWriter creates a standard io.Writer at a given position in the file
func (f *File) NewIOWriter(ctx context.Context, offset int64) io.Writer {
	return &ioFileReaderWriter{f, ctx, offset}
}

func (r *ioFileReaderWriter) Read(p []byte) (n int, err error) {

	r.f.m.Lock()
	defer r.f.m.Unlock()
	n, err = r.f.c.session.Read(r.ctx, r.f.fid, p, r.offset)

	r.offset += int64(n)
	return n, err
}
func (w *ioFileReaderWriter) Write(p []byte) (n int, err error) {
	w.f.m.Lock()
	defer w.f.m.Unlock()
	for err == nil || err == io.ErrShortWrite {
		var written int
		written, err = w.f.c.session.Write(w.ctx, w.f.fid, p, w.offset)
		p = p[written:]
		w.offset += int64(written)
		n += written
		if len(p) == 0 {
			break
		}
	}
	return
}
