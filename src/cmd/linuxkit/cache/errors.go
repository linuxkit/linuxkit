package cache

import (
	"fmt"
)

type noReferenceError struct {
	reference string
}

func (e *noReferenceError) Error() string {
	return fmt.Sprintf("no such reference: %s", e.reference)
}
