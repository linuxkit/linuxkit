package cache

import (
	"io"
	"os"
	"path/filepath"
)

// ReadOrComputeBlob stores in the cache io.Readers that are heavy to compute.
func (p *Provider) ReadOrComputeBlob(key string, compute func() (io.ReadCloser, error)) (io.ReadCloser, error) {
	cachePath := filepath.Join(p.blobs, key)
	if _, err := os.Stat(cachePath); err != nil {
		extracted, err := compute()
		if err != nil {
			return nil, err
		}
		defer extracted.Close()

		if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
			return nil, err
		}

		if err := writeToFile(cachePath, extracted); err != nil {
			return nil, err
		}
	}
	return os.Open(cachePath)
}

func writeToFile(dst string, src io.ReadCloser) error {
	file, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, src)
	return err
}
