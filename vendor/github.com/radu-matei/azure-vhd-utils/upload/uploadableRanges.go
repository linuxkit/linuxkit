package upload

import (
	"github.com/radu-matei/azure-vhd-utils/vhdcore/common"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/diskstream"
)

// LocateUploadableRanges detects the uploadable ranges in a VHD stream, size of each range is at most pageSizeInBytes.
//
// This method reads the existing ranges A from the disk stream, creates a new set of ranges B from A by removing the
// ranges identified by the parameter rangesToSkip, returns new set of ranges C (with each range of size at most
// pageSizeInBytes) by merging adjacent ranges in B or splitting ranges in B.
//
// Note that this method will not check whether ranges of a fixed disk contains zeros, hence inorder to filter out such
// ranges from the uploadable ranges, caller must use LocateNonEmptyRangeIndices method.
//
func LocateUploadableRanges(stream *diskstream.DiskStream, rangesToSkip []*common.IndexRange, pageSizeInBytes int64) ([]*common.IndexRange, error) {
	var err error
	var diskRanges = make([]*common.IndexRange, 0)
	stream.EnumerateExtents(func(ext *diskstream.StreamExtent, extErr error) bool {
		if extErr != nil {
			err = extErr
			return false
		}

		diskRanges = append(diskRanges, ext.Range)
		return true
	})

	if err != nil {
		return nil, err
	}

	diskRanges = common.SubtractRanges(diskRanges, rangesToSkip)
	diskRanges = common.ChunkRangesBySize(diskRanges, pageSizeInBytes)
	return diskRanges, nil
}
