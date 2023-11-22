package registry

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	log "github.com/sirupsen/logrus"
)

// these platforms are used only as the source for registry.PushManifestList(). Since it is
// configured to ignore missing, we could include dozens if we wanted; it doesn't hurt, adding maybe a few seconds to
// the whole run.
// Ideally, we could just look for all tags that start with linuxkit/foo:<hash>-*, but the registry API
// only supports "list all the tags" and "get a specific tag", no "get by pattern". The "get a specific tag"
// is exactly what registry.PushManifestList() uses, so no benefit to use doing that in advance,
// while "list all tags" is slow, and has to cycle through all of the (growing numbers of) tags
// before we know what exists. Might as well leave it as is.
var platformsToSearchForIndex = []string{
	"linux/amd64", "linux/arm64", "linux/s390x", "linux/riscv64", "linux/ppc64le",
}

// PushManifest create a manifest that supports each of the provided platforms and push it out.
func PushManifest(img string, options ...remote.Option) (hash string, length int64, err error) {
	baseRef, err := name.ParseReference(img)
	if err != nil {
		return hash, length, fmt.Errorf("parsing %s: %w", img, err)
	}

	adds := make([]mutate.IndexAddendum, 0, len(platformsToSearchForIndex))
	for i, platform := range platformsToSearchForIndex {
		osArchArr := strings.Split(platform, "/")
		if len(osArchArr) != 2 && len(osArchArr) != 3 {
			return hash, length, fmt.Errorf("platform argument %d is not of form 'os/arch': '%s'", i, platform)
		}
		variant := ""
		os, arch := osArchArr[0], osArchArr[1]
		if len(osArchArr) == 3 {
			variant = osArchArr[2]
		}
		refName := fmt.Sprintf("%s-%s", img, arch)
		ref, err := name.ParseReference(refName)
		if err != nil {
			return hash, length, fmt.Errorf("parsing %s: %w", refName, err)
		}
		remoteDesc, err := remote.Get(ref, options...)
		if err != nil {
			// TODO: Should distinguish between a 404 and a network error
			log.Warnf("image %s not found; skipping: %v", ref, err)
			continue
		}
		img, err := remoteDesc.Image()
		if err != nil {
			return hash, length, fmt.Errorf("getting image %s: %w", ref, err)
		}
		desc := remoteDesc.Descriptor
		desc.Platform = &v1.Platform{
			OS:           os,
			Architecture: arch,
			Variant:      variant,
		}
		adds = append(adds, mutate.IndexAddendum{
			Add:        img,
			Descriptor: desc,
		})
	}

	// add the desc to the index we will push
	index := mutate.AppendManifests(empty.Index, adds...)
	// base index with which we are working
	// get the existing index, if any
	desc, err := remote.Get(baseRef, options...)
	if err == nil && desc != nil {
		ii, err := desc.ImageIndex()
		if err != nil {
			return hash, length, fmt.Errorf("could not get index for existing reference %s: %w", img, err)
		}
		index, err = util.AppendIndex(index, ii)
		if err != nil {
			return hash, length, fmt.Errorf("could not append existing index for %s: %w", img, err)
		}
	}

	size, err := index.Size()
	if err != nil {
		return hash, length, fmt.Errorf("getting index size: %w", err)
	}
	dig, err := index.Digest()
	if err != nil {
		return hash, length, fmt.Errorf("getting index digest: %w", err)
	}
	// if it is unchanged, do nothing
	if desc != nil && desc.Digest == dig {
		log.Debugf("not pushing manifest list for %s, unchanged", img)
		return dig.String(), size, nil
	}
	log.Debugf("pushing manifest list for %s -> %#v", img, index)
	err = remote.WriteIndex(baseRef, index, options...)
	if err != nil {
		return hash, length, fmt.Errorf("writing index: %w", err)
	}
	return dig.String(), size, nil
}
