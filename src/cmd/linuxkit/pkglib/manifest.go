package pkglib

// manifest utilities

//go:generate ./gen

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
	auth "github.com/deislabs/oras/pkg/auth/docker"
	"github.com/docker/distribution/reference"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/estesp/manifest-tool/pkg/registry"
	"github.com/estesp/manifest-tool/pkg/store"
	"github.com/estesp/manifest-tool/pkg/types"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
)

/*
 EVERYTHING below here is because github.com/estesp/manifest-tool moved pushManifestList into
 a non-exported func and passed it cli. If/when it moves back, we can get rid of all of it.
 This code is copied almost verbatim from github.com/estesp/manifest-tool, mostly in
 push.go and util.go. It then was modified to remove any command-line dependencies.
*/

func pushManifestList(auth dockertypes.AuthConfig, input types.YAMLInput, ignoreMissing, insecure, plainHttp bool, configDir string) (hash string, length int, err error) {
	// resolve the target image reference for the combined manifest list/index
	targetRef, err := reference.ParseNormalizedNamed(input.Image)
	if err != nil {
		return hash, length, fmt.Errorf("Error parsing name for manifest list (%s): %v", input.Image, err)
	}

	var configDirs []string
	if configDir != "" {
		configDirs = append(configDirs, filepath.Join(configDir, "config.json"))
	}
	resolver := newResolver(auth.Username, auth.Password, insecure,
		plainHttp, configDirs...)

	imageType := types.Docker
	manifestList := types.ManifestList{
		Name:      input.Image,
		Reference: targetRef,
		Resolver:  resolver,
		Type:      imageType,
	}
	// create an in-memory store for OCI descriptors and content used during the push operation
	memoryStore := store.NewMemoryStore()

	log.Info("Retrieving digests of member images")
	for _, img := range input.Manifests {
		ref, err := parseName(img.Image)
		if err != nil {
			return hash, length, fmt.Errorf("Unable to parse image reference: %s: %v", img.Image, err)
		}
		if reference.Domain(targetRef) != reference.Domain(ref) {
			return hash, length, fmt.Errorf("Cannot use source images from a different registry than the target image: %s != %s", reference.Domain(ref), reference.Domain(targetRef))
		}
		descriptor, err := fetchDescriptor(resolver, memoryStore, ref)
		if err != nil {
			if ignoreMissing {
				log.Warnf("Couldn't access image '%q'. Skipping due to 'ignore missing' configuration.", img.Image)
				continue
			}
			return hash, length, fmt.Errorf("Inspect of image %q failed with error: %v", img.Image, err)
		}

		// Check that only member images of type OCI manifest or Docker v2.2 manifest are included
		switch descriptor.MediaType {
		case ocispec.MediaTypeImageIndex, types.MediaTypeDockerSchema2ManifestList:
			return hash, length, fmt.Errorf("Cannot include an image in a manifest list/index which is already a multi-platform image: %s", img.Image)
		case ocispec.MediaTypeImageManifest, types.MediaTypeDockerSchema2Manifest:
			// valid image type to include
		default:
			return hash, length, fmt.Errorf("Cannot include unknown media type '%s' in a manifest list/index push", descriptor.MediaType)
		}
		_, db, _ := memoryStore.Get(descriptor)
		var man ocispec.Manifest
		if err := json.Unmarshal(db, &man); err != nil {
			return hash, length, fmt.Errorf("Could not unmarshal manifest object from descriptor for image '%s': %v", img.Image, err)
		}
		_, cb, _ := memoryStore.Get(man.Config)
		var imgConfig types.Image
		if err := json.Unmarshal(cb, &imgConfig); err != nil {
			return hash, length, fmt.Errorf("Could not unmarshal config object from descriptor for image '%s': %v", img.Image, err)
		}
		// set labels for handling distribution source to get automatic cross-repo blob mounting for the layers
		info, _ := memoryStore.Info(context.TODO(), descriptor.Digest)
		for _, layer := range man.Layers {
			info.Digest = layer.Digest
			if _, err := memoryStore.Update(context.TODO(), info, ""); err != nil {
				log.Warnf("couldn't update in-memory store labels for %v: %v", info.Digest, err)
			}
		}

		// finalize the platform object that will be used to push with this manifest
		descriptor.Platform, err = resolvePlatform(descriptor, img, imgConfig)
		if err != nil {
			return hash, length, fmt.Errorf("Unable to create platform object for manifest %s: %v", descriptor.Digest.String(), err)
		}
		manifest := types.Manifest{
			Descriptor: descriptor,
			PushRef:    false,
		}

		if reference.Path(ref) != reference.Path(targetRef) {
			// the target manifest list/index is located in a different repo; need to push
			// the manifest as a digest to the target repo before the list/index is pushed
			manifest.PushRef = true
		}
		manifestList.Manifests = append(manifestList.Manifests, manifest)
	}

	if ignoreMissing && len(manifestList.Manifests) == 0 {
		// we need to verify we at least have one valid entry in the list
		// otherwise our manifest list will be totally empty
		return hash, length, fmt.Errorf("all entries were skipped due to missing source image references; no manifest list to push")
	}

	return registry.Push(manifestList, input.Tags, memoryStore)
}

func resolvePlatform(descriptor ocispec.Descriptor, img types.ManifestEntry, imgConfig types.Image) (*ocispec.Platform, error) {
	platform := &img.Platform
	if platform == nil {
		platform = &ocispec.Platform{}
	}
	// fill os/arch from inspected image if not specified in input YAML
	if img.Platform.OS == "" && img.Platform.Architecture == "" {
		// prefer a full platform object, if one is already available (and appears to have meaningful content)
		if descriptor.Platform.OS != "" || descriptor.Platform.Architecture != "" {
			platform = descriptor.Platform
		} else if imgConfig.OS != "" || imgConfig.Architecture != "" {
			platform.OS = imgConfig.OS
			platform.Architecture = imgConfig.Architecture
		}
	}
	// Windows: if the origin image has OSFeature and/or OSVersion information, and
	// these values were not specified in the creation YAML, then
	// retain the origin values in the Platform definition for the manifest list:
	if imgConfig.OSVersion != "" && img.Platform.OSVersion == "" {
		platform.OSVersion = imgConfig.OSVersion
	}
	if len(imgConfig.OSFeatures) > 0 && len(img.Platform.OSFeatures) == 0 {
		platform.OSFeatures = imgConfig.OSFeatures
	}

	// validate os/arch input
	if !isValidOSArch(platform.OS, platform.Architecture, platform.Variant) {
		return nil, fmt.Errorf("Manifest entry for image %s has unsupported os/arch or os/arch/variant combination: %s/%s/%s", img.Image, platform.OS, platform.Architecture, platform.Variant)
	}
	return platform, nil
}

func isValidOSArch(os string, arch string, variant string) bool {
	osarch := fmt.Sprintf("%s/%s", os, arch)

	if variant != "" {
		osarch = fmt.Sprintf("%s/%s/%s", os, arch, variant)
	}

	_, ok := validOSArch[osarch]
	return ok
}

// list of valid os/arch values (see "Optional Environment Variables" section
// of https://golang.org/doc/install/source
var validOSArch = map[string]bool{
	"darwin/386":      true,
	"darwin/amd64":    true,
	"darwin/arm":      true,
	"darwin/arm64":    true,
	"dragonfly/amd64": true,
	"freebsd/386":     true,
	"freebsd/amd64":   true,
	"freebsd/arm":     true,
	"linux/386":       true,
	"linux/amd64":     true,
	"linux/arm":       true,
	"linux/arm/v5":    true,
	"linux/arm/v6":    true,
	"linux/arm/v7":    true,
	"linux/arm64":     true,
	"linux/arm64/v8":  true,
	"linux/ppc64":     true,
	"linux/ppc64le":   true,
	"linux/mips64":    true,
	"linux/mips64le":  true,
	"linux/s390x":     true,
	"netbsd/386":      true,
	"netbsd/amd64":    true,
	"netbsd/arm":      true,
	"openbsd/386":     true,
	"openbsd/amd64":   true,
	"openbsd/arm":     true,
	"plan9/386":       true,
	"plan9/amd64":     true,
	"solaris/amd64":   true,
	"windows/386":     true,
	"windows/amd64":   true,
	"windows/arm":     true,
}

func parseName(name string) (reference.Named, error) {
	distref, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return nil, err
	}
	hostname, remoteName := splitHostname(distref.String())
	if hostname == "" {
		return nil, fmt.Errorf("Please use a fully qualified repository name")
	}
	return reference.ParseNormalizedNamed(fmt.Sprintf("%s/%s", hostname, remoteName))
}

const (
	// DefaultHostname is the default built-in registry (DockerHub)
	DefaultHostname = "docker.io"
	// LegacyDefaultHostname is the old hostname used for DockerHub
	LegacyDefaultHostname = "index.docker.io"
	// DefaultRepoPrefix is the prefix used for official images in DockerHub
	DefaultRepoPrefix = "library/"
)

// splitHostname splits a repository name to hostname and remotename string.
// If no valid hostname is found, the default hostname is used. Repository name
// needs to be already validated before.
func splitHostname(name string) (hostname, remoteName string) {
	i := strings.IndexRune(name, '/')
	if i == -1 || (!strings.ContainsAny(name[:i], ".:") && name[:i] != "localhost") {
		hostname, remoteName = DefaultHostname, name
	} else {
		hostname, remoteName = name[:i], name[i+1:]
	}
	if hostname == LegacyDefaultHostname {
		hostname = DefaultHostname
	}
	if hostname == DefaultHostname && !strings.ContainsRune(remoteName, '/') {
		remoteName = DefaultRepoPrefix + remoteName
	}
	return
}

func newResolver(username, password string, insecure, plainHTTP bool, configs ...string) remotes.Resolver {

	opts := docker.ResolverOptions{
		PlainHTTP: plainHTTP,
	}
	client := http.DefaultClient
	if insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}
	opts.Client = client

	if username != "" || password != "" {
		opts.Credentials = func(hostName string) (string, string, error) {
			return username, password, nil
		}
		return docker.NewResolver(opts)
	}
	cli, err := auth.NewClient(configs...)
	if err != nil {
		log.Warnf("Error loading auth file: %v", err)
	}
	resolver, err := cli.Resolver(context.Background(), client, plainHTTP)
	if err != nil {
		log.Warnf("Error loading resolver: %v", err)
		resolver = docker.NewResolver(opts)
	}
	return resolver
}

func fetchDescriptor(resolver remotes.Resolver, memoryStore *store.MemoryStore, imageRef reference.Named) (ocispec.Descriptor, error) {
	return registry.Fetch(context.Background(), memoryStore, types.NewRequest(imageRef, "", allMediaTypes(), resolver))
}

func allMediaTypes() []string {
	return []string{
		types.MediaTypeDockerSchema2Manifest,
		types.MediaTypeDockerSchema2ManifestList,
		ocispec.MediaTypeImageManifest,
		ocispec.MediaTypeImageIndex,
	}
}
