package util

import (
	"strings"
)

type refOpts struct {
	withTag bool
}
type ReferenceOption func(r *refOpts)

// ReferenceWithTag returns a ReferenceOption that ensures a tag is filled. If the tag is not provided,
// the default is added
func ReferenceWithTag() ReferenceOption {
	return func(r *refOpts) {
		r.withTag = true
	}
}

// ReferenceExpand expands "redis" to "docker.io/library/redis" so all images have a full domain,
// and similarly foo/bar to docker.io/foo/bar.
// If the image does not have a tag, ":latest" is added.
func ReferenceExpand(ref string, options ...ReferenceOption) string {
	var opts refOpts
	for _, opt := range options {
		opt(&opts)
	}
	ret := ref

	parts := strings.Split(ref, "/")
	switch len(parts) {
	case 1:
		ret = "docker.io/library/" + ref
	case 2:
		// If the first part is not a domain, assume it is a DockerHub user/org.
		// This logic is copied from moby:
		// https://github.com/moby/moby/blob/e7347f8a8c2fd3d2abd34b638d6fc8c18b0278d1/registry/search.go#L148C29-L149C71
		if !strings.Contains(parts[0], ".") && !strings.Contains(parts[0], ":") && parts[0] != "localhost" {
			ret = "docker.io/" + ref
		}
	}

	if opts.withTag && !strings.Contains(ret, ":") {
		ret += ":latest"
	}
	return ret
}
