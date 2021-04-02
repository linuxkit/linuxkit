package cache

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	namepkg "github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/registry"
)

// PushWithManifest push an image along with, optionally, a multi-arch index.
func PushWithManifest(dir string, name, suffix string, pushImage, pushManifest bool) error {
	var (
		err     error
		options []remote.Option
	)
	p, err := Get(dir)
	if err != nil {
		return err
	}

	imageName := name + suffix
	ref, err := namepkg.ParseReference(imageName)
	if err != nil {
		return err
	}

	if pushImage {
		fmt.Printf("Pushing %s\n", imageName)
		// do we even have the given one?
		root, err := findRootFromLayout(p, imageName)
		if err != nil {
			return err
		}
		options = append(options, remote.WithAuthFromKeychain(authn.DefaultKeychain))
		img, err1 := root.Image()
		ii, err2 := root.ImageIndex()
		switch {
		case err1 == nil:
			if err := remote.Write(ref, img, options...); err != nil {
				return err
			}
			fmt.Printf("Pushed image %s\n", imageName)
		case err2 == nil:
			if err := remote.WriteIndex(ref, ii, options...); err != nil {
				return err
			}
			fmt.Printf("Pushed index %s\n", imageName)
		default:
			return fmt.Errorf("name %s unknown in cache", imageName)
		}
	} else {
		fmt.Print("Image push disabled, skipping...\n")
	}

	auth, err := registry.GetDockerAuth()
	if err != nil {
		return fmt.Errorf("failed to get auth: %v", err)
	}

	if pushManifest {
		fmt.Printf("Pushing %s to manifest %s\n", imageName, name)
		_, _, err = registry.PushManifest(imageName, auth)
		if err != nil {
			return err
		}
	} else {
		fmt.Print("Manifest push disabled, skipping...\n")
	}
	return nil
}
