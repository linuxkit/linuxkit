package block

import (
	"io"

	"github.com/radu-matei/azure-vhd-utils/vhdcore/footer"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/reader"
)

// FixedDiskBlockReader type satisfies BlockDataReader interface,
// implementation of BlockDataReader::Read by this type can read the data from a block
// of a fixed disk.
//
type FixedDiskBlockReader struct {
	vhdReader        *reader.VhdReader
	blockSizeInBytes uint32
}

// NewFixedDiskBlockReader create a new instance of FixedDiskBlockReader which can read data from
// a fixed disk block.
// The parameter vhdReader is the reader to read the disk
// The parameter blockSizeInBytes is the size of the fixed disk block
//
func NewFixedDiskBlockReader(vhdReader *reader.VhdReader, blockSizeInBytes uint32) *FixedDiskBlockReader {
	return &FixedDiskBlockReader{
		vhdReader:        vhdReader,
		blockSizeInBytes: blockSizeInBytes,
	}
}

// Read reads the data in a block of a fixed disk
// The parameter block represents the block to read
//
func (r *FixedDiskBlockReader) Read(block *Block) ([]byte, error) {
	blockIndex := block.BlockIndex
	blockByteOffset := int64(blockIndex) * int64(r.blockSizeInBytes)
	blockDataBuffer := make([]byte, block.LogicalRange.Length())
	n, err := r.vhdReader.ReadBytes(blockByteOffset, blockDataBuffer)
	if err == io.ErrUnexpectedEOF {
		return blockDataBuffer[:n], nil
	}

	if err != nil {
		return nil, NewDataReadError(blockIndex, footer.DiskTypeFixed, err)
	}
	return blockDataBuffer, nil
}
