//go:build !windows
// +build !windows

package util

import (
	"os"
)

// HomeDir get the home directory for the user based on the HOME environment variable.
func HomeDir() string {
	return os.Getenv("HOME")
}
