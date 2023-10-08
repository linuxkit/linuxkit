package util

import "fmt"

// MArch turn an input arch into a canonical arch as given by `uname -m`
func MArch(in string) (string, error) {
	switch in {
	case "x86_64", "amd64":
		return "x86_64", nil
	case "aarch64", "arm64":
		return "aarch64", nil
	}
	return "", fmt.Errorf("unknown arch %q", in)
}

// GoArch turn an input arch into a go arch
func GoArch(in string) (string, error) {
	switch in {
	case "x86_64", "amd64":
		return "amd64", nil
	case "aarch64", "arm64":
		return "arm64", nil
	}
	return "", fmt.Errorf("unknown arch %q", in)
}
