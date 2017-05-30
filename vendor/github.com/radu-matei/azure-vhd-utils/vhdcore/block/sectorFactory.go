package block

import (
	"fmt"

	"github.com/radu-matei/azure-vhd-utils/vhdcore"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/reader"
)

// SectorFactory type is used to create Sector instance by reading 512 byte sector from block's 'data section'.
//
type SectorFactory struct {
	vhdReader       *reader.VhdReader
	blockHasData    func(uint32) bool
	getBlockAddress func(uint32) int64
	emptySectorBuf  []byte
}

// NewSectorFactory creates a new instance of SectorFactory, which can be used to create Sector instances
// by reading 512 byte sector from block's 'data section'
// vhdReader is the reader to be used to read the sector, blockHasData is a function which can be used to
// check a block is empty by providing block identifier, getBlockAddress is a function which can be used
// to fetch the absolute byte offset of a block by providing block identifier.
//
func NewSectorFactory(vhdReader *reader.VhdReader, blockHasData func(uint32) bool, getBlockAddress func(uint32) int64) *SectorFactory {
	return &SectorFactory{
		vhdReader:       vhdReader,
		blockHasData:    blockHasData,
		getBlockAddress: getBlockAddress,
	}
}

// Create creates an instance of Sector by reading a 512 byte sector from the 'data section' of a block.
// block describes the block containing the sector, sectorIndex identifies the sector to read.
// This function return error if requested sector is invalid or in case of any read error.
//
func (f *SectorFactory) Create(block *Block, sectorIndex uint32) (*Sector, error) {
	if int64(sectorIndex) > block.GetSectorCount() {
		return nil, fmt.Errorf("Total sectors: %d, Requested Sectors: %d", block.GetSectorCount(), sectorIndex)
	}

	blockIndex := block.BlockIndex
	if !f.blockHasData(blockIndex) {
		return f.CreateEmptySector(blockIndex, sectorIndex), nil
	}

	blockDataSectionByteOffset := f.getBlockAddress(blockIndex)
	sectorByteOffset := blockDataSectionByteOffset + vhdcore.VhdSectorLength*int64(sectorIndex)
	sectorBuf := make([]byte, vhdcore.VhdSectorLength)
	if _, err := f.vhdReader.ReadBytes(sectorByteOffset, sectorBuf); err != nil {
		return nil, NewSectorReadError(blockIndex, sectorIndex, err)
	}

	return &Sector{
		BlockIndex:  blockIndex,
		SectorIndex: int64(sectorIndex),
		Data:        sectorBuf,
	}, nil
}

// CreateEmptySector creates an instance of Sector representing empty sector. The Data property of this sector
// will be a slice of 512 bytes filled with zeros.
//
func (f *SectorFactory) CreateEmptySector(blockIndex, sectorIndex uint32) *Sector {
	if f.emptySectorBuf == nil {
		f.emptySectorBuf = make([]byte, vhdcore.VhdSectorLength)
	}

	return &Sector{
		BlockIndex:  blockIndex,
		SectorIndex: int64(sectorIndex),
		Data:        f.emptySectorBuf,
	}
}
