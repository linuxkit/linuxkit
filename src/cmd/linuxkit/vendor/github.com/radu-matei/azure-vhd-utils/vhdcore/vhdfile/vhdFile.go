package vhdFile

import (
	"fmt"

	"github.com/radu-matei/azure-vhd-utils/vhdcore/bat"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/block"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/footer"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/header"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/reader"
)

// VhdFile represents a VHD.
//
type VhdFile struct {
	// Footer represents the disk's footer.
	Footer *footer.Footer
	// Header represents the disk's header, this field is nil for fixed VHD.
	// Only Dynamic and Differencing disk has header.
	Header *header.Header
	// BlockAllocationTable represents the table holding absolute offset to the first sector
	// of blocks in the disk. Only Dynamic and Differencing disk has BAT.
	BlockAllocationTable *bat.BlockAllocationTable
	// VhdReader is the reader that can be used to read the disk.
	VhdReader *reader.VhdReader
	// Parent represents the parent VHD of Differencing disk, this field is nil for fixed
	// and dynamic disk.
	Parent *VhdFile
}

// GetDiskType returns the type of the disk. Possible values are DiskTypeFixed, DiskTypeDynamic
// and DiskTypeDifferencing.
//
func (f *VhdFile) GetDiskType() footer.DiskType {
	return f.Footer.DiskType
}

// GetBlockFactory returns a BlockFactory instance that can be used to create Block instances
// that represents blocks in the disk.
//
func (f *VhdFile) GetBlockFactory() (block.Factory, error) {
	params := &block.FactoryParams{
		VhdHeader: f.Header,
		VhdFooter: f.Footer,
		VhdReader: f.VhdReader,
	}

	switch f.GetDiskType() {
	case footer.DiskTypeFixed:
		return block.NewFixedDiskBlockFactoryWithDefaultBlockSize(params), nil

	case footer.DiskTypeDynamic:
		params.BlockAllocationTable = f.BlockAllocationTable
		return block.NewDynamicDiskFactory(params), nil

	case footer.DiskTypeDifferencing:
		params.BlockAllocationTable = f.BlockAllocationTable
		parentVhdFile := f.Parent
		if parentVhdFile.GetDiskType() == footer.DiskTypeFixed {
			params.ParentBlockFactory = block.NewFixedDiskBlockFactory(
				&block.FactoryParams{
					VhdHeader: parentVhdFile.Header,
					VhdFooter: parentVhdFile.Footer,
					VhdReader: parentVhdFile.VhdReader,
				},
				int64(f.Header.BlockSize)) // The block-size of parent FixedDisk and this DifferentialDisk will be same.

		} else {
			var err error
			params.ParentBlockFactory, err = parentVhdFile.GetBlockFactory()
			if err != nil {
				return nil, err
			}
		}
		return block.NewDifferencingDiskBlockFactory(params), nil
	}

	return nil, fmt.Errorf("Unsupported disk format: %d", f.GetDiskType())
}

// GetIdentityChain returns VHD identity chain, for differencing disk this will be a slice with
// unique ids of this and all it's ancestor disks. For fixed and dynamic disk, this will be a
// slice with one entry representing disk's unique id.
//
func (f *VhdFile) GetIdentityChain() []string {
	ids := []string{f.Footer.UniqueID.String()}
	for p := f.Parent; p != nil; p = p.Parent {
		ids = append(ids, p.Footer.UniqueID.String())
	}

	return ids
}
