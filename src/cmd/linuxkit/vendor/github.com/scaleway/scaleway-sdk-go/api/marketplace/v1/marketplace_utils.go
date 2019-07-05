package marketplace

import (
	"fmt"

	"github.com/scaleway/scaleway-sdk-go/utils"
)

// getLocalImage returns the correct local version of an image matching
// the current zone and the compatible commercial type
func (version *Version) getLocalImage(zone utils.Zone, commercialType string) (*LocalImage, error) {

	for _, localImage := range version.LocalImages {

		// Check if in correct zone
		if localImage.Zone != zone {
			continue
		}

		// Check if compatible with wanted commercial type
		for _, compatibleCommercialType := range localImage.CompatibleCommercialTypes {
			if compatibleCommercialType == commercialType {
				return localImage, nil
			}
		}
	}

	return nil, fmt.Errorf("couldn't find compatible local image for this image version (%s)", version.ID)

}

// getLatestVersion returns the current/latests version on an image,
// or an error in case the image doesn't have a public version.
func (image *Image) getLatestVersion() (*Version, error) {

	for _, version := range image.Versions {
		if version.ID == image.CurrentPublicVersion {
			return version, nil
		}
	}

	return nil, fmt.Errorf("latest version could not be found for image %s", image.Name)
}

// FindLocalImageIDByName search for an image with the given name (exact match) in the given region
// it returns the latest version of this specific image.
func (s *API) FindLocalImageIDByName(imageName string, zone utils.Zone, commercialType string) (string, error) {

	listImageRequest := &ListImagesRequest{}
	listImageResponse, err := s.ListImages(listImageRequest)
	if err != nil {
		return "", err
	}

	// TODO: handle pagination

	images := listImageResponse.Images
	_ = images

	for _, image := range images {

		// Match name of the image
		if image.Name == imageName {

			latestVersion, err := image.getLatestVersion()
			if err != nil {
				return "", fmt.Errorf("couldn't find a matching image for the given name (%s), zone (%s) and commercial type (%s): %s", imageName, zone, commercialType, err)
			}

			localImage, err := latestVersion.getLocalImage(zone, commercialType)
			if err != nil {
				return "", fmt.Errorf("couldn't find a matching image for the given name (%s), zone (%s) and commercial type (%s): %s", imageName, zone, commercialType, err)
			}

			return localImage.ID, nil
		}

	}

	return "", fmt.Errorf("couldn't find a matching image for the given name (%s), zone (%s) and commercial type (%s)", imageName, zone, commercialType)
}
