// +build !linux

package fifo

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestFifoCloseAfterRm(t *testing.T) {
	tmpdir, err := ioutil.TempDir("", "fifos")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpdir)

	// non-linux version of this test leaks a goroutine

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	f, err := OpenFifo(ctx, filepath.Join(tmpdir, "f0"), syscall.O_RDONLY|syscall.O_CREAT|syscall.O_NONBLOCK, 0600)
	assert.NoError(t, err)

	time.Sleep(time.Second) // non-linux doesn't allow removing before syscall has been called. will cause an error.

	err = os.RemoveAll(filepath.Join(tmpdir, "f0"))
	assert.NoError(t, err)

	cerr := make(chan error)

	go func() {
		b := make([]byte, 32)
		_, err := f.Read(b)
		cerr <- err
	}()

	select {
	case err := <-cerr:
		t.Fatalf("read should have blocked, but got %v", err)
	case <-time.After(500 * time.Millisecond):
	}

	err = f.Close()
	assert.NoError(t, err)

	select {
	case err := <-cerr:
		assert.EqualError(t, err, "reading from a closed fifo")
	case <-time.After(500 * time.Millisecond):
		t.Fatal("read should have been unblocked")
	}

	cancel()
	ctx, cancel = context.WithCancel(context.Background())
	cerr = make(chan error)
	go func() {
		_, err = OpenFifo(ctx, filepath.Join(tmpdir, "f1"), syscall.O_WRONLY|syscall.O_CREAT, 0600)
		cerr <- err
	}()

	select {
	case err := <-cerr:
		t.Fatalf("open should have blocked, but got %v", err)
	case <-time.After(500 * time.Millisecond):
	}

	err = os.RemoveAll(filepath.Join(tmpdir, "f1"))
	cancel()

	select {
	case err := <-cerr:
		assert.Error(t, err)
	case <-time.After(500 * time.Millisecond):
		t.Fatal("open should have been unblocked")
	}
}
