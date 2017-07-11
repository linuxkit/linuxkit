package block

import (
	"github.com/radu-matei/azure-vhd-utils/vhdcore/bat"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/footer"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/reader"
)

// DifferencingDiskBlockReader type satisfies BlockDataReader interface,
// implementation of BlockDataReader::Read by this type can read the 'data' section
// of a differencing disk's block.
//
type DifferencingDiskBlockReader struct {
	vhdReader            *reader.VhdReader
	blockAllocationTable *bat.BlockAllocationTable
	blockSizeInBytes     uint32
	emptyBlockData       []byte
}

// NewDifferencingDiskBlockReader create a new instance of DifferencingDiskBlockReader which read
// the 'data' section of differencing disk block.
// The parameter vhdReader is the reader to read the disk
// The parameter blockAllocationTable represents the disk's BAT
// The parameter blockSizeInBytes is the size of the differencing disk block
//
func NewDifferencingDiskBlockReader(vhdReader *reader.VhdReader, blockAllocationTable *bat.BlockAllocationTable, blockSizeInBytes uint32) *DifferencingDiskBlockReader {
	return &DifferencingDiskBlockReader{
		vhdReader:            vhdReader,
		blockAllocationTable: blockAllocationTable,
		blockSizeInBytes:     blockSizeInBytes,
		emptyBlockData:       nil,
	}
}

// Read reads the data in a block of a differencing disk
// The parameter block represents the block whose 'data' section to read
//
func (r *DifferencingDiskBlockReader) Read(block *Block) ([]byte, error) {
	blockIndex := block.BlockIndex
	if !r.blockAllocationTable.HasData(blockIndex) {
		if r.emptyBlockData == nil {
			r.emptyBlockData = make([]byte, r.blockSizeInBytes)
		}
		return r.emptyBlockData, nil
	}

	blockDataBuffer := make([]byte, r.blockSizeInBytes)
	index := 0
	sectorCount := block.GetSectorCount()
	for i := int64(0); i < sectorCount; i++ {
		sector, err := block.GetSector(uint32(i))
		if err != nil {
			return nil, NewDataReadError(blockIndex, footer.DiskTypeDifferencing, err)
		}

		n := copy(blockDataBuffer[index:], sector.Data)
		index += n
	}

	return blockDataBuffer, nil
}
