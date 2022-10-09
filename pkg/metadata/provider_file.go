package main

import (
	"os"
)

type fileProvider string

func (p fileProvider) String() string {
	return string(p)
}

func (p fileProvider) Probe() bool {
	_, err := os.Stat(string(p))
	return err == nil
}

func (p fileProvider) Extract() ([]byte, error) {
	return os.ReadFile(string(p))
}
