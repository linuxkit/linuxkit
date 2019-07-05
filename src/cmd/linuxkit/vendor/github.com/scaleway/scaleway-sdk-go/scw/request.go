package scw

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	"github.com/scaleway/scaleway-sdk-go/internal/auth"
	"github.com/scaleway/scaleway-sdk-go/internal/errors"
	"github.com/scaleway/scaleway-sdk-go/utils"
)

// ScalewayRequest contains all the contents related to performing a request on the Scaleway API.
type ScalewayRequest struct {
	Method  string
	Path    string
	Headers http.Header
	Query   url.Values
	Body    io.Reader
	Ctx     context.Context
}

// getAllHeaders constructs a http.Header object and aggregates all headers into the object.
func (req *ScalewayRequest) getAllHeaders(token auth.Auth, userAgent string, anonymized bool) http.Header {
	var allHeaders http.Header
	if anonymized {
		allHeaders = token.AnonymizedHeaders()
	} else {
		allHeaders = token.Headers()
	}

	allHeaders.Set("User-Agent", userAgent)
	if req.Body != nil {
		allHeaders.Set("content-type", "application/json")
	}
	for key, value := range req.Headers {
		for _, v := range value {
			allHeaders.Set(key, v)
		}
	}

	return allHeaders
}

// getURL constructs a URL based on the base url and the client.
func (req *ScalewayRequest) getURL(baseURL string) (*url.URL, SdkError) {
	url, err := url.Parse(baseURL + req.Path)
	if err != nil {
		return nil, errors.New("invalid url %s: %s", baseURL+req.Path, err)
	}
	url.RawQuery = req.Query.Encode()

	return url, nil
}

// SetBody json marshal the given body and write the json content type
// to the request. It also catches when body is a file.
func (req *ScalewayRequest) SetBody(body interface{}) error {
	var contentType string
	var content io.Reader

	switch b := body.(type) {
	case *utils.File:
		contentType = b.ContentType
		content = b.Content
	default:
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		contentType = "application/json"
		content = bytes.NewReader(buf)
	}

	req.Headers.Set("Content-Type", contentType)
	req.Body = content

	return nil
}
