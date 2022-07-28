package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/distribution/reference"
	"github.com/estesp/manifest-tool/v2/pkg/store"
	"github.com/estesp/manifest-tool/v2/pkg/types"
	"github.com/estesp/manifest-tool/v2/pkg/util"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/sirupsen/logrus"
)

func PushManifestList(username, password string, input types.YAMLInput, ignoreMissing, insecure, plainHttp bool, manifestType types.ManifestType, configDir string) (hash string, length int, err error) {
	// resolve the target image reference for the combined manifest list/index
	targetRef, err := reference.ParseNormalizedNamed(input.Image)
	if err != nil {
		return hash, length, fmt.Errorf("Error parsing name for manifest list (%s): %v", input.Image, err)
	}

	var configDirs []string
	if configDir != "" {
		configDirs = append(configDirs, configDir)
	}
	resolver := util.NewResolver(username, password, insecure,
		plainHttp, configDirs...)

	manifestList := types.ManifestList{
		Name:      input.Image,
		Reference: targetRef,
		Resolver:  resolver,
		Type:      manifestType,
	}
	// create an in-memory store for OCI descriptors and content used during the push operation
	memoryStore := store.NewMemoryStore()

	logrus.Info("Retrieving digests of member images")
	for _, img := range input.Manifests {
		ref, err := util.ParseName(img.Image)
		if err != nil {
			return hash, length, fmt.Errorf("Unable to parse image reference: %s: %v", img.Image, err)
		}
		if reference.Domain(targetRef) != reference.Domain(ref) {
			return hash, length, fmt.Errorf("Source image (%s) registry does not match target image (%s) registry", ref, targetRef)
		}
		descriptor, err := FetchDescriptor(resolver, memoryStore, ref)
		if err != nil {
			if ignoreMissing {
				logrus.Warnf("Couldn't access image '%q'. Skipping due to 'ignore missing' configuration.", img.Image)
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
			// only need to handle cross-repo blob mount for distributable layer types
			if skippable(layer.MediaType) {
				continue
			}
			info.Digest = layer.Digest
			if _, err := memoryStore.Update(context.TODO(), info, ""); err != nil {
				logrus.Warnf("couldn't update in-memory store labels for %v: %v", info.Digest, err)
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

	return Push(manifestList, input.Tags, memoryStore)
}

func resolvePlatform(descriptor ocispec.Descriptor, img types.ManifestEntry, imgConfig types.Image) (*ocispec.Platform, error) {
	platform := &img.Platform
	if platform == nil {
		platform = &ocispec.Platform{}
	}
	// fill os/arch from inspected image if not specified in input YAML
	if platform.OS == "" && platform.Architecture == "" {
		// prefer a full platform object, if one is already available (and appears to have meaningful content)
		if descriptor.Platform != nil && (descriptor.Platform.OS != "" || descriptor.Platform.Architecture != "") {
			platform = descriptor.Platform
		} else if imgConfig.OS != "" || imgConfig.Architecture != "" {
			platform.OS = imgConfig.OS
			platform.Architecture = imgConfig.Architecture
		}
	}
	// if Variant is specified in the origin image but not the descriptor or YAML, bubble it up
	if imgConfig.Variant != "" && platform.Variant == "" {
		platform.Variant = imgConfig.Variant
	}
	// Windows: if the origin image has OSFeature and/or OSVersion information, and
	// these values were not specified in the creation YAML, then
	// retain the origin values in the Platform definition for the manifest list:
	if imgConfig.OSVersion != "" && platform.OSVersion == "" {
		platform.OSVersion = imgConfig.OSVersion
	}
	if len(imgConfig.OSFeatures) > 0 && len(platform.OSFeatures) == 0 {
		platform.OSFeatures = imgConfig.OSFeatures
	}

	// validate os/arch input
	if !util.IsValidOSArch(platform.OS, platform.Architecture, platform.Variant) {
		return nil, fmt.Errorf("Manifest entry for image %s has unsupported os/arch or os/arch/variant combination: %s/%s/%s", img.Image, platform.OS, platform.Architecture, platform.Variant)
	}
	return platform, nil
}

func skippable(mediaType string) bool {
	// skip foreign/non-distributable layers
	if strings.Index(mediaType, "foreign") > 0 || strings.Index(mediaType, "nondistributable") > 0 {
		return true
	}
	// skip manifests (OCI or Dockerv2) as they are already handled on push references code
	switch mediaType {
	case ocispec.MediaTypeImageManifest, types.MediaTypeDockerSchema2Manifest:
		return true
	}
	return false
}
