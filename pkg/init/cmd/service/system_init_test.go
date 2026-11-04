package main

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	containersapi "github.com/containerd/containerd/api/services/containers/v1"
	"github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"google.golang.org/grpc"
)

type emptyContainers struct {
	containersapi.UnimplementedContainersServer
}

func (emptyContainers) ListStream(*containersapi.ListContainersRequest, containersapi.Containers_ListStreamServer) error {
	return nil
}

// bindOnly creates a unix socket file that refuses connections, as
// containerd's socket does between bind() and listen().
func bindOnly(t *testing.T) (int, string) {
	t.Helper()
	sock := filepath.Join(t.TempDir(), "containerd.sock")
	fd, err := syscall.Socket(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Bind(fd, &syscall.SockaddrUnix{Name: sock}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(sock); err != nil {
		t.Fatalf("socket file missing after bind: %v", err)
	}
	return fd, sock
}

// fakeContainerd starts a stand-in for the containerd process and a client
// for sock.
func fakeContainerd(t *testing.T, sock string) (*exec.Cmd, *client.Client, context.Context) {
	t.Helper()
	ctrd := sleeper(t)

	c, err := client.New(sock)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return ctrd, c, namespaces.WithNamespace(context.Background(), "services.linuxkit")
}

type listResult struct {
	ctrs []client.Container
	err  error
}

// TestListContainersBeforeListen reproduces containerd's startup window: the
// socket file exists (bind) but connections are refused until listen.
func TestListContainersBeforeListen(t *testing.T) {
	fd, sock := bindOnly(t)
	ctrd, c, ctx := fakeContainerd(t, sock)

	srv := grpc.NewServer()
	containersapi.RegisterContainersServer(srv, emptyContainers{})
	defer srv.Stop()
	go func() {
		time.Sleep(500 * time.Millisecond)
		if err := syscall.Listen(fd, 16); err != nil {
			t.Error(err)
			return
		}
		l, err := net.FileListener(os.NewFile(uintptr(fd), sock))
		if err != nil {
			t.Error(err)
			return
		}
		_ = srv.Serve(l)
	}()

	done := make(chan listResult, 1)
	go func() {
		ctrs, err := listContainers(ctx, ctrd, c, time.Now().Add(30*time.Second))
		done <- listResult{ctrs, err}
	}()
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatal(r.err)
		}
		if len(r.ctrs) != 0 {
			t.Fatalf("got %d containers, want 0", len(r.ctrs))
		}
	case <-time.After(30 * time.Second):
		t.Fatal("listContainers did not return after containerd started listening")
	}
}

// TestListContainersTimeout checks that a containerd which never listens is
// given up on once the deadline passes.
func TestListContainersTimeout(t *testing.T) {
	fd, sock := bindOnly(t)
	defer syscall.Close(fd)
	ctrd, c, ctx := fakeContainerd(t, sock)

	done := make(chan listResult, 1)
	go func() {
		ctrs, err := listContainers(ctx, ctrd, c, time.Now().Add(500*time.Millisecond))
		done <- listResult{ctrs, err}
	}()
	select {
	case r := <-done:
		if r.err == nil {
			t.Fatal("listContainers succeeded without a listening containerd")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("listContainers did not give up after its deadline")
	}
}

// sleeper starts a stand-in for the containerd process.
func sleeper(t *testing.T) *exec.Cmd {
	t.Helper()
	ctrd := exec.Command("sleep", "60")
	if err := ctrd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ctrd.Process.Kill(); _ = ctrd.Wait() })
	return ctrd
}

// TestWaitForSocketAppears checks that a socket file created after the wait
// starts ends it.
func TestWaitForSocketAppears(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "containerd.sock")
	ctrd := sleeper(t)
	go func() {
		time.Sleep(300 * time.Millisecond)
		if err := os.WriteFile(sock, nil, 0o600); err != nil {
			t.Error(err)
		}
	}()
	if err := waitForSocket(ctrd, sock, time.Now().Add(10*time.Second)); err != nil {
		t.Fatal(err)
	}
}

// TestWaitForSocketTimeout checks that a socket file which never appears is
// given up on once the deadline passes.
func TestWaitForSocketTimeout(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "containerd.sock")
	ctrd := sleeper(t)

	done := make(chan error, 1)
	go func() { done <- waitForSocket(ctrd, sock, time.Now().Add(500*time.Millisecond)) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("waitForSocket succeeded without a socket file")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("waitForSocket did not give up after its deadline")
	}
}
