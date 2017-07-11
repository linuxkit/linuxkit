package block

import "fmt"

// SectorReadError is the error type representing block's sector read error.
//
type SectorReadError struct {
	BlockIndex  uint32
	SectorIndex uint32
	err         error
}

// Error returns the string representation  of the SectorReadError instance.
//
func (e *SectorReadError) Error() string {
	return fmt.Sprint("Read sector '%d' of block '%d' failed: %s", e.SectorIndex, e.BlockIndex, e.err)
}

// NewSectorReadError returns a new SectorReadError instance.
// The parameter blockIndex represents index of the block
// The parameter sectorIndex represents index of the sector within the block
// The parameter err is the underlying read error.
//
func NewSectorReadError(blockIndex, sectorIndex uint32, err error) error {
	return &SectorReadError{
		BlockIndex:  blockIndex,
		SectorIndex: sectorIndex,
		err:         err,
	}
}
