package util

import "fmt"

//go:generate go run osgen.go

var (
	armVariants = map[string]bool{
		"v5": true,
		"v6": true,
		"v7": true,
	}
)

// IsValidOSArch checks against the generated list of os/arch combinations
// from Go as well as checking for valid variants for ARM (the only architecture that uses variants)
func IsValidOSArch(os string, arch string, variant string) bool {
	osarch := fmt.Sprintf("%s/%s", os, arch)
	if _, ok := validOS[os]; !ok {
		return false
	}
	if _, ok := validArch[arch]; !ok {
		return false
	}
	if variant == "" {
		return true
	}

	// only arm/arm64 can use variant
	switch osarch {
	case "linux/arm":
		_, ok := armVariants[variant]
		return ok
	case "linux/arm64":
		if variant == "v8" {
			return true
		}
	default:
		return false
	}
	return false
}
