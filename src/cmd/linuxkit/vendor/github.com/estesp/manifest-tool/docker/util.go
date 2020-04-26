package docker

import "fmt"

// list of valid os/arch values (see "Optional Environment Variables" section
// of https://golang.org/doc/install/source
// Added linux/s390x as we know System z support already exists

var validOSArch = map[string]bool{
	"darwin/386":      true,
	"darwin/amd64":    true,
	"darwin/arm":      true,
	"darwin/arm64":    true,
	"dragonfly/amd64": true,
	"freebsd/386":     true,
	"freebsd/amd64":   true,
	"freebsd/arm":     true,
	"linux/386":       true,
	"linux/amd64":     true,
	"linux/arm":       true,
	"linux/arm/v5":    true,
	"linux/arm/v6":    true,
	"linux/arm/v7":    true,
	"linux/arm64":     true,
	"linux/arm64/v8":  true,
	"linux/ppc64":     true,
	"linux/ppc64le":   true,
	"linux/mips64":    true,
	"linux/mips64le":  true,
	"linux/s390x":     true,
	"netbsd/386":      true,
	"netbsd/amd64":    true,
	"netbsd/arm":      true,
	"openbsd/386":     true,
	"openbsd/amd64":   true,
	"openbsd/arm":     true,
	"plan9/386":       true,
	"plan9/amd64":     true,
	"solaris/amd64":   true,
	"windows/386":     true,
	"windows/amd64":   true,
	"windows/arm":     true,
}

func isValidOSArch(os string, arch string, variant string) bool {
	osarch := fmt.Sprintf("%s/%s", os, arch)

	if variant != "" {
		osarch = fmt.Sprintf("%s/%s/%s", os, arch, variant)
	}

	_, ok := validOSArch[osarch]
	return ok
}
