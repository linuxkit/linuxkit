package cache

import (
	"fmt"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/match"
	log "github.com/sirupsen/logrus"
)

// Remove removes all references pointed to by the provided reference, whether it is an image or an index.
// If it is not found, it is a no-op. This should be viewed as "Ensure this reference is not in the cache",
// rather than "Remove this reference from the cache".
func (p *Provider) Remove(name string) error {
	root, err := p.FindRoot(name)
	if err != nil {
		return err
	}
	var blobs []v1.Hash
	// the provided name could be an image or an index, so we need to check both
	img, err := root.Image()
	if err == nil {
		imgBlobs, err := blobsForImage(img)
		if err != nil {
			return err
		}
		blobs = append(blobs, imgBlobs...)
		imgDigest, err := img.Digest()
		if err != nil {
			return err
		}
		blobs = append(blobs, imgDigest)
	} else {
		ii, err := root.ImageIndex()
		if err != nil {
			return nil
		}
		// get blobs for each provided image
		manifests, err := ii.IndexManifest()
		if err != nil {
			return fmt.Errorf("unable to list manifests in index for %s: %v", name, err)
		}
		for _, man := range manifests.Manifests {
			img, err := ii.Image(man.Digest)
			if err != nil {
				return fmt.Errorf("unable to get image for digest %s in index for %s: %v", man.Digest, name, err)
			}
			imgBlobs, err := blobsForImage(img)
			if err != nil {
				return err
			}
			blobs = append(blobs, imgBlobs...)
			blobs = append(blobs, man.Digest)
		}
		indexDigest, err := ii.Digest()
		if err != nil {
			return err
		}
		blobs = append(blobs, indexDigest)
	}
	// at this point, blobs contains all of the blobs that need to be removed.
	for _, blob := range blobs {
		log.Debugf("removing blob %s", blob)
		if err := p.cache.RemoveBlob(blob); err != nil {
			log.Warnf("unable to remove blob %s for %s: %v", blob, name, err)
		}
	}
	return p.cache.RemoveDescriptors(match.Name(name))
}

func blobsForImage(img v1.Image) ([]v1.Hash, error) {
	var blobs []v1.Hash
	layers, err := img.Layers()
	if err != nil {
		// if we could not find the layers locally, that is fine;
		// we are trying to ensure they don't exist in the cache,
		// and they already don't exist.
		return nil, nil
	}
	for _, layer := range layers {
		dig, err := layer.Digest()
		if err != nil {
			return nil, err
		}
		blobs = append(blobs, dig)
	}
	if config, err := img.ConfigName(); err == nil {
		blobs = append(blobs, config)
	}
	return blobs, nil
}
