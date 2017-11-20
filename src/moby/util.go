package moby

import (
	"path/filepath"
)

var (
	// MobyDir is the location of the cache directory, defaults to ~/.moby
	MobyDir string
)

func defaultMobyConfigDir() string {
	mobyDefaultDir := ".moby"
	home := homeDir()
	return filepath.Join(home, mobyDefaultDir)
}
