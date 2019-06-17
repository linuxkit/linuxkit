package auth

import "net/http"

type token struct {
	accessKey string
	secretKey string
}

// XAuthTokenHeader is Scaleway standard auth header
const XAuthTokenHeader = "X-Auth-Token"

// NewToken create a token authentication from an
// access key and a secret key
func NewToken(accessKey, secretKey string) *token {
	return &token{accessKey: accessKey, secretKey: secretKey}
}

// Headers returns headers that must be add to the http request
func (t *token) Headers() http.Header {
	headers := http.Header{}
	headers.Set(XAuthTokenHeader, t.secretKey)
	return headers
}

// AnonymizedHeaders returns an anonymized version of Headers()
// This method could be use for logging purpose.
func (t *token) AnonymizedHeaders() http.Header {
	headers := http.Header{}
	var secret string

	switch {
	case len(t.secretKey) == 0:
		secret = ""
	case len(t.secretKey) > 8:
		secret = t.secretKey[0:8] + "-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
	default:
		secret = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
	}
	headers.Set(XAuthTokenHeader, secret)
	return headers
}
