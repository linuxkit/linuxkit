package bat

import (
	"math"

	"github.com/radu-matei/azure-vhd-utils/vhdcore"
)

// BlockAllocationTable type represents the Block Allocation Table (BAT) of the disk, BAT served as
// index to access the disk's blocks.
// A block is a unit of expansion for dynamic and differencing hard disks. All blocks within a given
// image must be the same size.
// The number of entries in the BAT is the number of blocks needed to store the contents of the disk
// when fully expanded. Each entry in this table is the absolute sector offset to a block. Each entry
// is four bytes long. if the disk is not fully expanded then, even though BAT has entries reserved
// for unexpanded blocks, the corresponding block will not exists. All such unused table entries
// are initialized to 0xFFFFFFFF.
// A block consists of two sections 'data section' and 'block bitmap section'. The 'BlockSize' field
// of the disk header is the size of the 'data section' of the block, it does not include the size of
// the 'block bitmap section'. Each bit in the bitmap indicates the state of the corresponding sector
// in 'data section', 1 indicates sector contains valid data, 0 indicates the sector have never been
// modified.
//
type BlockAllocationTable struct {
	BATEntriesCount uint32
	BAT             []uint32
	blockSize       uint32
}

// NewBlockAllocationTable creates an instance of BlockAllocationTable, BAT is the block allocation table,
// each entry in this table is the absolute sector offset to a block, blockSize is the size of block's
// 'data section' in bytes.
//
func NewBlockAllocationTable(blockSize uint32, bat []uint32) *BlockAllocationTable {
	return &BlockAllocationTable{BATEntriesCount: uint32(len(bat)), blockSize: blockSize, BAT: bat}
}

// GetBitmapSizeInBytes returns the size of the 'block bitmap section' that stores the state
// of the sectors in block's 'data section'. This means the number of bits in the bitmap is equivalent
// to the number of sectors in 'data section', dividing this number by 8 will yield the number of bytes
// required to store the bitmap.
// As per vhd specification sectors per block must be power of two. The sector length is always 512 bytes.
// This means the block size will be power of two as well e.g. 512 * 2^3, 512 * 2^4, 512 * 2^5 etc..
//
func (b *BlockAllocationTable) GetBitmapSizeInBytes() int32 {
	return int32(b.blockSize / uint32(vhdcore.VhdSectorLength) / 8)
}

// GetSectorPaddedBitmapSizeInBytes returns the size of the 'block bitmap section' in bytes which is
// padded to a 512-byte sector boundary. The bitmap of a block is always padded to a 512-byte sector
// boundary.
func (b *BlockAllocationTable) GetSectorPaddedBitmapSizeInBytes() int32 {
	sectorSizeInBytes := float64(vhdcore.VhdSectorLength)
	bitmapSizeInBytes := float64(b.GetBitmapSizeInBytes())
	return int32(math.Ceil(bitmapSizeInBytes/sectorSizeInBytes) * sectorSizeInBytes)
}

// GetBitmapAddress returns the address of the 'block bitmap section' of a given block. Address is the
// absolute byte offset of the 'block bitmap section'. A block consists of 'block bitmap section' and
// 'data section'
//
func (b *BlockAllocationTable) GetBitmapAddress(blockIndex uint32) int64 {
	return int64(b.BAT[blockIndex]) * vhdcore.VhdSectorLength
}

// GetBlockDataAddress returns the address of the 'data section' of a given block. Address is the absolute
// byte offset of the 'data section'. A block consists of 'block bitmap section' and 'data section'
//
func (b *BlockAllocationTable) GetBlockDataAddress(blockIndex uint32) int64 {
	return b.GetBitmapAddress(blockIndex) + int64(b.GetSectorPaddedBitmapSizeInBytes())
}

// HasData returns true if the given block has not yet expanded hence contains no data.
//
func (b *BlockAllocationTable) HasData(blockIndex uint32) bool {
	return blockIndex != vhdcore.VhdNoDataInt && b.BAT[blockIndex] != vhdcore.VhdNoDataInt
}
