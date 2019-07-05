package utils

import "io"

// File is the structure used to receive / send a file from / to the API
type File struct {
	// Name of the file
	Name string `json:"name"`

	// ContentType used in the HTTP header `Content-Type`
	ContentType string `json:"content_type"`

	// Content of the file
	Content io.Reader `json:"content"`
}
