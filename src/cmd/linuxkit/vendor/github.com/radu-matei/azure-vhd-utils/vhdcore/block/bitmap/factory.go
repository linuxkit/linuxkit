package bitmap

import (
	"github.com/radu-matei/azure-vhd-utils/vhdcore/bat"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/reader"
)

// Factory type is used to create BitMap instance by reading 'bitmap section' of a block.
//
type Factory struct {
	vhdReader            *reader.VhdReader
	blockAllocationTable *bat.BlockAllocationTable
}

// NewFactory creates a new instance of Factory, which can be used to create a BitMap instance by reading
// the 'bitmap section' of a block. vhdReader is the reader to read the disk, blockAllocationTable wraps
// the disk's BAT table, which has one entry per block, this is used to retrieve the absolute offset to
// the beginning of the 'bitmap section' of a block and the size of the 'bitmap section'.
//
func NewFactory(vhdReader *reader.VhdReader, blockAllocationTable *bat.BlockAllocationTable) *Factory {
	return &Factory{vhdReader: vhdReader, blockAllocationTable: blockAllocationTable}
}

// Create creates a BitMap instance by reading block's 'bitmap section', block is the index of the
// block entry in the BAT whose 'bitmap section' needs to be read.
// This function return error if any error occurs while reading or parsing the block's bitmap.
//
func (f *Factory) Create(blockIndex uint32) (*BitMap, error) {
	bitmapAbsoluteByteOffset := f.blockAllocationTable.GetBitmapAddress(blockIndex)
	bitmapSizeInBytes := f.blockAllocationTable.GetBitmapSizeInBytes()
	bitmapBytes := make([]byte, bitmapSizeInBytes)
	if _, err := f.vhdReader.ReadBytes(bitmapAbsoluteByteOffset, bitmapBytes); err != nil {
		return nil, NewParseError(blockIndex, err)
	}
	return NewBitMapFromByteSlice(bitmapBytes), nil
}
