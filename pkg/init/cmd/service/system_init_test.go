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

// TestListContainersBeforeListen reproduces containerd's startup window: the
// socket file exists (bind) but connections are refused until listen.
func TestListContainersBeforeListen(t *testing.T) {
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

	ctrd := exec.Command("sleep", "60")
	if err := ctrd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ctrd.Process.Kill(); _ = ctrd.Wait() }()

	c, err := client.New(sock)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	ctx := namespaces.WithNamespace(context.Background(), "services.linuxkit")

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

	done := make(chan []client.Container, 1)
	go func() { done <- listContainers(ctx, ctrd, c) }()
	select {
	case ctrs := <-done:
		if len(ctrs) != 0 {
			t.Fatalf("got %d containers, want 0", len(ctrs))
		}
	case <-time.After(30 * time.Second):
		t.Fatal("listContainers did not return after containerd started listening")
	}
}
