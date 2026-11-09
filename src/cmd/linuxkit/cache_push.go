package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	cachepkg "github.com/linuxkit/linuxkit/src/cmd/linuxkit/cache"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	"github.com/spf13/cobra"
)

func cachePushCmd() *cobra.Command {
	timeout := cachepkg.DefaultPushTimeout
	progress := string(cachepkg.PushProgressAuto)
	traceUpload := false
	traceInterval := 5 * time.Second
	var (
		remoteName           string
		pushArchSpecificTags bool
		override             bool
	)
	cmd := &cobra.Command{
		Use:   "push",
		Short: "push images from the linuxkit cache",
		Long: `Push named images from the linuxkit cache to registry. Can provide short name, like linuxkit/kernel:6.6.13
		or nginx, or canonical name, like docker.io/library/nginx:latest.
		It is efficient, as blobs with the same content are not replaced.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode := cachepkg.PushProgressMode(progress)
			settings := cachepkg.PushOptions{Progress: mode, TraceUpload: traceUpload, TraceInterval: traceInterval}
			if err := settings.Validate(); err != nil {
				return err
			}
			if timeout < 0 {
				return fmt.Errorf("push timeout must not be negative")
			}
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			names := args
			for _, name := range names {
				fullname := util.ReferenceExpand(name)

				p, err := cachepkg.NewProvider(cacheDir)
				if err != nil {
					return fmt.Errorf("unable to read local cache: %w", err)
				}

				pushCtx := ctx
				cancel := func() {}
				if timeout > 0 {
					pushCtx, cancel = context.WithTimeout(ctx, timeout)
				}
				err = p.PushWithOptions(pushCtx, fullname, remoteName, pushArchSpecificTags, override, settings)
				cancel()
				if err != nil {
					return fmt.Errorf("unable to push image named %s: %w", name, err)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&remoteName, "remote-name", "", "Push it under a different name, e.g. push local image foo/bar:mine as baz/bee:yours. If blank, uses same local name.")
	cmd.Flags().BoolVar(&pushArchSpecificTags, "with-arch-tags", false, "When the local reference is an index, add to the remote arch-specific tags for each arch in the index, each as their own tag with the same name as the index, but with the architecture appended, e.g. image:foo will have image:foo-amd64, image:foo-arm64, etc.")
	cmd.Flags().BoolVar(&override, "override", false, "Even if the image already exists in the registry, push it again, overwriting the existing image.")
	cmd.Flags().StringVar(&progress, "progress", progress, "Progress output: auto, tty, or plain (plain disables terminal redraws)")
	cmd.Flags().BoolVar(&traceUpload, "trace-upload", false, "Opt in to sampled upload diagnostics on stderr (requires plain or non-terminal output)")
	cmd.Flags().DurationVar(&traceInterval, "trace-interval", traceInterval, "Sampling interval when --trace-upload is enabled")
	cmd.Flags().DurationVar(&timeout, "timeout", timeout, "Maximum duration for each image push, including registry lookups and retries (0 disables)")
	return cmd
}
