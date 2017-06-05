package block

import (
	"github.com/radu-matei/azure-vhd-utils/vhdcore"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/block/bitmap"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/common"
)

// DifferencingDiskBlockFactory is a type which is used for following purposes
// To create a Block instance representing a differencing disk block
// To get the number of blocks in the differencing disk
// To get the block size of the block in differencing disk
// To get a Sector instance representing sector of differencing disk's block
// To get the logical footer range of fixed disk generated from the differencing disk and it's parents.
//
type DifferencingDiskBlockFactory struct {
	params          *FactoryParams
	bitmapFactory   *bitmap.Factory
	sectorFactory   *SectorFactory
	blockDataReader DataReader
	cachedBlock     *Block
}

// NewDifferencingDiskBlockFactory creates a DifferencingDiskBlockFactory instance which can be used to
// create a Block objects representing differential disk block of a size specified in header BlockSize
// field parameter vhdFile represents the differencing disk.
//
func NewDifferencingDiskBlockFactory(params *FactoryParams) *DifferencingDiskBlockFactory {
	blockFactory := &DifferencingDiskBlockFactory{params: params}

	blockFactory.bitmapFactory = bitmap.NewFactory(blockFactory.params.VhdReader,
		blockFactory.params.BlockAllocationTable)

	blockFactory.sectorFactory = NewSectorFactory(blockFactory.params.VhdReader,
		blockFactory.params.BlockAllocationTable.HasData,
		blockFactory.params.BlockAllocationTable.GetBlockDataAddress)

	blockFactory.blockDataReader = NewDifferencingDiskBlockReader(blockFactory.params.VhdReader,
		blockFactory.params.BlockAllocationTable,
		blockFactory.params.VhdHeader.BlockSize)

	return blockFactory
}

// GetBlockCount returns the number of blocks in the differential disk.
//
func (f *DifferencingDiskBlockFactory) GetBlockCount() int64 {
	return int64(f.params.BlockAllocationTable.BATEntriesCount)
}

// GetBlockSize returns the size of the 'data section' of block in bytes in the differential disk.
//
func (f *DifferencingDiskBlockFactory) GetBlockSize() int64 {
	return int64(f.params.VhdHeader.BlockSize)
}

// GetFooterRange returns the logical range of the footer when converting this differential vhd to
// fixed logical range of footer is the absolute start and end byte offset of the footer.
//
func (f *DifferencingDiskBlockFactory) GetFooterRange() *common.IndexRange {
	return common.NewIndexRangeFromLength(f.GetBlockCount()*f.GetBlockSize(), vhdcore.VhdFooterSize)
}

// Create returns an instance of Block which represents a differencing disk block, the parameter blockIndex
// identifies the block. If the block to be read is marked as empty in the differencing disk BAT then this
// method will query parent disk for the same block. This function return error if the block cannot be created
// due to any read error.
//
func (f *DifferencingDiskBlockFactory) Create(blockIndex uint32) (*Block, error) {
	if !f.params.BlockAllocationTable.HasData(blockIndex) {
		if f.cachedBlock == nil || f.cachedBlock.BlockIndex != blockIndex {
			var err error
			f.cachedBlock, err = f.params.ParentBlockFactory.Create(blockIndex)
			if err != nil {
				return nil, err
			}
		}

		return f.cachedBlock, nil
	}

	if f.cachedBlock == nil || f.cachedBlock.BlockIndex != blockIndex {
		logicalRange := common.NewIndexRangeFromLength(int64(blockIndex)*f.GetBlockSize(), f.GetBlockSize())
		f.cachedBlock = &Block{
			BlockIndex:      blockIndex,
			LogicalRange:    logicalRange,
			VhdUniqueID:     f.params.VhdFooter.UniqueID,
			IsEmpty:         false,
			BlockDataReader: f.blockDataReader,
		}

		var err error
		f.cachedBlock.BitMap, err = f.bitmapFactory.Create(blockIndex)
		if err != nil {
			return nil, err
		}
	}

	return f.cachedBlock, nil
}

// GetSector returns an instance of Sector in a differencing disk, parameter block object identifies the block
// containing the sector, the parameter sectorIndex identifies the sector in the block. If the sector to be
// read is marked as empty in the block's bitmap then this method will query parent disk for the same sector.
// This function return error if the sector cannot be created due to any read error or if the requested sector
// index is invalid.
//
func (f *DifferencingDiskBlockFactory) GetSector(block *Block, sectorIndex uint32) (*Sector, error) {
	blockIndex := block.BlockIndex
	if block.IsEmpty {
		return f.sectorFactory.CreateEmptySector(blockIndex, sectorIndex), nil
	}

	if block.BitMap != nil {
		s, err := block.BitMap.Get(int32(sectorIndex))
		if err != nil {
			return nil, err
		}

		if s {
			return f.sectorFactory.Create(block, sectorIndex)
		}
	}

	blockInParentDisk, err := f.params.ParentBlockFactory.Create(blockIndex)
	if err != nil {
		return nil, err
	}

	return f.params.ParentBlockFactory.GetSector(blockInParentDisk, sectorIndex)
}

// GetBitmapFactory returns an instance of BitmapFactory that can be used to create the bitmap of a block
// by reading block from differencing disk.
//
func (f *DifferencingDiskBlockFactory) GetBitmapFactory() *bitmap.Factory {
	return f.bitmapFactory
}
