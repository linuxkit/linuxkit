// +build !windows

package iso9660

import (
	"os"
	"syscall"
)

func statt(fi os.FileInfo) (uint32, uint32, uint32) {
	if sys := fi.Sys(); sys != nil {
		if stat, ok := sys.(*syscall.Stat_t); ok {
			return uint32(stat.Nlink), uint32(stat.Uid), uint32(stat.Gid)
		}
	}

	return uint32(0), uint32(0), uint32(0)
}
