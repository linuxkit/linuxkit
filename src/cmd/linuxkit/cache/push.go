package cache

import (
	"fmt"

	"github.com/google/go-containerregistry/pkg/authn"
	namepkg "github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/registry"
)

// PushWithManifest push an image along with, optionally, a multi-arch index.
func (p *Provider) PushWithManifest(name, suffix string, pushImage, pushManifest bool) error {
	var (
		err     error
		options []remote.Option
	)
	imageName := name + suffix
	ref, err := namepkg.ParseReference(imageName)
	if err != nil {
		return err
	}

	if pushImage {
		fmt.Printf("Pushing %s\n", imageName)
		// do we even have the given one?
		root, err := p.FindRoot(imageName)
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
		_, _, err = registry.PushManifest(name, auth)
		if err != nil {
			return err
		}
	} else {
		fmt.Print("Manifest push disabled, skipping...\n")
	}
	return nil
}

// Push push an image along with a multi-arch index.
func (p *Provider) Push(name string) error {
	var (
		err     error
		options []remote.Option
	)
	ref, err := namepkg.ParseReference(name)
	if err != nil {
		return err
	}

	fmt.Printf("Pushing %s\n", name)
	// do we even have the given one?
	root, err := p.FindRoot(name)
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
		fmt.Printf("Pushed image %s\n", name)
	case err2 == nil:
		if err := remote.WriteIndex(ref, ii, options...); err != nil {
			return err
		}
		fmt.Printf("Pushed index %s\n", name)
	default:
		return fmt.Errorf("name %s unknown in cache", name)
	}
	return nil
}
