package diskstream

import (
	"errors"
	"io"

	"github.com/radu-matei/azure-vhd-utils/vhdcore"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/block"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/common"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/footer"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/vhdfile"
)

// DiskStream provides a logical stream over a VHD file.
// The type exposes the VHD as a fixed VHD, regardless of actual underlying VHD type (dynamic, differencing
// or fixed type)
//
type DiskStream struct {
	offset          int64
	size            int64
	isClosed        bool
	vhdFactory      *vhdFile.FileFactory
	vhdFile         *vhdFile.VhdFile
	vhdBlockFactory block.Factory
	vhdFooterRange  *common.IndexRange
	vhdDataRange    *common.IndexRange
}

// StreamExtent describes a block range of a disk which contains data.
//
type StreamExtent struct {
	Range            *common.IndexRange
	OwnerVhdUniqueID *common.UUID
}

// CreateNewDiskStream creates a new DiskStream.
// Parameter vhdPath is the path to VHD
//
func CreateNewDiskStream(vhdPath string) (*DiskStream, error) {
	var err error
	stream := &DiskStream{offset: 0, isClosed: false}
	stream.vhdFactory = &vhdFile.FileFactory{}
	if stream.vhdFile, err = stream.vhdFactory.Create(vhdPath); err != nil {
		return nil, err
	}

	if stream.vhdBlockFactory, err = stream.vhdFile.GetBlockFactory(); err != nil {
		return nil, err
	}

	stream.vhdFooterRange = stream.vhdBlockFactory.GetFooterRange()
	stream.size = stream.vhdFooterRange.End + 1
	stream.vhdDataRange = common.NewIndexRangeFromLength(0, stream.size-stream.vhdFooterRange.Length())
	return stream, nil
}

// GetDiskType returns the type of the disk, expected values are DiskTypeFixed, DiskTypeDynamic
// or DiskTypeDifferencing
//
func (s *DiskStream) GetDiskType() footer.DiskType {
	return s.vhdFile.GetDiskType()
}

// GetSize returns the length of the stream in bytes.
//
func (s *DiskStream) GetSize() int64 {
	return s.size
}

// Read reads up to len(b) bytes from the Vhd file. It returns the number of bytes read and an error,
// if any. EOF is signaled when no more data to read and n will set to 0.
//
// If the internal read offset is a byte offset in the data segment of the VHD and If reader reaches
// end of data section after reading some but not all the bytes then Read won't read from the footer
// section, the next Read will read from the footer.
//
// If the internal read offset is a byte offset in the footer segment of the VHD and if reader reaches
// end of footer section after reading some but not all the bytes then Read won't return any error.
//
// Read satisfies io.Reader interface
//
func (s *DiskStream) Read(p []byte) (n int, err error) {
	if s.offset >= s.size {
		return 0, io.EOF
	}

	count := len(p)
	if count == 0 {
		return 0, nil
	}

	rangeToRead := common.NewIndexRangeFromLength(s.offset, int64(count))
	if s.vhdDataRange.Intersects(rangeToRead) {
		writtenCount, err := s.readFromBlocks(rangeToRead, p)
		s.offset += int64(writtenCount)
		return writtenCount, err
	}

	if s.vhdFooterRange.Intersects(rangeToRead) {
		writtenCount, err := s.readFromFooter(rangeToRead, p)
		s.offset += int64(writtenCount)
		return writtenCount, err
	}

	return 0, nil
}

// Seek sets the offset for the next Read on the stream to offset, interpreted according to whence:
// 0 means relative to the origin of the stream, 1 means relative to the current offset, and 2
// means relative to the end. It returns the new offset and an error, if any.
//
// Seek satisfies io.Seeker interface
//
func (s *DiskStream) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	default:
		return 0, errors.New("Seek: invalid whence")
	case 0:
		offset += 0
	case 1:
		offset += s.offset
	case 2:
		offset += s.size - 1
	}

	if offset < 0 || offset >= s.size {
		return 0, errors.New("Seek: invalid offset")
	}

	s.offset = offset
	return offset, nil
}

// Close closes the VHD file, rendering it unusable for I/O. It returns an error, if any.
//
// Close satisfies io.Closer interface
//
func (s *DiskStream) Close() error {
	if !s.isClosed {
		s.vhdFactory.Dispose(nil)
		s.isClosed = true
	}

	return nil
}

// GetExtents gets the extents of the stream that contain non-zero data. Each extent describes a block's data
// section range which contains data.
// For dynamic or differencing disk - a block is empty if the BAT corresponding to the block contains 0xFFFFFFFF
// so returned extents slice will not contain such range.
// For fixed disk - this method returns extents describing ranges of all blocks, to rule out fixed disk block
// ranges containing zero bytes use DetectEmptyRanges function in upload package.
//
func (s *DiskStream) GetExtents() ([]*StreamExtent, error) {
	extents := make([]*StreamExtent, 1)
	blocksCount := s.vhdBlockFactory.GetBlockCount()
	for i := int64(0); i < blocksCount; i++ {
		currentBlock, err := s.vhdBlockFactory.Create(uint32(i))
		if err != nil {
			return nil, err
		}
		if !currentBlock.IsEmpty {
			extents = append(extents, &StreamExtent{
				Range:            currentBlock.LogicalRange,
				OwnerVhdUniqueID: currentBlock.VhdUniqueID,
			})
		}
	}
	extents = append(extents, &StreamExtent{
		Range:            s.vhdFooterRange,
		OwnerVhdUniqueID: s.vhdFile.Footer.UniqueID,
	})

	return extents, nil
}

// EnumerateExtents iterate through the extents of the stream that contain non-zero data and invokes the function
// identified by the parameter f for each extent. Each extent describes a block's data section range which
// contains data.
// For dynamic or differencing disk - a block is empty if the BAT corresponding to the block contains 0xFFFFFFFF
// so returned extents slice will not contain such range.
// For fixed disk - this method returns extents describing ranges of all blocks, to rule out fixed disk block
// ranges containing zero bytes use DetectEmptyRanges function in upload package.
//
func (s *DiskStream) EnumerateExtents(f func(*StreamExtent, error) bool) {
	blocksCount := s.vhdBlockFactory.GetBlockCount()
	i := int64(0)
	for ; i < blocksCount; i++ {
		if currentBlock, err := s.vhdBlockFactory.Create(uint32(i)); err != nil {
			continueEnumerate := f(nil, err)
			if !continueEnumerate {
				break
			}
		} else {
			if !currentBlock.IsEmpty {
				continueEnumerate := f(&StreamExtent{
					Range:            currentBlock.LogicalRange,
					OwnerVhdUniqueID: currentBlock.VhdUniqueID,
				}, nil)
				if !continueEnumerate {
					break
				}
			}
		}
	}
	if i == blocksCount {
		f(&StreamExtent{
			Range:            s.vhdFooterRange,
			OwnerVhdUniqueID: s.vhdFile.Footer.UniqueID,
		}, nil)
	}
}

// readFromBlocks identifies the blocks constituting the range rangeToRead, and read data from these
// blocks into p. It returns the number of bytes read, which will be the minimum of sum of lengths
// of all constituting range and len(p), provided there is no error.
//
func (s *DiskStream) readFromBlocks(rangeToRead *common.IndexRange, p []byte) (n int, err error) {
	rangeToReadFromBlocks := s.vhdDataRange.Intersection(rangeToRead)
	if rangeToReadFromBlocks == nil {
		return 0, nil
	}

	writtenCount := 0
	maxCount := len(p)
	blockSize := s.vhdBlockFactory.GetBlockSize()
	startingBlock := s.byteToBlock(rangeToReadFromBlocks.Start)
	endingBlock := s.byteToBlock(rangeToReadFromBlocks.End)

	for blockIndex := startingBlock; blockIndex <= endingBlock && writtenCount < maxCount; blockIndex++ {
		currentBlock, err := s.vhdBlockFactory.Create(uint32(blockIndex))
		if err != nil {
			return writtenCount, err
		}

		blockData, err := currentBlock.Data()
		if err != nil {
			return writtenCount, err
		}

		rangeToReadInBlock := currentBlock.LogicalRange.Intersection(rangeToReadFromBlocks)
		copyStartIndex := rangeToReadInBlock.Start % blockSize
		writtenCount += copy(p[writtenCount:], blockData[copyStartIndex:(copyStartIndex+rangeToReadInBlock.Length())])
	}

	return writtenCount, nil
}

// readFromFooter reads the range rangeToRead from footer into p. It returns the number of bytes read, which
// will be minimum of the given range length and len(p), provided there is no error.
//
func (s *DiskStream) readFromFooter(rangeToRead *common.IndexRange, p []byte) (n int, err error) {
	rangeToReadFromFooter := s.vhdFooterRange.Intersection(rangeToRead)
	if rangeToReadFromFooter == nil {
		return 0, nil
	}

	vhdFooter := s.vhdFile.Footer.CreateCopy()
	if vhdFooter.DiskType != footer.DiskTypeFixed {
		vhdFooter.DiskType = footer.DiskTypeFixed
		vhdFooter.HeaderOffset = vhdcore.VhdNoDataLong
		vhdFooter.CreatorApplication = "wa"
	}
	// As per VHD spec, the size reported by the footer should same as 'header.MaxTableEntries * header.BlockSize'
	// But the VHD created by some tool (e.g. qemu) are not honoring this. Azure will reject the VHD if the size
	// specified in the footer of VHD not match 'VHD blob size - VHD Footer Size'
	//
	vhdFooter.PhysicalSize = s.GetSize() - vhdcore.VhdFooterSize
	vhdFooter.VirtualSize = s.GetSize() - vhdcore.VhdFooterSize

	// Calculate the checksum and serialize the footer
	//
	vhdFooterBytes := footer.SerializeFooter(vhdFooter)
	copyStartIndex := rangeToReadFromFooter.Start - s.vhdFooterRange.Start
	writtenCount := copy(p, vhdFooterBytes[copyStartIndex:copyStartIndex+rangeToReadFromFooter.Length()])
	return writtenCount, nil
}

// byteToBlock returns the block index corresponding to the given byte position.
//
func (s *DiskStream) byteToBlock(position int64) int64 {
	sectorsPerBlock := s.vhdBlockFactory.GetBlockSize() / vhdcore.VhdSectorLength
	return position / vhdcore.VhdSectorLength / sectorsPerBlock
}
