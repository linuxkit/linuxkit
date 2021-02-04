package cache

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/containerd/containerd/reference"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/match"
	"github.com/google/go-containerregistry/pkg/v1/partial"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
)

const (
	linux = "linux"
)

// ImageWrite takes an image name and pulls it down, writing it locally. It should be
// efficient and only write missing blobs, based on their content hash.
func ImageWrite(dir string, ref *reference.Spec, trustedRef, architecture string) (ImageSource, error) {
	p, err := Get(dir)
	if err != nil {
		return ImageSource{}, err
	}
	image := ref.String()
	pullImageName := image
	remoteOptions := []remote.Option{remote.WithAuthFromKeychain(authn.DefaultKeychain)}
	if trustedRef != "" {
		pullImageName = trustedRef
	}
	log.Debugf("ImageWrite to cache %s trusted reference %s", image, pullImageName)
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
		log.Debugf("ImageWrite retrieved %s is index, saving", pullImageName)
		err = p.ReplaceIndex(ii, match.Name(image), layout.WithAnnotations(annotations))
	} else {
		var im v1.Image
		// try an image
		im, err = desc.Image()
		if err != nil {
			return ImageSource{}, fmt.Errorf("provided image is neither an image nor an index: %s", image)
		}
		log.Debugf("ImageWrite retrieved %s is image, saving", pullImageName)
		err = p.ReplaceImage(im, match.Name(image), layout.WithAnnotations(annotations))
	}
	if err != nil {
		return ImageSource{}, fmt.Errorf("unable to save image to cache: %v", err)
	}
	return NewSource(
		ref,
		dir,
		architecture,
		&desc.Descriptor,
	), nil
}

// ImageWriteTar takes an OCI format image tar stream and writes it locally. It should be
// efficient and only write missing blobs, based on their content hash.
func ImageWriteTar(dir string, ref *reference.Spec, architecture string, r io.Reader) (ImageSource, error) {
	p, err := Get(dir)
	if err != nil {
		return ImageSource{}, err
	}

	var (
		tr    = tar.NewReader(r)
		index bytes.Buffer
	)
	var suffix string
	switch architecture {
	case "amd64", "arm64", "s390x":
		suffix = "-" + architecture
	default:
		return ImageSource{}, fmt.Errorf("Unknown arch %q", architecture)
	}
	imageName := ref.String() + suffix
	log.Debugf("ImageWriteTar to cache %s", imageName)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return ImageSource{}, err
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
				return ImageSource{}, fmt.Errorf("error reading data for file %s : %v", filename, err)
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
				return ImageSource{}, fmt.Errorf("invalid hash filename for %s: %v", filename, err)
			}
			log.Debugf("writing %s as hash %s", filename, hash)
			if err := p.WriteBlob(hash, ioutil.NopCloser(tr)); err != nil {
				return ImageSource{}, fmt.Errorf("error reading data for file %s : %v", filename, err)
			}
		}
	}
	// update the index in the cache directory
	var descriptor *v1.Descriptor
	if index.Len() != 0 {
		im, err := v1.ParseIndexManifest(&index)
		if err != nil {
			return ImageSource{}, fmt.Errorf("error reading index.json")
		}
		// in theory, we should support a tar stream with multiple images in it. However, how would we
		// know which one gets the single name annotation we have? We will find some way in the future.
		if len(im.Manifests) != 1 {
			return ImageSource{}, fmt.Errorf("currently only support OCI tar stream that has a single image")
		}
		if err := p.RemoveDescriptors(match.Name(imageName)); err != nil {
			return ImageSource{}, fmt.Errorf("unable to remove old descriptors for %s: %v", imageName, err)
		}
		for _, desc := range im.Manifests {
			// make sure that we have the correct image name annotation
			if desc.Annotations == nil {
				desc.Annotations = map[string]string{}
			}
			desc.Annotations[imagespec.AnnotationRefName] = imageName
			descriptor = &desc

			log.Debugf("appending descriptor %#v", descriptor)
			if err := p.AppendDescriptor(desc); err != nil {
				return ImageSource{}, fmt.Errorf("error appending descriptor to layout index: %v", err)
			}
		}
	}
	if descriptor != nil && descriptor.Platform == nil {
		descriptor.Platform = &v1.Platform{
			OS:           linux,
			Architecture: architecture,
		}
	}
	return NewSource(
		ref,
		dir,
		architecture,
		descriptor,
	), nil
}

// IndexWrite takes an image name and creates an index for the targets to which it points.
// does not pull down any images; entirely assumes that the subjects of the manifests are present.
// If a reference to the provided already exists and it is an index, updates the manifests in the
// existing index.
func IndexWrite(dir string, ref *reference.Spec, descriptors ...v1.Descriptor) (ImageSource, error) {
	p, err := Get(dir)
	if err != nil {
		return ImageSource{}, err
	}
	image := ref.String()
	log.Debugf("writing an index for %s", image)

	ii, err := p.ImageIndex()
	if err != nil {
		return ImageSource{}, fmt.Errorf("unable to get root index at %s: %v", dir, err)
	}
	images, err := partial.FindImages(ii, match.Name(image))
	if err != nil {
		return ImageSource{}, fmt.Errorf("error parsing index at %s: %v", dir, err)
	}
	if err == nil && len(images) > 0 {
		return ImageSource{}, fmt.Errorf("image named %s already exists in cache at %s and is not an index", image, dir)
	}
	indexes, err := partial.FindIndexes(ii, match.Name(image))
	if err != nil {
		return ImageSource{}, fmt.Errorf("error parsing index at %s: %v", dir, err)
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
		descReplace := map[string]v1.Descriptor{}
		for _, desc := range descriptors {
			descReplace[fmt.Sprintf("%s/%s/%s", desc.Platform.OS, desc.Platform.Architecture, desc.Platform.OSVersion)] = desc
		}
		// now we can go through each one and see if it already exists, and, if so, replace it
		var manifests []v1.Descriptor
		for _, m := range manifest.Manifests {
			if m.Platform != nil {
				lookup := fmt.Sprintf("%s/%s/%s", m.Platform.OS, m.Platform.Architecture, m.Platform.OSVersion)
				if desc, ok := descReplace[lookup]; ok {
					manifests = append(manifests, desc)
					// already added, so do not need it in the lookup list any more
					delete(descReplace, lookup)
					continue
				}
			}
			manifests = append(manifests, m)
		}
		// any left get added
		for _, desc := range descReplace {
			manifests = append(manifests, desc)
		}
		manifest.Manifests = manifests
		im = *manifest
		// remove the old index
		if err := p.RemoveBlob(oldhash); err != nil {
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
	if err := p.WriteBlob(hash, ioutil.NopCloser(bytes.NewReader(b))); err != nil {
		return ImageSource{}, fmt.Errorf("error writing new index to json: %v", err)
	}
	// finally update the descriptor in the root
	if err := p.RemoveDescriptors(match.Name(image)); err != nil {
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
	if err := p.AppendDescriptor(desc); err != nil {
		return ImageSource{}, fmt.Errorf("unable to append new descriptor to index.json: %v", err)
	}

	return NewSource(
		ref,
		dir,
		"",
		&desc,
	), nil
}

// DescriptorWrite writes a name for a given descriptor
func DescriptorWrite(dir string, ref *reference.Spec, descriptors ...v1.Descriptor) (ImageSource, error) {
	p, err := Get(dir)
	if err != nil {
		return ImageSource{}, err
	}
	image := ref.String()
	log.Debugf("writing descriptors for image %s: %v", image, descriptors)

	ii, err := p.ImageIndex()
	if err != nil {
		return ImageSource{}, fmt.Errorf("unable to get root index at %s: %v", dir, err)
	}
	images, err := partial.FindImages(ii, match.Name(image))
	if err != nil {
		return ImageSource{}, fmt.Errorf("error parsing index at %s: %v", dir, err)
	}
	if err == nil && len(images) > 0 {
		return ImageSource{}, fmt.Errorf("image named %s already exists in cache at %s and is not an index", image, dir)
	}
	indexes, err := partial.FindIndexes(ii, match.Name(image))
	if err != nil {
		return ImageSource{}, fmt.Errorf("error parsing index at %s: %v", dir, err)
	}
	var im v1.IndexManifest
	// do we update an existing one? Or create a new one?
	if len(indexes) > 0 {
		// we already had one, so update just the referenced index and return
		im, err := indexes[0].IndexManifest()
		if err != nil {
			return ImageSource{}, fmt.Errorf("unable to convert index for %s into its manifest: %v", image, err)
		}
		oldhash, err := indexes[0].Digest()
		if err != nil {
			return ImageSource{}, fmt.Errorf("unable to get hash of existing index: %v", err)
		}
		// we only care about avoiding duplicate arch/OS/Variant
		descReplace := map[string]v1.Descriptor{}
		for _, desc := range descriptors {
			descReplace[fmt.Sprintf("%s/%s/%s", desc.Platform.OS, desc.Platform.Architecture, desc.Platform.OSVersion)] = desc
		}
		// now we can go through each one and see if it already exists, and, if so, replace it
		var manifests []v1.Descriptor
		for _, m := range im.Manifests {
			lookup := fmt.Sprintf("%s/%s/%s", m.Platform.OS, m.Platform.Architecture, m.Platform.OSVersion)
			if desc, ok := descReplace[lookup]; ok {
				manifests = append(manifests, desc)
				// already added, so do not need it in the lookup list any more
				delete(descReplace, lookup)
				continue
			}
			manifests = append(manifests, m)
		}
		// any left get added
		for _, desc := range descReplace {
			manifests = append(manifests, desc)
		}
		im.Manifests = manifests

		// remove the old index - unfortunately, there is no "RemoveBlob" option in the library
		// once https://github.com/google/go-containerregistry/pull/936/ is in, we can get rid of some of this
		oldfile := path.Join(dir, oldhash.Algorithm, oldhash.Hex)
		if err := os.RemoveAll(oldfile); err != nil {
			return ImageSource{}, fmt.Errorf("unable to remove old file %s: %v", oldfile, err)
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
	if err := p.WriteBlob(hash, ioutil.NopCloser(bytes.NewReader(b))); err != nil {
		return ImageSource{}, fmt.Errorf("error writing new index to json: %v", err)
	}
	// finally update the descriptor in the root
	if err := p.RemoveDescriptors(match.Name(image)); err != nil {
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
	if err := p.AppendDescriptor(desc); err != nil {
		return ImageSource{}, fmt.Errorf("unable to append new descriptor to index.json: %v", err)
	}

	return NewSource(
		ref,
		dir,
		"",
		&desc,
	), nil
}
