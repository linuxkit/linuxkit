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

// ReferenceExpand expands "redis" to "docker.io/library/redis" so all images have a full domain
func ReferenceExpand(ref string, options ...ReferenceOption) string {
	var opts refOpts
	for _, opt := range options {
		opt(&opts)
	}
	ret := ref

	parts := strings.Split(ref, "/")
	if len(parts) == 1 {
		ret = "docker.io/library/" + ref
	}

	if opts.withTag && !strings.Contains(ret, ":") {
		ret += ":latest"
	}
	return ret
}
