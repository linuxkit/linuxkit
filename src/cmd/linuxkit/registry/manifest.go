package registry

import (
	"fmt"
	"strings"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/estesp/manifest-tool/v2/pkg/registry"
	"github.com/estesp/manifest-tool/v2/pkg/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
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
func PushManifest(img string, auth dockertypes.AuthConfig) (hash string, length int, err error) {
	srcImages := []types.ManifestEntry{}

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
		srcImages = append(srcImages, types.ManifestEntry{
			Image: fmt.Sprintf("%s-%s", img, arch),
			Platform: ocispec.Platform{
				OS:           os,
				Architecture: arch,
				Variant:      variant,
			},
		})
	}

	yamlInput := types.YAMLInput{
		Image:     img,
		Manifests: srcImages,
	}

	log.Debugf("pushing manifest list for %s -> %#v", img, yamlInput)

	// push the manifest list with the auth as given, ignore missing, do not allow insecure
	return registry.PushManifestList(auth.Username, auth.Password, yamlInput, true, false, false, types.OCI, "")
}
