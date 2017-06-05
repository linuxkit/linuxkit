package block

import (
	"fmt"

	"github.com/radu-matei/azure-vhd-utils/vhdcore"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/block/bitmap"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/common"
)

// Block type represents Block of a vhd. A block of a dynamic or differential vhd starts with a
// 'bitmap' section followed by the 'data' section, in case of fixed vhd the entire block is used
// to store the 'data'.
//
type Block struct {
	// BlockIndex is the index of the block, block indices are consecutive values starting from 0
	// for the first block.
	BlockIndex uint32
	// IsEmpty is true if the block is empty, for dynamic and differencing disk the BAT entry of an
	// empty block contains 0xFFFFFFFF.
	IsEmpty bool
	// BitMap represents 'bitmap section' of the block. Each bit in the bitmap represents the state
	// of a sector in the block.
	//
	// This field is always nil for fixed disk. For Dynamic and differencing disk this field is set
	// if the block is not marked as empty in the BAT.
	//
	// The dynamic and differencing subsystem reads the sector marked as dirty from the current disk,
	// if a sector is marked as clean and if the current disk disk is dynamic then the sector will be
	// read from the parent disk.
	BitMap *bitmap.BitMap
	// LogicalRange holds the absolute start and end byte offset of the block's 'data' in the converted
	// fixed disk. When converting dynamic and differential vhd to fixed vhd, we place all block's 'data'
	// consecutively starting at byte offset 0 of the fixed disk.
	LogicalRange *common.IndexRange
	// VhdUniqueId holds the unique identifier of the vhd that the block belongs to. The vhd unique
	// identifier is stored in the vhd footer unique id field.
	VhdUniqueID *common.UUID
	// BlockData is a byte slice containing the block's 'data'
	blockData []byte
	// blockDataReader is the reader for reading the block's 'data' from the disk.
	BlockDataReader DataReader
	// sectorProvider enables retrieving the sector
	blockFactory Factory
}

// Data returns the block data, the content of entire block in case of fixed vhd and the content
// of block's data section in case of dynamic and differential vhd.
//
func (b *Block) Data() ([]byte, error) {
	if b.blockData == nil {
		var err error
		b.blockData, err = b.BlockDataReader.Read(b)
		if err != nil {
			return nil, err
		}
	}
	return b.blockData, nil
}

// GetSector returns an instance of Sector representing a sector with the given Id in this block.
// The parameter sectorIndex is the index of the sector in this block to read.
//
func (b *Block) GetSector(sectorIndex uint32) (*Sector, error) {
	return b.blockFactory.GetSector(b, sectorIndex)
}

// GetSectorCount returns the number of sectors in the block.
//
func (b *Block) GetSectorCount() int64 {
	return b.LogicalRange.Length() / vhdcore.VhdSectorLength
}

// String returns formatted representation of the block
// This satisfies Stringer interface.
//
func (b *Block) String() string {
	return fmt.Sprintf("Block:%d", b.BlockIndex)
}
