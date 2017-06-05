package header

// ParseError is the error type representing disk header parse error.
//
type ParseError struct {
	HeaderField string
	err         error
}

// Error returns the string representation of the ParseError instance.
//
func (e *ParseError) Error() string {
	return "Parse header field" + " '" + e.HeaderField + "' failed: " + e.err.Error()
}

// GetInnerErr returns the inner error, this method satisfies InnerErr interface
//
func (e *ParseError) GetInnerErr() error {
	return e.err
}

// NewParseError returns a new ParseError instance.
// The parameter headerField represents the field in the header that failed to parse
// The parameter err is the underlying error for parse failure.
//
func NewParseError(headerField string, err error) error {
	return &ParseError{
		HeaderField: headerField,
		err:         err,
	}
}
