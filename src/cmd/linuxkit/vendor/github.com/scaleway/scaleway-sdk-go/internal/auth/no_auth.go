package auth

import "net/http"

type noAuth struct {
}

// NewNoAuth return an auth with no authentication method
func NewNoAuth() *noAuth {
	return &noAuth{}
}

func (t *noAuth) Headers() http.Header {
	return http.Header{}
}

func (t *noAuth) AnonymizedHeaders() http.Header {
	return http.Header{}
}
