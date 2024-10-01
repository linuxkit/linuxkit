package cache

import (
	"fmt"
	"strings"

	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

func platformString(p imagespec.Platform) string {
	parts := []string{p.OS, p.Architecture}
	if p.Variant != "" {
		parts = append(parts, p.Variant)
	}
	return strings.Join(parts, "/")
}

func platformMessageGenerator(platforms []imagespec.Platform) string {
	var platformMessage string
	switch {
	case len(platforms) == 0:
		platformMessage = "all platforms"
	case len(platforms) == 1:
		platformMessage = fmt.Sprintf("platform %s", platformString(platforms[0]))
	default:
		var platStrings []string
		for _, p := range platforms {
			platStrings = append(platStrings, platformString(p))
		}
		platformMessage = fmt.Sprintf("platforms %s", strings.Join(platStrings, ","))
	}
	return platformMessage
}
