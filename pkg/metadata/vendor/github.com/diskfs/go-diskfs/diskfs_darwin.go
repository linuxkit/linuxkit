package diskfs

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// this constants should be part of "golang.org/x/sys/unix", but aren't, yet
const (
	DKIOCGETBLOCKSIZE         = 0x40046418
	DKIOCGETPHYSICALBLOCKSIZE = 0x4004644D
	DKIOCGETBLOCKCOUNT        = 0x40086419
)

// getSectorSizes get the logical and physical sector sizes for a block device
func getSectorSizes(f *os.File) (logicalSectorSize, physicalSectorSize int64, err error) {
	//nolint:gocritic // we keep this for reference to the underlying syscall
	/*
		ioctl(fd, BLKPBSZGET, &physicalsectsize);

	*/
	fd := f.Fd()

	logicalSectorSizeInt, err := unix.IoctlGetInt(int(fd), DKIOCGETBLOCKSIZE)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to get device logical sector size: %v", err)
	}
	physicalSectorSizeInt, err := unix.IoctlGetInt(int(fd), DKIOCGETPHYSICALBLOCKSIZE)
	if err != nil {
		return 0, 0, fmt.Errorf("unable to get device physical sector size: %v", err)
	}
	return int64(logicalSectorSizeInt), int64(physicalSectorSizeInt), nil
}
