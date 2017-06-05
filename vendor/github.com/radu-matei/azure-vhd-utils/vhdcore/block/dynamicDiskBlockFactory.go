package block

import (
	"github.com/radu-matei/azure-vhd-utils/vhdcore"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/block/bitmap"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/common"
)

// DynamicDiskBlockFactory is a type which is used for following purposes
// To create a Block instance representing a dynamic disk block
// To get the number of blocks in the dynamic disk
// To get the block size of the block in dynamic disk
// To get a Sector instance representing sector of dynamic disk's block
// To get the logical footer range of fixed disk generated from the dynamic disk
//
type DynamicDiskBlockFactory struct {
	params             *FactoryParams
	bitmapFactory      *bitmap.Factory
	sectorFactory      *SectorFactory
	blockDataReader    DataReader
	cachedDynamicBlock *Block
}

// NewDynamicDiskFactory creates a DynamicDiskBlockFactory instance which can be used to create a
// Block objects representing dynamic disk block of a size specified in header BlockSize field
// parameter params contains header, footer, BAT of dynamic disk and reader to read the disk.
//
func NewDynamicDiskFactory(params *FactoryParams) *DynamicDiskBlockFactory {
	blockFactory := &DynamicDiskBlockFactory{params: params}

	blockFactory.bitmapFactory = bitmap.NewFactory(blockFactory.params.VhdReader,
		blockFactory.params.BlockAllocationTable)

	blockFactory.sectorFactory = NewSectorFactory(blockFactory.params.VhdReader,
		blockFactory.params.BlockAllocationTable.HasData,
		blockFactory.params.BlockAllocationTable.GetBlockDataAddress)

	blockFactory.blockDataReader = NewDynamicDiskBlockReader(blockFactory.params.VhdReader,
		blockFactory.params.BlockAllocationTable,
		blockFactory.params.VhdHeader.BlockSize)
	return blockFactory
}

// GetBlockCount returns the number of blocks in the dynamic disk.
//
func (f *DynamicDiskBlockFactory) GetBlockCount() int64 {
	return int64(f.params.BlockAllocationTable.BATEntriesCount)
}

// GetBlockSize returns the size of the 'data section' of block in bytes in the dynamic disk.
//
func (f *DynamicDiskBlockFactory) GetBlockSize() int64 {
	return int64(f.params.VhdHeader.BlockSize)
}

// GetFooterRange returns the logical range of the footer when converting this dynamic vhd to fixed
// logical range of footer is the absolute start and end byte offset of the footer.
//
func (f *DynamicDiskBlockFactory) GetFooterRange() *common.IndexRange {
	return common.NewIndexRangeFromLength(f.GetBlockCount()*f.GetBlockSize(), vhdcore.VhdFooterSize)
}

// Create returns an instance of Block which represents a dynamic disk block, the parameter blockIndex
// identifies the block. This function return error if the block cannot be created due to any read error.
//
func (f *DynamicDiskBlockFactory) Create(blockIndex uint32) (*Block, error) {
	if f.cachedDynamicBlock == nil || f.cachedDynamicBlock.BlockIndex != blockIndex {
		logicalRange := common.NewIndexRangeFromLength(int64(blockIndex)*f.GetBlockSize(), f.GetBlockSize())
		f.cachedDynamicBlock = &Block{
			BlockIndex:      blockIndex,
			LogicalRange:    logicalRange,
			VhdUniqueID:     f.params.VhdFooter.UniqueID,
			BlockDataReader: f.blockDataReader,
		}

		if f.params.BlockAllocationTable.HasData(blockIndex) {
			var err error
			f.cachedDynamicBlock.BitMap, err = f.bitmapFactory.Create(blockIndex)
			if err != nil {
				return nil, err
			}

			f.cachedDynamicBlock.IsEmpty = false
		} else {
			f.cachedDynamicBlock.BitMap = nil
			f.cachedDynamicBlock.IsEmpty = true
		}
	}

	return f.cachedDynamicBlock, nil
}

// GetSector returns an instance of Sector in a dynamic disk, parameter block object identifying the block
// containing the sector, the parameter sectorIndex identifies the sector in the block. This function return
// error if the sector cannot be created due to any read error or if the requested sector index is invalid.
//
func (f *DynamicDiskBlockFactory) GetSector(block *Block, sectorIndex uint32) (*Sector, error) {
	blockIndex := block.BlockIndex
	if block.IsEmpty {
		return f.sectorFactory.CreateEmptySector(blockIndex, sectorIndex), nil
	}

	return f.sectorFactory.Create(block, sectorIndex)
}

// GetBitmapFactory returns an instance of BitmapFactory that can be used to create the bitmap of a block
// by reading block from dynamic disk.
//
func (f *DynamicDiskBlockFactory) GetBitmapFactory() *bitmap.Factory {
	return f.bitmapFactory
}
