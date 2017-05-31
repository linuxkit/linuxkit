package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnforceContentTrust(t *testing.T) {
	// Simple positive and negative cases for Image subkey
	require.True(t, enforceContentTrust("image", &TrustConfig{Image: []string{"image"}}))
	require.True(t, enforceContentTrust("image", &TrustConfig{Image: []string{"more", "than", "one", "image"}}))
	require.True(t, enforceContentTrust("image", &TrustConfig{Image: []string{"more", "than", "one", "image"}, Org: []string{"random", "orgs"}}))

	require.False(t, enforceContentTrust("image", &TrustConfig{}))
	require.False(t, enforceContentTrust("image", &TrustConfig{Image: []string{"not", "in", "here!"}}))
	require.False(t, enforceContentTrust("image", &TrustConfig{Image: []string{"not", "in", "here!"}, Org: []string{""}}))

	// Tests for Image subkey with tags
	require.True(t, enforceContentTrust("image:tag", &TrustConfig{Image: []string{"image:tag"}}))
	require.True(t, enforceContentTrust("image:tag", &TrustConfig{Image: []string{"image"}}))
	require.False(t, enforceContentTrust("image:tag", &TrustConfig{Image: []string{"image:otherTag"}}))
	require.False(t, enforceContentTrust("image:tag", &TrustConfig{Image: []string{"image@sha256:abc123"}}))

	// Tests for Image subkey with digests
	require.True(t, enforceContentTrust("image@sha256:abc123", &TrustConfig{Image: []string{"image@sha256:abc123"}}))
	require.True(t, enforceContentTrust("image@sha256:abc123", &TrustConfig{Image: []string{"image"}}))
	require.False(t, enforceContentTrust("image@sha256:abc123", &TrustConfig{Image: []string{"image:Tag"}}))
	require.False(t, enforceContentTrust("image@sha256:abc123", &TrustConfig{Image: []string{"image@sha256:def456"}}))

	// Tests for Image subkey with digests
	require.True(t, enforceContentTrust("image@sha256:abc123", &TrustConfig{Image: []string{"image@sha256:abc123"}}))
	require.True(t, enforceContentTrust("image@sha256:abc123", &TrustConfig{Image: []string{"image"}}))
	require.False(t, enforceContentTrust("image@sha256:abc123", &TrustConfig{Image: []string{"image:Tag"}}))
	require.False(t, enforceContentTrust("image@sha256:abc123", &TrustConfig{Image: []string{"image@sha256:def456"}}))

	// Tests for Org subkey
	require.True(t, enforceContentTrust("linuxkit/image", &TrustConfig{Image: []string{"notImage"}, Org: []string{"linuxkit"}}))
	require.True(t, enforceContentTrust("linuxkit/differentImage", &TrustConfig{Image: []string{}, Org: []string{"linuxkit"}}))
	require.True(t, enforceContentTrust("linuxkit/differentImage:tag", &TrustConfig{Image: []string{}, Org: []string{"linuxkit"}}))
	require.True(t, enforceContentTrust("linuxkit/differentImage@sha256:abc123", &TrustConfig{Image: []string{}, Org: []string{"linuxkit"}}))

	require.False(t, enforceContentTrust("linuxkit/differentImage", &TrustConfig{Image: []string{}, Org: []string{"notlinuxkit"}}))
	require.False(t, enforceContentTrust("linuxkit/differentImage:tag", &TrustConfig{Image: []string{}, Org: []string{"notlinuxkit"}}))
	require.False(t, enforceContentTrust("linuxkit/differentImage@sha256:abc123", &TrustConfig{Image: []string{}, Org: []string{"notlinuxkit"}}))
}
