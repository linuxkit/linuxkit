package bat

import "fmt"

// BlockAllocationTableParseError is the error type representing BAT parse error.
//
type BlockAllocationTableParseError struct {
	BATItemIndex uint32
	err          error
}

// Error returns the string representation  of the BlockAllocationTableParseError instance.
//
func (e *BlockAllocationTableParseError) Error() string {
	return fmt.Sprintf("Parse BAT entry at '%d' failed: "+e.err.Error(), e.BATItemIndex)
}

// GetInnerErr returns the inner error, this method satisfies InnerErr interface
//
func (e *BlockAllocationTableParseError) GetInnerErr() error {
	return e.err
}

// NewBlockAllocationTableParseError returns a new BlockAllocationTableParseError instance.
// The parameter batItemIndex represents BAT index failed to parse
// The parameter err is the underlying error for parse failure.
//
func NewBlockAllocationTableParseError(batItemIndex uint32, err error) error {
	return &BlockAllocationTableParseError{
		BATItemIndex: batItemIndex,
		err:          err,
	}
}
