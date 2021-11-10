package util

import (
	"fmt"
	"strings"

	"github.com/docker/distribution/reference"
)

const (
	// DefaultHostname is the default built-in registry (DockerHub)
	DefaultHostname = "docker.io"
	// LegacyDefaultHostname is the old hostname used for DockerHub
	LegacyDefaultHostname = "index.docker.io"
	// DefaultRepoPrefix is the prefix used for official images in DockerHub
	DefaultRepoPrefix = "library/"
)

func ParseName(name string) (reference.Named, error) {
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
