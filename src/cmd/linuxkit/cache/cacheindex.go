// ALL writes to index.json at the root of the cache directory
// must be done through calls in this file. This is to ensure that it always does
// proper locking.
package cache

import (
	"errors"
	"fmt"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/match"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
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
	// will be replaced with locking
	return p.cache.RemoveDescriptors(matcher)
}
