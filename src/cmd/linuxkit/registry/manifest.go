package registry

import (
	"fmt"
	"strings"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/estesp/manifest-tool/pkg/registry"
	"github.com/estesp/manifest-tool/pkg/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
)

var platforms = []string{
	"linux/amd64", "linux/arm64", "linux/s390x",
}

// PushManifest create a manifest that supports each of the provided platforms and push it out.
func PushManifest(img string, auth dockertypes.AuthConfig) (hash string, length int, err error) {
	srcImages := []types.ManifestEntry{}

	for i, platform := range platforms {
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
	return registry.PushManifestList(auth.Username, auth.Password, yamlInput, true, false, false, "")
}
