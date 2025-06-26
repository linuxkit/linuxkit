// ALL writes to index.json at the root of the cache directory
// must be done through calls in this file. This is to ensure that it always does
// proper locking.
package cache

import (
	"errors"
	"fmt"
	"path/filepath"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
)

const (
	indexFile = "index.json"
)

// DescriptorWrite writes a descriptor to the cache index; it validates that it has a name
// and replaces any existing one
func (p *Provider) DescriptorWrite(image string, desc v1.Descriptor) error {
	if image == "" {
		return errors.New("cannot write descriptor without reference name")
	}
	if desc.Annotations == nil {
		desc.Annotations = map[string]string{}
	}
	desc.Annotations[imagespec.AnnotationRefName] = image
	log.Debugf("writing descriptor for image %s", image)

	// get our lock
	lock, err := util.Lock(filepath.Join(p.dir, indexFile))
	if err != nil {
		return fmt.Errorf("unable to lock cache index for writing descriptor for %s: %v", image, err)
	}
	defer func() {
		if err := lock.Unlock(); err != nil {
			log.Errorf("unable to close lock for cache index after writing descriptor for %s: %v", image, err)
		}
	}()

	// do we update an existing one? Or create a new one?
	if err := p.cache.RemoveDescriptors(match.Name(image)); err != nil {
		return fmt.Errorf("unable to remove old descriptors for %s: %v", image, err)
	}

	if err := p.cache.AppendDescriptor(desc); err != nil {
		return fmt.Errorf("unable to append new descriptor for %s: %v", image, err)
	}

	return nil
}

// RemoveDescriptors removes all descriptors that match the provided matcher.
// It does so in a parallel-access-safe way
func (p *Provider) RemoveDescriptors(matcher match.Matcher) error {
	// get our lock
	lock, err := util.Lock(filepath.Join(p.dir, indexFile))
	if err != nil {
		return fmt.Errorf("unable to lock cache index for removing descriptor for %v: %v", matcher, err)
	}
	defer func() {
		if err := lock.Unlock(); err != nil {
			log.Errorf("unable to close lock for cache index after writing descriptor for %v: %v", matcher, err)
		}
	}()
	return p.cache.RemoveDescriptors(matcher)
}
