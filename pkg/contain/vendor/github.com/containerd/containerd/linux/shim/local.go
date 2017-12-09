// +build !windows

package shim

import (
	"context"
	"path/filepath"

	shimapi "github.com/containerd/containerd/linux/shim/v1"
	ptypes "github.com/gogo/protobuf/types"
	"golang.org/x/sys/unix"
)

// NewLocal returns a shim client implementation for issue commands to a shim
func NewLocal(s *Service) shimapi.ShimService {
	return &local{
		s: s,
	}
}

type local struct {
	s *Service
}

func (c *local) Create(ctx context.Context, in *shimapi.CreateTaskRequest) (*shimapi.CreateTaskResponse, error) {
	return c.s.Create(ctx, in)
}

func (c *local) Start(ctx context.Context, in *shimapi.StartRequest) (*shimapi.StartResponse, error) {
	return c.s.Start(ctx, in)
}

func (c *local) Delete(ctx context.Context, in *ptypes.Empty) (*shimapi.DeleteResponse, error) {
	// make sure we unmount the containers rootfs for this local
	if err := unix.Unmount(filepath.Join(c.s.config.Path, "rootfs"), 0); err != nil {
		return nil, err
	}
	return c.s.Delete(ctx, in)
}

func (c *local) DeleteProcess(ctx context.Context, in *shimapi.DeleteProcessRequest) (*shimapi.DeleteResponse, error) {
	return c.s.DeleteProcess(ctx, in)
}

func (c *local) Exec(ctx context.Context, in *shimapi.ExecProcessRequest) (*ptypes.Empty, error) {
	return c.s.Exec(ctx, in)
}

func (c *local) ResizePty(ctx context.Context, in *shimapi.ResizePtyRequest) (*ptypes.Empty, error) {
	return c.s.ResizePty(ctx, in)
}

func (c *local) State(ctx context.Context, in *shimapi.StateRequest) (*shimapi.StateResponse, error) {
	return c.s.State(ctx, in)
}

func (c *local) Pause(ctx context.Context, in *ptypes.Empty) (*ptypes.Empty, error) {
	return c.s.Pause(ctx, in)
}

func (c *local) Resume(ctx context.Context, in *ptypes.Empty) (*ptypes.Empty, error) {
	return c.s.Resume(ctx, in)
}

func (c *local) Kill(ctx context.Context, in *shimapi.KillRequest) (*ptypes.Empty, error) {
	return c.s.Kill(ctx, in)
}

func (c *local) ListPids(ctx context.Context, in *shimapi.ListPidsRequest) (*shimapi.ListPidsResponse, error) {
	return c.s.ListPids(ctx, in)
}

func (c *local) CloseIO(ctx context.Context, in *shimapi.CloseIORequest) (*ptypes.Empty, error) {
	return c.s.CloseIO(ctx, in)
}

func (c *local) Checkpoint(ctx context.Context, in *shimapi.CheckpointTaskRequest) (*ptypes.Empty, error) {
	return c.s.Checkpoint(ctx, in)
}

func (c *local) ShimInfo(ctx context.Context, in *ptypes.Empty) (*shimapi.ShimInfoResponse, error) {
	return c.s.ShimInfo(ctx, in)
}

func (c *local) Update(ctx context.Context, in *shimapi.UpdateTaskRequest) (*ptypes.Empty, error) {
	return c.s.Update(ctx, in)
}

func (c *local) Wait(ctx context.Context, in *shimapi.WaitRequest) (*shimapi.WaitResponse, error) {
	return c.s.Wait(ctx, in)
}
