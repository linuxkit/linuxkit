package parentlocator

// ParseError is the error type representing disk header parent locator parse error.
//
type ParseError struct {
	LocatorField string
	err          error
}

// Error returns the string representation of the ParseError instance.
//
func (e *ParseError) Error() string {
	return "Parse parent locator field" + " '" + e.LocatorField + "' failed: " + e.err.Error()
}

// GetInnerErr returns the inner error, this method satisfies InnerErr interface
//
func (e *ParseError) GetInnerErr() error {
	return e.err
}

// NewParseError returns a new ParseError instance.
// The parameter headerField represents the field in the header parent locator that failed to parse
// The parameter err is the underlying error for parse failure.
//
func NewParseError(locatorField string, err error) error {
	return &ParseError{
		LocatorField: locatorField,
		err:          err,
	}
}
