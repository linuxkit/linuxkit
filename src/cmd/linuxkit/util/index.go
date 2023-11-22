package util

import (
	"fmt"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
)

const (
	AnnotationDockerReferenceDigest = "vnd.docker.reference.digest"
	AnnotationDockerReferenceType   = "vnd.docker.reference.type"
	AnnotationAttestationManifest   = "attestation-manifest"
	AnnotationInTotoPredicateType   = "in-toto.io/predicate-type"
	AnnotationSPDXDoc               = "https://spdx.dev/Document"
)

// AppendIndex creates a new ImageIndex composed of the merge of the elements of secondary ImageIndex
// and primary ImageIndex.
// In the case of conflicts, the primary ImageIndex wins.
// For example, if both have a manifest for a specific platform, then use the one from primary.
// The append is aware of the buildkit-style attestations, and will keep any attestations that point to a valid
// manifest in the list, discarding any that do not.
// If either of the two ImageIndexes is nil, the other is returned.
// If the secondary ImageIndex is a precise superset of primary,
// then the returned new ImageIndex is an identical copy of the secondary ImageIndex.
func AppendIndex(primary, secondary v1.ImageIndex) (v1.ImageIndex, error) {
	primaryManifest, err := primary.IndexManifest()
	if err != nil {
		return nil, err
	}
	secondaryManifest, err := secondary.IndexManifest()
	if err != nil {
		return nil, err
	}
	// figure out what already is in the index, and what should be overwritten
	// what should be checked in the existing index:
	// 1. platform - if it is in remote index but not in local, add to local
	// 2. attestation - after all platforms, does it point to something in the updated index?
	//      If not, remove

	if indexSuperset(primaryManifest, secondaryManifest) {
		return secondary, nil
	}
	// make a map of all the digests already in the index, so we can know what is there
	var (
		manifestMap = map[v1.Hash]bool{}
		platformMap = map[string]bool{}
		adds        []mutate.IndexAddendum
	)
	for _, m := range primaryManifest.Manifests {
		if m.Platform == nil || m.Platform.Architecture == "" {
			continue
		}
		platformKey := fmt.Sprintf("%s/%s/%s", m.Platform.Architecture, m.Platform.OS, m.Platform.Variant)
		manifestMap[m.Digest] = true
		platformMap[platformKey] = true
	}

	for _, m := range secondaryManifest.Manifests {
		// ignore any of those without a platform for this run (we will deal witb attestations in a second pass)
		if m.Platform == nil || m.Platform.Architecture == "" || (m.Platform.Architecture == "unknown" && m.Platform.OS == "unknown") {
			continue
		}
		platformKey := fmt.Sprintf("%s/%s/%s", m.Platform.Architecture, m.Platform.OS, m.Platform.Variant)
		// primary wins if we already have this platform covered
		if _, ok := platformMap[platformKey]; ok {
			continue
		}
		if _, ok := manifestMap[m.Digest]; ok {
			// we already have this one, so we can skip it
			continue
		}
		img, err := secondary.Image(m.Digest)
		if err != nil {
			return nil, fmt.Errorf("could not get image %s: %v", m.Digest, err)
		}
		adds = append(adds, mutate.IndexAddendum{
			Add:        img,
			Descriptor: m,
		})
		// want to add manifest "m"
		manifestMap[m.Digest] = true
	}

	// now we have assured that all of the images in the remote index are in the local index
	// or overridden by matching local ones
	// next we have to make sure that any sboms already on the remote index are still valid
	// we either add them to the local index, or remove them if they are no longer valid
	// we assume the ones in the local index are valid because they would have been generated now
	for _, m := range secondaryManifest.Manifests {
		if m.Platform == nil || m.Platform.Architecture != "unknown" || m.Platform.OS != "unknown" || m.Annotations == nil || m.Annotations[AnnotationDockerReferenceDigest] == "" {
			continue
		}
		// if we already have this one, we are good
		if _, ok := manifestMap[m.Digest]; ok {
			continue
		}
		// the hash to which this attestation points
		hash := m.Annotations[AnnotationDockerReferenceDigest]
		dig, err := v1.NewHash(hash)
		if err != nil {
			return nil, fmt.Errorf("could not parse hash %s: %v", hash, err)
		}
		// if this points at something not in the local index, do not bother adding it
		if _, ok := manifestMap[dig]; !ok {
			continue
		}
		img, err := secondary.Image(m.Digest)
		if err != nil {
			return nil, fmt.Errorf("could not get image %s: %v", m.Digest, err)
		}
		adds = append(adds, mutate.IndexAddendum{
			Add:        img,
			Descriptor: m,
		})
		// want to add manifest "m"
		manifestMap[m.Digest] = true
	}
	ret := mutate.AppendManifests(primary, adds...)

	return ret, nil
}

func indexSuperset(inner, outer *v1.IndexManifest) bool {
	var manifestMap = map[v1.Hash]bool{}
	if len(inner.Manifests) > len(outer.Manifests) {
		return false
	}
	for _, m := range outer.Manifests {
		manifestMap[m.Digest] = true
	}
	for _, m := range inner.Manifests {
		if _, ok := manifestMap[m.Digest]; !ok {
			return false
		}
	}
	return true
}
