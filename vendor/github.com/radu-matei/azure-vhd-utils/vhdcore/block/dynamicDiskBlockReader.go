package block

import (
	"io"

	"github.com/radu-matei/azure-vhd-utils/vhdcore/bat"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/footer"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/reader"
)

// DynamicDiskBlockReader type satisfies BlockDataReader interface,
// implementation of BlockDataReader::Read by this type can read the 'data' section
// of a dynamic disk's block.
//
type DynamicDiskBlockReader struct {
	vhdReader            *reader.VhdReader
	blockAllocationTable *bat.BlockAllocationTable
	blockSizeInBytes     uint32
	emptyBlockData       []byte
}

// NewDynamicDiskBlockReader create a new instance of DynamicDiskBlockReader which read
// the 'data' section of dynamic disk block.
// The parameter vhdReader is the reader to read the disk
// The parameter blockAllocationTable represents the disk's BAT
// The parameter blockSizeInBytes is the size of the dynamic disk block
//
func NewDynamicDiskBlockReader(vhdReader *reader.VhdReader, blockAllocationTable *bat.BlockAllocationTable, blockSizeInBytes uint32) *DynamicDiskBlockReader {

	return &DynamicDiskBlockReader{
		vhdReader:            vhdReader,
		blockAllocationTable: blockAllocationTable,
		blockSizeInBytes:     blockSizeInBytes,
		emptyBlockData:       nil,
	}
}

// Read reads the data in a block of a dynamic disk
// The parameter block represents the block whose 'data' section to read
//
func (r *DynamicDiskBlockReader) Read(block *Block) ([]byte, error) {
	blockIndex := block.BlockIndex
	if !r.blockAllocationTable.HasData(blockIndex) {
		if r.emptyBlockData == nil {
			r.emptyBlockData = make([]byte, r.blockSizeInBytes)
		}
		return r.emptyBlockData, nil
	}

	blockDataByteOffset := r.blockAllocationTable.GetBlockDataAddress(blockIndex)
	blockDataBuffer := make([]byte, r.blockSizeInBytes)
	n, err := r.vhdReader.ReadBytes(blockDataByteOffset, blockDataBuffer)
	if err == io.ErrUnexpectedEOF {
		return blockDataBuffer[:n], nil
	}

	if err != nil {
		return nil, NewDataReadError(blockIndex, footer.DiskTypeDynamic, err)
	}

	return blockDataBuffer, nil
}
