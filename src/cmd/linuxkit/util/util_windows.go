package util

import (
	"os"
)

// HomeDir return the home directory based on the USERPROFILE environment variable.
func HomeDir() string {
	return os.Getenv("USERPROFILE")
}
