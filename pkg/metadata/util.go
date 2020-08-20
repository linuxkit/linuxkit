package main

import (
	"fmt"
	"io/ioutil"
	"os"
)

// ProviderUtil utility for providers
type ProviderUtil interface {
	WriteDataToFile(contentTrait string, permission os.FileMode, content string, dest string) error
	MakeFolder(folderTrait string, permission os.FileMode, dest string) error
}

// DefaultProviderUtil default method for provider utility, same as default interface/trait method in c#/rust
type DefaultProviderUtil struct {
	ProviderUtil
	ProviderShortName
}

// WriteDataToFile write data to file with permission
func (p *DefaultProviderUtil) WriteDataToFile(contentTrait string, permission os.FileMode, content string, dest string) error {
	if err := ioutil.WriteFile(dest, []byte(content), permission); err != nil {
		return fmt.Errorf("%s: failed to write %s: %s", p.ShortName(), contentTrait, err)
	}
	return nil
}

// MakeFolder make folder with permission
func (p *DefaultProviderUtil) MakeFolder(folderTrait string, permission os.FileMode, dest string) error {
	if err := os.Mkdir(dest, permission); err != nil {
		return fmt.Errorf("%s: failed to make folder for %s: %s", p.ShortName(), folderTrait, err)
	}
	return nil
}
