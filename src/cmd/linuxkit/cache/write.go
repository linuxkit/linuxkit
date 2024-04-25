package cache

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/containerd/containerd/images"
	"github.com/containerd/containerd/reference"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	lktspec "github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	lktutil "github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
)

const (
	linux = "linux"
)

// ImagePull takes an image name and ensures that the image manifest or index to which it refers
// exists in local cache and, if not, pulls it from the registry and writes it locally. It should be
// efficient and only write missing blobs, based on their content hash.
// If the ref and all of the content for the requested
// architectures already exist in the cache, it will not pull anything, unless alwaysPull is set to true.
// If you call it multiple times, even with different architectures, the ref will continue to point to the same index.
// Only the underlying content will be added.
// However, do note that it *always* reaches out to the remote registry to check the content.
// If you just want to check the status of a local ref, use ValidateImage.
// Note that ImagePull does try ValidateImage first, so if the image is already in the cache, it will not
// do any network activity at all.
func (p *Provider) ImagePull(ref *reference.Spec, trustedRef, architecture string, alwaysPull bool) (lktspec.ImageSource, error) {
	imageName := util.ReferenceExpand(ref.String())
	canonicalRef, err := reference.Parse(imageName)
	if err != nil {
		return ImageSource{}, fmt.Errorf("invalid image name %s: %v", imageName, err)
	}
	ref = &canonicalRef
	image := ref.String()
	pullImageName := image
	remoteOptions := []remote.Option{remote.WithAuthFromKeychain(authn.DefaultKeychain)}
	if trustedRef != "" {
		pullImageName = trustedRef
	}
	log.Debugf("ImagePull to cache %s trusted reference %s", image, pullImageName)

	// unless alwaysPull is set to true, check locally first
	if alwaysPull {
		log.Printf("Instructed always to pull, so pulling image %s arch %s", image, architecture)
	} else {
		imgSrc, err := p.ValidateImage(ref, architecture)
		switch {
		case err == nil && imgSrc != nil:
			log.Printf("Image %s arch %s found in local cache, not pulling", image, architecture)
			return imgSrc, nil
		case err != nil && errors.Is(err, &noReferenceError{}):
			log.Printf("Image %s arch %s not found in local cache, pulling", image, architecture)
		default:
			log.Printf("Image %s arch %s incomplete or invalid in local cache, error %v, pulling", image, architecture, err)
		}
		// there was an error, so try to pull
	}
	remoteRef, err := name.ParseReference(pullImageName)
	if err != nil {
		return ImageSource{}, fmt.Errorf("invalid image name %s: %v", pullImageName, err)
	}

	desc, err := remote.Get(remoteRef, remoteOptions...)
	if err != nil {
		return ImageSource{}, fmt.Errorf("error getting manifest for trusted image %s: %v", pullImageName, err)
	}

	// use the original image name in the annotation
	annotations := map[string]string{
		imagespec.AnnotationRefName: image,
	}

	// first attempt as an index
	ii, err := desc.ImageIndex()
	if err == nil {
		log.Debugf("ImageWrite retrieved %s is index, saving, first checking if it contains target arch %s", pullImageName, architecture)
		im, err := ii.IndexManifest()
		if err != nil {
			return ImageSource{}, fmt.Errorf("unable to get IndexManifest: %v", err)
		}
		// only useful if it contains our architecture
		var foundArch bool
		for _, m := range im.Manifests {
			if m.MediaType.IsImage() && m.Platform != nil && m.Platform.Architecture == architecture && m.Platform.OS == linux {
				foundArch = true
				break
			}
		}
		if !foundArch {
			return ImageSource{}, fmt.Errorf("index %s does not contain target architecture %s", pullImageName, architecture)
		}

		if err := p.cache.WriteIndex(ii); err != nil {
			return ImageSource{}, fmt.Errorf("unable to write index: %v", err)
		}
		if _, err := p.DescriptorWrite(ref, desc.Descriptor); err != nil {
			return ImageSource{}, fmt.Errorf("unable to write index descriptor to cache: %v", err)
		}
	} else {
		var im v1.Image
		// try an image
		im, err = desc.Image()
		if err != nil {
			return ImageSource{}, fmt.Errorf("provided image is neither an image nor an index: %s", image)
		}
		log.Debugf("ImageWrite retrieved %s is image, saving", pullImageName)
		if err = p.cache.ReplaceImage(im, match.Name(image), layout.WithAnnotations(annotations)); err != nil {
			return ImageSource{}, fmt.Errorf("unable to save image to cache: %v", err)
		}
	}
	return p.NewSource(
		ref,
		architecture,
		&desc.Descriptor,
	), nil
}

// ImageLoad takes an OCI format image tar stream and writes it locally. It should be
// efficient and only write missing blobs, based on their content hash.
// Returns any descriptors that are in the tar stream's index.json as manifests.
// Does not try to resolve lower levels. Most such tar streams will have a single
// manifest in the index.json's manifests list, but it is possible to have more.
func (p *Provider) ImageLoad(r io.Reader) ([]v1.Descriptor, error) {
	var (
		tr    = tar.NewReader(r)
		index bytes.Buffer
	)
	log.Debugf("ImageWriteTar to cache")
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return nil, err
		}

		// get the filename and decide what to do with the file on that basis
		// there are only a few kinds of files in an oci archive:
		//   blobs/sha256/<hash>   - these we write out to our cache unless it already exists
		//   index.json            - we just take the data out of it and append to our index.json
		//   manifest.json         - not interested
		//   oci-layout            - not interested
		filename := header.Name
		switch {
		case filename == "manifest.json":
			log.Debugf("ignoring %s", filename)
		case filename == "oci-layout":
			log.Debugf("ignoring %s", filename)
		case header.Typeflag == tar.TypeDir:
			log.Debugf("ignoring directory %s", filename)
		case filename == "index.json":
			log.Debugf("saving %s to memory to parse", filename)
			// any errors should stop and get reported
			if _, err := io.Copy(&index, tr); err != nil {
				return nil, fmt.Errorf("error reading data for file %s : %v", filename, err)
			}
		case strings.HasPrefix(filename, "blobs/sha256/"):
			// must have a file named blob/sha256/<hash>
			parts := strings.Split(filename, "/")
			// if we had a file that is just the directory, ignore it
			if len(parts) != 3 {
				log.Debugf("ignoring %s", filename)
				continue
			}
			hash, err := v1.NewHash(fmt.Sprintf("%s:%s", parts[1], parts[2]))
			if err != nil {
				// malformed file
				return nil, fmt.Errorf("invalid hash filename for %s: %v", filename, err)
			}
			log.Debugf("writing %s as hash %s", filename, hash)
			if err := p.cache.WriteBlob(hash, io.NopCloser(tr)); err != nil {
				return nil, fmt.Errorf("error reading data for file %s : %v", filename, err)
			}
		}
	}
	// update the index in the cache directory
	var descs []v1.Descriptor
	if index.Len() != 0 {
		im, err := v1.ParseIndexManifest(&index)
		if err != nil {
			return nil, fmt.Errorf("error reading index.json")
		}
		// these manifests are in the root index.json of the tar stream
		// each of these is either an image or an index
		// either way, it gets added directly to the linuxkit cache index.
		for _, desc := range im.Manifests {
			if imgName, ok := desc.Annotations[images.AnnotationImageName]; ok {
				// remove the old descriptor, if it exists
				if err := p.cache.RemoveDescriptors(match.Name(imgName)); err != nil {
					return nil, fmt.Errorf("unable to remove old descriptors for %s: %v", imgName, err)
				}
				// save the image name under our proper annotation
				if desc.Annotations == nil {
					desc.Annotations = map[string]string{}
				}
				desc.Annotations[imagespec.AnnotationRefName] = imgName
			}
			log.Debugf("appending descriptor %#v", desc)
			if err := p.cache.AppendDescriptor(desc); err != nil {
				return nil, fmt.Errorf("error appending descriptor to layout index: %v", err)
			}
			descs = append(descs, desc)
		}
	}
	return descs, nil
}

// IndexWrite takes an image name and creates an index for the targets to which it points.
// does not pull down any images; entirely assumes that the subjects of the manifests are present.
// If a reference to the provided already exists and it is an index, updates the manifests in the
// existing index.
func (p *Provider) IndexWrite(ref *reference.Spec, descriptors ...v1.Descriptor) (lktspec.ImageSource, error) {
	image := ref.String()
	log.Debugf("writing an index for %s", image)
	if len(descriptors) < 1 {
		return ImageSource{}, errors.New("cannot create index without any manifests")
	}

	ii, err := p.cache.ImageIndex()
	if err != nil {
		return ImageSource{}, fmt.Errorf("unable to get root index: %v", err)
	}
	images, err := partial.FindImages(ii, match.Name(image))
	if err != nil {
		return ImageSource{}, fmt.Errorf("error parsing index: %v", err)
	}
	if err == nil && len(images) > 0 {
		return ImageSource{}, fmt.Errorf("image named %s already exists in cache and is not an index", image)
	}
	indexes, err := partial.FindIndexes(ii, match.Name(image))
	if err != nil {
		return ImageSource{}, fmt.Errorf("error parsing index: %v", err)
	}
	var im v1.IndexManifest
	// do we update an existing one? Or create a new one?
	if len(indexes) > 0 {
		// we already had one, so update just the referenced index and return
		manifest, err := indexes[0].IndexManifest()
		if err != nil {
			return ImageSource{}, fmt.Errorf("unable to convert index for %s into its manifest: %v", image, err)
		}
		oldhash, err := indexes[0].Digest()
		if err != nil {
			return ImageSource{}, fmt.Errorf("unable to get hash of existing index: %v", err)
		}
		// we only care about avoiding duplicate arch/OS/Variant
		var (
			descReplace    = map[string]v1.Descriptor{}
			descNonReplace []v1.Descriptor
		)
		for _, desc := range descriptors {
			// we do not replace "unknown" because those are attestations; we might remove attestations that point at things we remove
			if desc.Platform == nil || (desc.Platform.Architecture == "unknown" && desc.Platform.OS == "unknown") {
				descNonReplace = append(descNonReplace, desc)
				continue
			}
			descReplace[fmt.Sprintf("%s/%s/%s", desc.Platform.OS, desc.Platform.Architecture, desc.Platform.OSVersion)] = desc
		}
		// now we can go through each one and see if it already exists, and, if so, replace it
		// however, we do not replace attestations unless they point at something we are removing
		var (
			manifests         []v1.Descriptor
			referencedDigests = map[string]bool{}
		)
		for _, m := range manifest.Manifests {
			if m.Platform != nil {
				lookup := fmt.Sprintf("%s/%s/%s", m.Platform.OS, m.Platform.Architecture, m.Platform.OSVersion)
				if desc, ok := descReplace[lookup]; ok {
					manifests = append(manifests, desc)
					referencedDigests[desc.Digest.String()] = true
					// already added, so do not need it in the lookup list any more
					delete(descReplace, lookup)
					continue
				}
			}
			manifests = append(manifests, m)
			referencedDigests[m.Digest.String()] = true
		}
		// any left get added
		for _, desc := range descReplace {
			manifests = append(manifests, desc)
			referencedDigests[desc.Digest.String()] = true
		}
		for _, desc := range descNonReplace {
			manifests = append(manifests, desc)
			referencedDigests[desc.Digest.String()] = true
		}
		// before we complete, go through the manifests, and if any are attestations that point to something
		// no longer there, remove them
		// everything in the list already has its digest marked in the digests map, so we can just check that
		manifest.Manifests = []v1.Descriptor{}
		appliedManifests := map[v1.Hash]bool{}
		for _, m := range manifests {
			// we already added it; do not add it twice
			if _, ok := appliedManifests[m.Digest]; ok {
				continue
			}
			if len(m.Annotations) < 1 {
				manifest.Manifests = append(manifest.Manifests, m)
				appliedManifests[m.Digest] = true
				continue
			}
			value, ok := m.Annotations[lktutil.AnnotationDockerReferenceDigest]
			if !ok {
				manifest.Manifests = append(manifest.Manifests, m)
				appliedManifests[m.Digest] = true
				continue
			}
			if _, ok := referencedDigests[value]; ok {
				manifest.Manifests = append(manifest.Manifests, m)
				appliedManifests[m.Digest] = true
				continue
			}
			// if we got this far, we have an attestation that points to something no longer in the index,
			// do do not add it
		}

		im = *manifest
		// remove the old index
		if err := p.cache.RemoveBlob(oldhash); err != nil {
			return ImageSource{}, fmt.Errorf("unable to remove old index file: %v", err)
		}
	} else {
		// we did not have one, so create an index, store it, update the root index.json, and return
		im = v1.IndexManifest{
			MediaType:     types.OCIImageIndex,
			Manifests:     descriptors,
			SchemaVersion: 2,
		}
	}

	// write the updated index, remove the old one
	b, err := json.Marshal(im)
	if err != nil {
		return ImageSource{}, fmt.Errorf("unable to marshal new index to json: %v", err)
	}
	hash, size, err := v1.SHA256(bytes.NewReader(b))
	if err != nil {
		return ImageSource{}, fmt.Errorf("error calculating hash of index json: %v", err)
	}
	if err := p.cache.WriteBlob(hash, io.NopCloser(bytes.NewReader(b))); err != nil {
		return ImageSource{}, fmt.Errorf("error writing new index to json: %v", err)
	}
	// finally update the descriptor in the root
	if err := p.cache.RemoveDescriptors(match.Name(image)); err != nil {
		return ImageSource{}, fmt.Errorf("unable to remove old descriptor from index.json: %v", err)
	}
	desc := v1.Descriptor{
		MediaType: types.OCIImageIndex,
		Size:      size,
		Digest:    hash,
		Annotations: map[string]string{
			imagespec.AnnotationRefName: image,
		},
	}
	if err := p.cache.AppendDescriptor(desc); err != nil {
		return ImageSource{}, fmt.Errorf("unable to append new descriptor to index.json: %v", err)
	}

	return p.NewSource(
		ref,
		"",
		&desc,
	), nil
}

// DescriptorWrite writes a descriptor to the cache index; it validates that it has a name
// and replaces any existing one
func (p *Provider) DescriptorWrite(ref *reference.Spec, desc v1.Descriptor) (lktspec.ImageSource, error) {
	if ref == nil {
		return ImageSource{}, errors.New("cannot write descriptor without reference name")
	}
	image := ref.String()
	if desc.Annotations == nil {
		desc.Annotations = map[string]string{}
	}
	desc.Annotations[imagespec.AnnotationRefName] = image
	log.Debugf("writing descriptor for image %s", image)

	// do we update an existing one? Or create a new one?
	if err := p.cache.RemoveDescriptors(match.Name(image)); err != nil {
		return ImageSource{}, fmt.Errorf("unable to remove old descriptors for %s: %v", image, err)
	}

	if err := p.cache.AppendDescriptor(desc); err != nil {
		return ImageSource{}, fmt.Errorf("unable to append new descriptor for %s: %v", image, err)
	}

	return p.NewSource(
		ref,
		"",
		&desc,
	), nil
}

func (p *Provider) ImageInCache(ref *reference.Spec, trustedRef, architecture string) (bool, error) {
	img, err := p.findImage(ref.String(), architecture)
	if err != nil {
		return false, err
	}
	// findImage only checks if we had the pointer to it; it does not check if it is complete.
	// We need to do that next.

	// check that all of the layers exist
	layers, err := img.Layers()
	if err != nil {
		return false, fmt.Errorf("layers not found: %v", err)
	}
	for _, layer := range layers {
		dig, err := layer.Digest()
		if err != nil {
			return false, fmt.Errorf("unable to get digest of layer: %v", err)
		}
		var rc io.ReadCloser
		if rc, err = p.cache.Blob(dig); err != nil {
			return false, fmt.Errorf("layer %s not found: %v", dig, err)
		}
		rc.Close()
	}
	// check that the config exists
	config, err := img.ConfigName()
	if err != nil {
		return false, fmt.Errorf("unable to get config: %v", err)
	}
	var rc io.ReadCloser
	if rc, err = p.cache.Blob(config); err != nil {
		return false, fmt.Errorf("config %s not found: %v", config, err)
	}
	rc.Close()
	return true, nil
}

// ImageInRegistry takes an image name and checks that the image manifest or index to which it refers
// exists in the registry.
func (p *Provider) ImageInRegistry(ref *reference.Spec, trustedRef, architecture string) (bool, error) {
	image := ref.String()
	remoteOptions := []remote.Option{remote.WithAuthFromKeychain(authn.DefaultKeychain)}
	log.Debugf("Checking image %s in registry", image)

	remoteRef, err := name.ParseReference(image)
	if err != nil {
		return false, fmt.Errorf("invalid image name %s: %v", image, err)
	}

	desc, err := remote.Get(remoteRef, remoteOptions...)
	if err != nil {
		log.Debugf("Retrieving image %s returned an error, ignoring: %v", image, err)
		return false, nil
	}
	// first attempt as an index
	ii, err := desc.ImageIndex()
	if err == nil {
		log.Debugf("ImageExists retrieved %s as index", remoteRef)
		im, err := ii.IndexManifest()
		if err != nil {
			return false, fmt.Errorf("unable to get IndexManifest: %v", err)
		}
		for _, m := range im.Manifests {
			if m.MediaType.IsImage() && (m.Platform == nil || m.Platform.Architecture == architecture) {
				return true, nil
			}
		}
		// we went through all of the manifests and did not find one that matches the target architecture
	} else {
		var im v1.Image
		// try an image
		im, err = desc.Image()
		if err != nil {
			return false, fmt.Errorf("provided image is neither an image nor an index: %s", image)
		}
		log.Debugf("ImageExists retrieved %s as image", remoteRef)
		conf, err := im.ConfigFile()
		if err != nil {
			return false, fmt.Errorf("unable to get ConfigFile: %v", err)
		}
		if conf.Architecture == architecture {
			return true, nil
		}
		// the image had the wrong architecture
	}
	return false, nil
}
