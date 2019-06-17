package scw

import (
	"context"
)

// RequestOption is a function that applies options to a ScalewayRequest.
type RequestOption func(*requestSettings)

// WithContext request option sets the context of a ScalewayRequest
func WithContext(ctx context.Context) RequestOption {
	return func(s *requestSettings) {
		s.ctx = ctx
	}
}

// WithAllPages aggregate all pages in the response of a List request.
// Will error when pagination is not supported on the request.
func WithAllPages() RequestOption {
	return func(s *requestSettings) {
		s.allPages = true
	}
}

type requestSettings struct {
	ctx      context.Context
	allPages bool
}

func newRequestSettings() *requestSettings {
	return &requestSettings{}
}

func (s *requestSettings) apply(opts []RequestOption) {
	for _, opt := range opts {
		opt(s)
	}
}

func (s *requestSettings) validate() SdkError {
	// nothing so far
	return nil
}
