package footer

// ParseError is the error type representing disk footer parse error.
//
type ParseError struct {
	FooterField string
	err         error
}

// Error returns the string representation  of the ParseError instance.
//
func (e *ParseError) Error() string {
	return "Parse footer field" + " '" + e.FooterField + "' failed: " + e.err.Error()
}

// GetInnerErr returns the inner error, this method satisfies InnerErr interface
//
func (e *ParseError) GetInnerErr() error {
	return e.err
}

// NewParseError returns a new ParseError instance.
// The parameter footerField represents the field in the footer that failed to parse
// The parameter err is the underlying error for parse failure.
//
func NewParseError(footerField string, err error) error {
	return &ParseError{
		FooterField: footerField,
		err:         err,
	}
}
