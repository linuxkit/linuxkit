package main

import (
	"context"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/containers"
	. "github.com/containerd/containerd/oci"
	"github.com/containerd/typeurl"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func WithMounterMounts(Source string, Destination string) SpecOpts {
	return func(ctx context.Context, client Client, c *containers.Container, s *specs.Spec) error {
		kubeletMountOpts := []string{"rbind", "rw", "shared"}
		kubeletMount := specs.Mount{
			Source:      Source,
			Destination: Destination,
			Type:        "bind",
			Options:     kubeletMountOpts}
		resolvMount := specs.Mount{
			Source:      "/etc/resolv.conf",
			Destination: "/etc/resolv.conf",
			Type:        "bind"}
		devMount := specs.Mount{
			Source:      "/dev",
			Destination: "/dev",
			Type:        "bind"}
		s.Mounts = append(s.Mounts, kubeletMount, resolvMount, devMount)
		return nil
	}
}

func updateContainerArgs(args []string) SpecOpts {
	return func(ctx context.Context, client Client, c *containers.Container, s *specs.Spec) error {
		s.Process.Args = args
		return nil
	}
}

func WithUpdatedSpecs(opts ...SpecOpts) containerd.UpdateContainerOpts {
	return func(ctx context.Context, client *containerd.Client, c *containers.Container) error {
		v, err := typeurl.UnmarshalAny(c.Spec)
		if err != nil {
			return err
		}
		rspecs := v.(*specs.Spec)
		for _, o := range opts {
			if err := o(ctx, client, c, rspecs); err != nil {
				return err
			}
		}
		c.Spec, err = typeurl.MarshalAny(rspecs)
		return err
	}
}
