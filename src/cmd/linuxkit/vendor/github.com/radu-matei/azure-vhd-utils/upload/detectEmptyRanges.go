package upload

import (
	"fmt"
	"io"
	"math"

	"github.com/radu-matei/azure-vhd-utils/vhdcore/block/bitmap"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/common"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/diskstream"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/footer"
)

// DataWithRange type describes a range and data associated with the range.
//
type DataWithRange struct {
	Range *common.IndexRange
	Data  []byte
}

// DetectEmptyRanges read the ranges identified by the parameter uploadableRanges from the disk stream, detect the empty
// ranges and update the uploadableRanges slice by removing the empty ranges. This method returns the updated ranges.
// The empty range detection required only for Fixed disk, if the stream is a expandable disk stream this method simply
// returns the parameter uploadableRanges as it is.
//
func DetectEmptyRanges(diskStream *diskstream.DiskStream, uploadableRanges []*common.IndexRange) ([]*common.IndexRange, error) {
	if diskStream.GetDiskType() != footer.DiskTypeFixed {
		return uploadableRanges, nil
	}

	fmt.Println("\nDetecting empty ranges..")
	totalRangesCount := len(uploadableRanges)
	lastIndex := int32(-1)
	emptyRangesCount := int32(0)
	bits := make([]byte, int32(math.Ceil(float64(totalRangesCount)/float64(8))))
	bmap := bitmap.NewBitMapFromByteSliceCopy(bits)
	indexChan, errChan := LocateNonEmptyRangeIndices(diskStream, uploadableRanges)
L:
	for {
		select {
		case index, ok := <-indexChan:
			if !ok {
				break L
			}
			bmap.Set(index, true)
			emptyRangesCount += index - lastIndex - 1
			lastIndex = index
			fmt.Printf("\r Empty ranges : %d/%d", emptyRangesCount, totalRangesCount)
		case err := <-errChan:
			return nil, err
		}
	}

	// Remove empty ranges from the uploadable ranges slice.
	i := int32(0)
	for j := 0; j < totalRangesCount; j++ {
		if set, _ := bmap.Get(int32(j)); set {
			uploadableRanges[i] = uploadableRanges[j]
			i++
		}
	}
	uploadableRanges = uploadableRanges[:i]
	return uploadableRanges, nil
}

// LocateNonEmptyRangeIndices scan the given range and detects  a subset of ranges which contains data.
// It reports the indices of non-empty ranges via a channel. This method returns two channels, an int channel - used
// to report the non-empty range indices and error channel - used to report any error while performing empty detection.
// int channel will be closed on a successful completion, the caller must not expect any more value in the
// int channel if the error channel is signaled.
//
func LocateNonEmptyRangeIndices(stream *diskstream.DiskStream, ranges []*common.IndexRange) (<-chan int32, <-chan error) {
	indexChan := make(chan int32, 0)
	errorChan := make(chan error, 0)
	go func() {
		count := int64(-1)
		var buf []byte
		for index, r := range ranges {
			if count != r.Length() {
				count = r.Length()
				buf = make([]byte, count)
			}

			_, err := stream.Seek(r.Start, 0)
			if err != nil {
				errorChan <- err
				return
			}
			_, err = io.ReadFull(stream, buf)
			if err != nil {
				errorChan <- err
				return
			}
			if !isAllZero(buf) {
				indexChan <- int32(index)
			}
		}
		close(indexChan)
	}()
	return indexChan, errorChan
}

// isAllZero returns true if the given byte slice contain all zeros
//
func isAllZero(buf []byte) bool {
	l := len(buf)
	j := 0
	for ; j < l; j++ {
		if buf[j] != byte(0) {
			break
		}
	}
	return j == l
}
