package datakit

import (
	"io"
	"log"
	"strings"

	p9p "github.com/docker/go-p9p"
	"context"
)

type watch struct {
	client *Client
	file   *File
	offset int64 // current offset within head.live file
}

type Watch struct {
	watch
}

// NewWatch starts watching a path within a branch
func NewWatch(ctx context.Context, client *Client, fromBranch string, path []string) (*Watch, error) {
	// SHA=$(cat branch/<fromBranch>/watch/<path.node>/tree.live)
	p := []string{"branch", fromBranch, "watch"}
	for _, dir := range path {
		p = append(p, dir+".node")
	}
	p = append(p, "tree.live")
	file, err := client.Open(ctx, p9p.OREAD, p...)
	if err != nil {
		log.Println("Failed to open", p, err)
		return nil, err
	}
	offset := int64(0)
	return &Watch{watch{client: client, file: file, offset: offset}}, nil
}

func (w *Watch) Next(ctx context.Context) (*Snapshot, error) {
	buf := make([]byte, 512)
	sawFlush := false
	for {
		// NOTE: irmin9p-direct will never return a fragment;
		// we can rely on the buffer containing a whold number
		// of lines.
		n, err := w.file.Read(ctx, buf, w.offset)
		if n == 0 {
			// Two reads of "" in a row means end-of-file
			if sawFlush {
				return nil, io.EOF
			} else {
				sawFlush = true
				continue
			}
		} else {
			sawFlush = false
		}
		w.offset = w.offset + int64(n)
		if err != nil {
			log.Println("Failed to Read head.live", err)
			return nil, io.EOF
		}
		lines := strings.Split(string(buf[0:n]), "\n")
		// Use the last non-empty line
		thing := ""
		for _, line := range lines {
			if line != "" {
				thing = line
			}
		}
		return NewSnapshot(ctx, w.client, OBJECT, thing), nil
	}
}

func (w *Watch) Close(ctx context.Context) {
	w.file.Close(ctx)
}
