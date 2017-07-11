package block

import (
	"github.com/radu-matei/azure-vhd-utils/vhdcore/bat"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/footer"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/header"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/reader"
)

// FactoryParams represents type of the parameter for different disk block
// factories.
//
type FactoryParams struct {
	VhdFooter            *footer.Footer
	VhdHeader            *header.Header
	BlockAllocationTable *bat.BlockAllocationTable
	VhdReader            *reader.VhdReader
	ParentBlockFactory   Factory
}
