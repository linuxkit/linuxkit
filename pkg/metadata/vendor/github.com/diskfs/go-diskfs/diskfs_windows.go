package diskfs

import (
	"errors"
	"os"
)

// getSectorSizes get the logical and physical sector sizes for a block device
func getSectorSizes(f *os.File) (int64, int64, error) {
	return 0, 0, errors.New("block devices not supported on windows")
}
