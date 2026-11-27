package main

import (
	"context"
	"os"
)

type fileProvider string

func (p fileProvider) String() string {
	return string(p)
}

func (p fileProvider) Probe(ctx context.Context) bool {
	_, err := os.Stat(string(p))
	return err == nil
}

func (p fileProvider) Extract(ctx context.Context) ([]byte, error) {
	return os.ReadFile(string(p))
}
