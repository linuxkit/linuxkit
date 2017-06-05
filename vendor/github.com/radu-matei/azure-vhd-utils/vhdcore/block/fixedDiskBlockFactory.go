package block

import (
	"log"
	"math"

	"github.com/radu-matei/azure-vhd-utils/vhdcore"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/common"
)

// FixedDiskBlockFactory is a type which is used for following purposes
// To create a Block instance representing a fixed disk block
// To get the number of blocks in the fixed disk
// To get the block size of the block in fixed disk
// To get a Sector instance representing sector of fixed disk's block
// To get the logical footer range of the fixed disk
//
type FixedDiskBlockFactory struct {
	params           *FactoryParams
	sectorFactory    *SectorFactory
	blockDataReader  DataReader
	blockCount       int64
	blockSize        int64
	extraBlockIndex  *int64
	cachedFixedBlock *Block
}

// NewFixedDiskBlockFactoryWithDefaultBlockSize creates a FixedDiskBlockFactory instance which can
// be used to create a Block object representing fixed disk block of default size 512 KB.
// parameter params contains header, footer of the fixed disk and reader to read the disk.
//
func NewFixedDiskBlockFactoryWithDefaultBlockSize(params *FactoryParams) *FixedDiskBlockFactory {
	return NewFixedDiskBlockFactory(params, vhdcore.VhdDefaultBlockSize)
}

// NewFixedDiskBlockFactory creates a FixedDiskBlockFactory instance which can be used to create a
// Block objects representing fixed disk block of a specific size, parameter params contains header,
// footer of the fixed disk and reader to read the disk, parameter blockSize represents the size
// of blocks in the fixed disk
//
func NewFixedDiskBlockFactory(params *FactoryParams, blockSize int64) *FixedDiskBlockFactory {
	blockFactory := &FixedDiskBlockFactory{params: params}

	// VirtualSize is the current size of the fixed disk in bytes.
	c := float64(blockFactory.params.VhdFooter.VirtualSize) / float64(blockSize)
	cf := int64(math.Floor(c))
	cc := int64(math.Ceil(c))
	if cf < cc {
		blockFactory.extraBlockIndex = &cf
	} else {
		blockFactory.extraBlockIndex = nil
	}
	blockFactory.blockCount = cc
	blockFactory.blockSize = blockSize
	blockFactory.sectorFactory = NewSectorFactory(blockFactory.params.VhdReader,
		func(blockIndex uint32) bool {
			return blockIndex != vhdcore.VhdNoDataInt
		},
		func(blockIndex uint32) int64 {
			return int64(blockIndex) * blockSize
		},
	)
	blockFactory.blockDataReader = NewFixedDiskBlockReader(blockFactory.params.VhdReader, uint32(blockSize))
	return blockFactory
}

// GetBlockCount returns the number of blocks in the fixed disk.
//
func (f *FixedDiskBlockFactory) GetBlockCount() int64 {
	return f.blockCount
}

// GetBlockSize returns the size of the block in bytes of the fixed disk.
//
func (f *FixedDiskBlockFactory) GetBlockSize() int64 {
	return f.blockSize
}

// GetFooterRange returns the logical range of the footer of the fixed disk, logical range of footer
// is the absolute start and end byte offset of the footer.
//
func (f *FixedDiskBlockFactory) GetFooterRange() *common.IndexRange {
	footerStartIndex := f.params.VhdReader.Size - vhdcore.VhdFooterSize
	return common.NewIndexRangeFromLength(footerStartIndex, vhdcore.VhdFooterSize)
}

// Create returns an instance of Block which represents a fixed disk block, the parameter blockIndex
// identifies the block.
//
func (f *FixedDiskBlockFactory) Create(blockIndex uint32) (*Block, error) {
	if f.cachedFixedBlock == nil || f.cachedFixedBlock.BlockIndex != blockIndex {
		var logicalRange *common.IndexRange
		if f.extraBlockIndex != nil && *f.extraBlockIndex == int64(blockIndex) {
			logicalRange = f.getExtraBlockLogicalRange()
		} else {
			logicalRange = common.NewIndexRangeFromLength(int64(blockIndex)*f.blockSize, f.blockSize)
		}

		f.cachedFixedBlock = &Block{
			BlockIndex:      blockIndex,
			LogicalRange:    logicalRange,
			VhdUniqueID:     f.params.VhdFooter.UniqueID,
			BitMap:          nil, // Bitmap applies to dynamic and differentials disks
			BlockDataReader: f.blockDataReader,
		}

		f.cachedFixedBlock.IsEmpty = blockIndex == vhdcore.VhdNoDataInt
	}
	return f.cachedFixedBlock, nil
}

// GetSector returns an instance of Sector in a fixed disk, parameter block describes the block containing the
// sector, the parameter sectorIndex identifies the sector in the block. This function return error if the sector
// cannot be created due to any read error or if the requested sector index is invalid.
//
func (f *FixedDiskBlockFactory) GetSector(block *Block, sectorIndex uint32) (*Sector, error) {
	blockIndex := block.BlockIndex
	if block.IsEmpty {
		return f.sectorFactory.CreateEmptySector(blockIndex, sectorIndex), nil
	}

	return f.sectorFactory.Create(block, sectorIndex)
}

// getExtraBlockLogicalRange returns the IndexRange representing the additional block if any. Additional block
// is the last block whose size < FixedDiskBlockFactory.BlockSize
//
func (f *FixedDiskBlockFactory) getExtraBlockLogicalRange() *common.IndexRange {
	if f.extraBlockIndex == nil {
		log.Panicf("Unexpected state, extraBlockIndex not set")
	}

	startIndex := *(f.extraBlockIndex) * f.blockSize
	size := f.params.VhdReader.Size - startIndex - vhdcore.VhdFooterSize
	return common.NewIndexRangeFromLength(startIndex, size)
}
