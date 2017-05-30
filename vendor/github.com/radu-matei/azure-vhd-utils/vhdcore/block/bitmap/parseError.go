package bitmap

import "fmt"

// ParseError is the error type representing parsing error of a block's bitmap.
//
type ParseError struct {
	BlockIndex uint32
	err        error
}

// Error returns the string representation  of the BitmapParseError instance.
//
func (e *ParseError) Error() string {
	return fmt.Sprintf("Parse Bitmap section of block '%d' failed: "+e.err.Error(), e.BlockIndex)
}

// GetInnerErr returns the inner error, this method satisfies InnerErr interface
//
func (e *ParseError) GetInnerErr() error {
	return e.err
}

// NewParseError returns a new ParseError instance.
// The parameter blockIndex represents index of the block whose bitmap failed to parse
// The parameter err is the underlying error for parse failure.
//
func NewParseError(blockIndex uint32, err error) error {
	return &ParseError{
		BlockIndex: blockIndex,
		err:        err,
	}
}
