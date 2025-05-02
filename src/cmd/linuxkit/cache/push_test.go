package cache

import (
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/mutate"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"github.com/google/go-containerregistry/pkg/v1/types"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"

	"github.com/stretchr/testify/require"
)

func TestCleanDanglingReferences(t *testing.T) {
	// Create some base images to use for digest references
	baseImg1, err := random.Image(1024, 1)
	require.NoError(t, err)
	baseImg1Digest, err := baseImg1.Digest()
	require.NoError(t, err)

	baseImg2, err := random.Image(1024, 1)
	require.NoError(t, err)
	baseImg2Digest, err := baseImg2.Digest()
	require.NoError(t, err)

	// Create valid platform manifests
	platformManifest1 := v1.Descriptor{
		MediaType: types.OCIManifestSchema1,
		Digest:    baseImg1Digest,
		Platform: &v1.Platform{
			Architecture: "amd64",
			OS:           "linux",
		},
	}

	platformManifest2 := v1.Descriptor{
		MediaType: types.OCIManifestSchema1,
		Digest:    baseImg2Digest,
		Platform: &v1.Platform{
			Architecture: "arm64",
			OS:           "linux",
		},
	}

	// create an index with just those manifests

	// Build index
	index := mutate.AppendManifests(empty.Index,
		mutate.IndexAddendum{Add: baseImg1, Descriptor: platformManifest1},
		mutate.IndexAddendum{Add: baseImg2, Descriptor: platformManifest2},
	)

	// Call the function
	cleaned, err := cleanDanglingReferences(index)
	require.NoError(t, err)

	// Get resulting manifest list
	manifest, err := cleaned.IndexManifest()
	require.NoError(t, err)

	// Check that the same 2 manifests remain: platformManifest1 and platformManifest2
	require.Len(t, manifest.Manifests, 2)

	// next add sbom references to the index
	sbom1, err := random.Image(1024, 1)
	require.NoError(t, err)
	sbom1Digest, err := sbom1.Digest()
	require.NoError(t, err)

	sbom2, err := random.Image(1024, 1)
	require.NoError(t, err)
	sbom2Digest, err := sbom2.Digest()
	require.NoError(t, err)

	// Create valid sbom manifests
	sbomManifest1 := v1.Descriptor{
		MediaType: types.OCIManifestSchema1,
		Digest:    sbom1Digest,
		Platform: &v1.Platform{
			Architecture: unknown,
			OS:           unknown,
		},
		Annotations: map[string]string{
			util.AnnotationDockerReferenceDigest: baseImg1Digest.String(),
			util.AnnotationDockerReferenceType:   util.AnnotationAttestationManifest,
		},
	}

	sbomManifest2 := v1.Descriptor{
		MediaType: types.OCIManifestSchema1,
		Digest:    sbom2Digest,
		Platform: &v1.Platform{
			Architecture: unknown,
			OS:           unknown,
		},
		Annotations: map[string]string{
			util.AnnotationDockerReferenceDigest: baseImg2Digest.String(),
			util.AnnotationDockerReferenceType:   util.AnnotationAttestationManifest,
		},
	}

	index = mutate.AppendManifests(index,
		mutate.IndexAddendum{Add: sbom1, Descriptor: sbomManifest1},
		mutate.IndexAddendum{Add: sbom2, Descriptor: sbomManifest2},
	)

	// Call the function
	cleaned, err = cleanDanglingReferences(index)
	require.NoError(t, err)

	// Get resulting manifest list
	manifest, err = cleaned.IndexManifest()
	require.NoError(t, err)

	// Check that the same 4 manifests remain: 2*image and 2*sbom
	require.Len(t, manifest.Manifests, 4)

	// finally add a dangling reference to the index
	baseImg3, err := random.Image(1024, 1)
	require.NoError(t, err)
	baseImg3Digest, err := baseImg3.Digest()
	require.NoError(t, err)

	danglingSbom, err := random.Image(1024, 1)
	require.NoError(t, err)
	danglingDigest, err := danglingSbom.Digest()
	require.NoError(t, err)

	// Create valid sbom manifests
	danglingManifest := v1.Descriptor{
		MediaType: types.OCIManifestSchema1,
		Digest:    danglingDigest,
		Platform: &v1.Platform{
			Architecture: unknown,
			OS:           unknown,
		},
		Annotations: map[string]string{
			util.AnnotationDockerReferenceDigest: baseImg3Digest.String(),
			util.AnnotationDockerReferenceType:   util.AnnotationAttestationManifest,
		},
	}

	// we add the dangling reference sbom but *not* the image it points to
	// which should cause it to be removed
	index = mutate.AppendManifests(index,
		mutate.IndexAddendum{Add: danglingSbom, Descriptor: danglingManifest},
	)
	// Call the function
	cleaned, err = cleanDanglingReferences(index)
	require.NoError(t, err)
	// Get resulting manifest list
	manifest, err = cleaned.IndexManifest()
	require.NoError(t, err)
	// Check that the same 4 manifests remain: 2*image and 2*sbom
	require.Len(t, manifest.Manifests, 4)
	// Check that the dangling reference has been removed
	var foundDangling bool
	for _, m := range manifest.Manifests {
		if m.Digest.String() == danglingDigest.String() {
			foundDangling = true
		}
	}
	require.False(t, foundDangling, "dangling reference should have been removed")
}
