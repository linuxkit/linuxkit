package build

import (
	"path/filepath"
	"time"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
)

var (
	// MobyDir is the location of the cache directory, defaults to ~/.moby
	MobyDir = defaultMobyConfigDir()
	// Default ModTime for files created during build. Roughly the time LinuxKit got open sourced.
	defaultModTime = time.Date(2017, time.April, 18, 16, 30, 0, 0, time.UTC)
)

func defaultMobyConfigDir() string {
	mobyDefaultDir := ".moby"
	home := util.HomeDir()
	return filepath.Join(home, mobyDefaultDir)
}
