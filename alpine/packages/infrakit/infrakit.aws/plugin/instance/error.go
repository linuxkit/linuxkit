package instance

import (
	"fmt"
)

// ErrUnexpectedResponse is error when the API call violates contract and has unexpected results.
type ErrUnexpectedResponse struct{}

func (e *ErrUnexpectedResponse) Error() string {
	return "Unexpected"
}

// ErrInvalidRequest is error for invalid request from the client.
type ErrInvalidRequest struct{}

func (e *ErrInvalidRequest) Error() string {
	return fmt.Sprintf("Invalid request")
}

// ErrExceededAttempts is error when attempts have exceeded given threshold.
type ErrExceededAttempts struct {
	attempts int
}

func (e *ErrExceededAttempts) Error() string {
	return fmt.Sprintf("Max attempts exceeded: %d", e.attempts)
}
