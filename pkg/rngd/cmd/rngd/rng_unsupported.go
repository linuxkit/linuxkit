// +build !linux !amd64,!arm64

package main

import "errors"

func initRand() bool {
	return false
}

func rand() (uint64, error) {
	return 0, errors.New("No rng available")
}
