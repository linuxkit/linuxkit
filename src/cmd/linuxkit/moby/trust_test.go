package moby

import "testing"

func TestEnforceContentTrust(t *testing.T) {
	type enforceContentTrustCase struct {
		result      bool
		imageName   string
		trustConfig *TrustConfig
	}
	testCases := []enforceContentTrustCase{
		// Simple positive and negative cases for Image subkey
		{true, "image", &TrustConfig{Image: []string{"image"}}},
		{true, "image", &TrustConfig{Image: []string{"more", "than", "one", "image"}}},
		{true, "image", &TrustConfig{Image: []string{"more", "than", "one", "image"}, Org: []string{"random", "orgs"}}},
		{false, "image", &TrustConfig{}},
		{false, "image", &TrustConfig{Image: []string{"not", "in", "here!"}}},
		{false, "image", &TrustConfig{Image: []string{"not", "in", "here!"}, Org: []string{""}}},

		// Tests for Image subkey with tags
		{true, "image:tag", &TrustConfig{Image: []string{"image:tag"}}},
		{true, "image:tag", &TrustConfig{Image: []string{"image"}}},
		{false, "image:tag", &TrustConfig{Image: []string{"image:otherTag"}}},
		{false, "image:tag", &TrustConfig{Image: []string{"image@sha256:abc123"}}},

		// Tests for Image subkey with digests
		{true, "image@sha256:abc123", &TrustConfig{Image: []string{"image@sha256:abc123"}}},
		{true, "image@sha256:abc123", &TrustConfig{Image: []string{"image"}}},
		{false, "image@sha256:abc123", &TrustConfig{Image: []string{"image:Tag"}}},
		{false, "image@sha256:abc123", &TrustConfig{Image: []string{"image@sha256:def456"}}},

		// Tests for Image subkey with digests
		{true, "image@sha256:abc123", &TrustConfig{Image: []string{"image@sha256:abc123"}}},
		{true, "image@sha256:abc123", &TrustConfig{Image: []string{"image"}}},
		{false, "image@sha256:abc123", &TrustConfig{Image: []string{"image:Tag"}}},
		{false, "image@sha256:abc123", &TrustConfig{Image: []string{"image@sha256:def456"}}},

		// Tests for Org subkey
		{true, "linuxkit/image", &TrustConfig{Image: []string{"notImage"}, Org: []string{"linuxkit"}}},
		{true, "linuxkit/differentImage", &TrustConfig{Image: []string{}, Org: []string{"linuxkit"}}},
		{true, "linuxkit/differentImage:tag", &TrustConfig{Image: []string{}, Org: []string{"linuxkit"}}},
		{true, "linuxkit/differentImage@sha256:abc123", &TrustConfig{Image: []string{}, Org: []string{"linuxkit"}}},
		{false, "linuxkit/differentImage", &TrustConfig{Image: []string{}, Org: []string{"notlinuxkit"}}},
		{false, "linuxkit/differentImage:tag", &TrustConfig{Image: []string{}, Org: []string{"notlinuxkit"}}},
		{false, "linuxkit/differentImage@sha256:abc123", &TrustConfig{Image: []string{}, Org: []string{"notlinuxkit"}}},

		// Tests for Org with library organization
		{true, "nginx", &TrustConfig{Image: []string{}, Org: []string{"library"}}},
		{true, "nginx:alpine", &TrustConfig{Image: []string{}, Org: []string{"library"}}},
		{true, "library/nginx:alpine", &TrustConfig{Image: []string{}, Org: []string{"library"}}},
		{false, "nginx", &TrustConfig{Image: []string{}, Org: []string{"notLibrary"}}},
	}
	for _, testCase := range testCases {
		if enforceContentTrust(testCase.imageName, testCase.trustConfig) != testCase.result {
			t.Errorf("incorrect trust enforcement result for %s against configuration %v, expected: %v", testCase.imageName, testCase.trustConfig, testCase.result)
		}
	}
}
