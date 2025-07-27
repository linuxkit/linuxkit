package main

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	namepkg "github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/registry"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func pkgRemoteTagCmd() *cobra.Command {
	var release string
	cmd := &cobra.Command{
		Use:   "remote-tag",
		Short: "tag a package in a remote registry with another tag",
		Long: `Tag a package in a remote registry with another tag, without downloading or pulling.
		Will simply tag using the identical descriptor.
		First argument is "from" tag, second is "to" tag.

		If the "to" and "from" repositories are the same, then it is a simple tag operation.
		If they are not, then the "from" image is pulled and pushed to the "to" repository.
`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var finalErr error
			from := args[0]
			to := args[1]
			remoteOptions := []remote.Option{remote.WithAuthFromKeychain(authn.DefaultKeychain)}

			fromFullname := util.ReferenceExpand(from, util.ReferenceWithTag())
			toFullname := util.ReferenceExpand(to, util.ReferenceWithTag())
			fromRef, err := namepkg.ParseReference(fromFullname)
			if err != nil {
				return err
			}
			toRef, err := namepkg.ParseReference(toFullname)
			if err != nil {
				return err
			}
			fromDesc, err := registry.GetRemote().Get(fromRef, remoteOptions...)
			if err != nil {
				return fmt.Errorf("error getting manifest for from image %s: %v", fromFullname, err)
			}
			toDesc, err := registry.GetRemote().Get(toRef, remoteOptions...)
			if err == nil {
				if toDesc.Digest == fromDesc.Digest {
					log.Infof("image %s already exists in the registry, identical to %s, skipping", toFullname, fromFullname)
					return nil
				}
				log.Infof("image %s already exists in the registry, but is different from %s, overwriting", toFullname, fromFullname)
			}
			// see if they are from the same sources
			if fromRef.Context().String() == toRef.Context().String() {
				toTag, err := namepkg.NewTag(toFullname)
				if err != nil {
					return err
				}
				finalErr = registry.GetRemote().Tag(toTag, fromDesc, remoteOptions...)
			} else {
				// different, so need to copy
				finalErr = crane.Copy(fromFullname, toFullname)
			}

			return finalErr
		},
	}
	cmd.Flags().StringVar(&release, "release", "", "Release the given version")

	return cmd
}
