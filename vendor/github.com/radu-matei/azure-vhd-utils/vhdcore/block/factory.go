package block

import (
	"github.com/radu-matei/azure-vhd-utils/vhdcore/common"
)

// Factory interface that all block factories specific to disk type (fixed,
// dynamic, differencing) needs to satisfy.
//
type Factory interface {
	GetBlockCount() int64
	GetBlockSize() int64
	Create(blockIndex uint32) (*Block, error)
	GetFooterRange() *common.IndexRange
	GetSector(block *Block, sectorIndex uint32) (*Sector, error)
}
