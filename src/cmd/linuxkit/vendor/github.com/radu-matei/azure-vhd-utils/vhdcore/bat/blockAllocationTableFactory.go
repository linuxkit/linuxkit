package bat

import (
	"github.com/radu-matei/azure-vhd-utils/vhdcore/header"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/reader"
)

// BlockAllocationTableFactory type is used to create BlockAllocationTable instance by reading BAT
// section of the disk which follows the header
//
type BlockAllocationTableFactory struct {
	vhdReader *reader.VhdReader
	vhdHeader *header.Header
}

// NewBlockAllocationFactory creates a new instance of BlockAllocationTableFactory, which can be used
// to create BlockAllocationTable instance by reading BAT section of the Vhd.
// vhdReader is the reader to be used to read the entry, vhdHeader is the header structure representing
// the disk header.
//
func NewBlockAllocationFactory(vhdReader *reader.VhdReader, vhdHeader *header.Header) *BlockAllocationTableFactory {
	return &BlockAllocationTableFactory{
		vhdReader: vhdReader,
		vhdHeader: vhdHeader,
	}
}

// Create creates a BlockAllocationTable instance by reading the BAT section of the disk.
// This function return error if any error occurs while reading or parsing the BAT entries.
//
func (f *BlockAllocationTableFactory) Create() (*BlockAllocationTable, error) {
	var err error
	batEntriesCount := f.vhdHeader.MaxTableEntries
	batEntryOffset := f.vhdHeader.TableOffset
	bat := make([]uint32, batEntriesCount)
	for i := uint32(0); i < batEntriesCount; i++ {
		bat[i], err = f.vhdReader.ReadUInt32(batEntryOffset)
		if err != nil {
			return nil, NewBlockAllocationTableParseError(i, err)
		}

		batEntryOffset += 4
	}
	return NewBlockAllocationTable(f.vhdHeader.BlockSize, bat), nil
}
