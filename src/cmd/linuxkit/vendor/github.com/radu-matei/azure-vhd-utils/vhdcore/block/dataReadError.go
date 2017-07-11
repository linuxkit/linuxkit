package block

import (
	"fmt"

	"github.com/radu-matei/azure-vhd-utils/vhdcore/footer"
)

// DataReadError is the error type representing block data read error.
//
type DataReadError struct {
	BlockIndex uint32
	DiskType   footer.DiskType
	err        error
}

// Error returns the string representation  of the BlockDataReadError instance.
//
func (e *DataReadError) Error() string {
	return fmt.Sprintf("Error in Reading block  '%d', DiskType - %s  : %s", e.BlockIndex, e.DiskType, e.err)
}

// GetInnerErr returns the inner error, this method satisfies InnerErr interface
//
func (e *DataReadError) GetInnerErr() error {
	return e.err
}

// NewDataReadError returns a new DataReadError instance.
// The parameter blockIndex represents index of the block whose bitmap failed to parse
// The parameter err is the underlying error for parse failure.
//
func NewDataReadError(blockIndex uint32, diskType footer.DiskType, err error) error {
	return &DataReadError{
		BlockIndex: blockIndex,
		DiskType:   diskType,
		err:        err,
	}
}
