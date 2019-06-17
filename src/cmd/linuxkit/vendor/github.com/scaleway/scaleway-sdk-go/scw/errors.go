package scw

import (
	"fmt"
)

// SdkError is a base interface for all Scaleway SDK errors.
type SdkError interface {
	Error() string
	IsScwSdkError()
}

// ResponseError is an error type for the Scaleway API
type ResponseError struct {
	// Message is a human-friendly error message
	Message string `json:"message"`

	// Type is a string code that defines the kind of error
	Type string `json:"type,omitempty"`

	// Fields contains detail about validation error
	Fields map[string][]string `json:"fields,omitempty"`

	// StatusCode is the HTTP status code received
	StatusCode int `json:"-"`

	// Status is the HTTP status received
	Status string `json:"-"`
}

func (e *ResponseError) Error() string {
	s := fmt.Sprintf("scaleway-sdk-go: http error %s", e.Status)

	if e.Message != "" {
		s = fmt.Sprintf("%s: %s", s, e.Message)
	}

	if len(e.Fields) > 0 {
		s = fmt.Sprintf("%s: %v", s, e.Fields)
	}

	return s
}

// IsScwSdkError implement SdkError interface
func (e *ResponseError) IsScwSdkError() {}
