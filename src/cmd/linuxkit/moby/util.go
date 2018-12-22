package moby

import (
	"path/filepath"
	"time"
)

var (
	// MobyDir is the location of the cache directory, defaults to ~/.moby
	MobyDir string
	// Default ModTime for files created during build. Roughly the time LinuxKit got open sourced.
	defaultModTime = time.Date(2017, time.April, 18, 16, 30, 0, 0, time.UTC)
)

func defaultMobyConfigDir() string {
	mobyDefaultDir := ".moby"
	home := homeDir()
	return filepath.Join(home, mobyDefaultDir)
}
