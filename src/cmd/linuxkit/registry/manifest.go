package registry

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	dockertypes "github.com/docker/docker/api/types"
	"github.com/estesp/manifest-tool/pkg/registry"
	"github.com/estesp/manifest-tool/pkg/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
)

const (
	notaryServer                     = "https://notary.docker.io"
	notaryDelegationPassphraseEnvVar = "NOTARY_DELEGATION_PASSPHRASE"
	notaryAuthEnvVar                 = "NOTARY_AUTH"
	dctEnvVar                        = "DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE"
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

// SignTag sign a tag on a registry.
func SignTag(img, digest string, length int, auth dockertypes.AuthConfig) error {
	imgParts := strings.Split(img, ":")
	if len(imgParts) < 2 {
		return fmt.Errorf("image not composed of <repo>:<tag> '%s'", img)
	}
	repo := imgParts[0]
	tag := imgParts[1]

	digestParts := strings.Split(digest, ":")
	if len(digestParts) < 2 {
		return fmt.Errorf("digest not composed of <algo>:<hash> '%s'", digest)
	}
	algo, hash := digestParts[0], digestParts[1]
	if algo != "sha256" {
		return fmt.Errorf("notary works with sha256 hash, not the provided %s", algo)
	}

	notaryAuth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", auth.Username, auth.Password)))
	// run the notary command to sign
	args := []string{
		"-s",
		notaryServer,
		"-d",
		path.Join(os.Getenv("HOME"), ".docker/trust"),
		"addhash",
		"-p",
		fmt.Sprintf("docker.io/%s", repo),
		tag,
		strconv.Itoa(length),
		"--sha256",
		hash,
		"-r",
		"targets/releases",
	}
	cmd := exec.Command("notary", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", notaryDelegationPassphraseEnvVar, os.Getenv(dctEnvVar)), fmt.Sprintf("%s=%s", notaryAuthEnvVar, notaryAuth))
	log.Debugf("Executing: %v", cmd.Args)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute notary-tool: %v", err)
	}

	// report output
	fmt.Printf("Signed manifest index: %s:%s\n", repo, tag)

	return nil
}
