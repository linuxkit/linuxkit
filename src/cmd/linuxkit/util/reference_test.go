package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReferenceExpand(t *testing.T) {
	tests := []struct {
		name    string
		ref     string
		options []ReferenceOption
		want    string
	}{
		{
			"basic image name should expand to docker.io/library image",
			"redis",
			nil,
			"docker.io/library/redis",
		},
		{
			"image name with user/org should expand to docker.io image",
			"foo/bar",
			nil,
			"docker.io/foo/bar",
		},
		{
			"custom registry image name should not expand",
			"myregistry.io/foo",
			nil,
			"myregistry.io/foo",
		},
		{
			"image name with more than three parts should not expand",
			"foo/bar/baz",
			nil,
			"foo/bar/baz",
		},
		{
			"with tag should add latest if image does not have tag",
			"redis",
			[]ReferenceOption{ReferenceWithTag()},
			"docker.io/library/redis:latest",
		},
		{
			"with tag should not add latest if image already has tag",
			"redis:alpine",
			[]ReferenceOption{ReferenceWithTag()},
			"docker.io/library/redis:alpine",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReferenceExpand(tt.ref, tt.options...)
			assert.Equal(t, tt.want, got)
		})
	}
}
