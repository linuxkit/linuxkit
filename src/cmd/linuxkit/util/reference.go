package util

import "strings"

// ReferenceExpand expands "redis" to "docker.io/library/redis" so all images have a full domain
func ReferenceExpand(ref string) string {
	parts := strings.Split(ref, "/")
	switch len(parts) {
	case 1:
		return "docker.io/library/" + ref
	case 2:
		return "docker.io/" + ref
	default:
		return ref
	}
}
