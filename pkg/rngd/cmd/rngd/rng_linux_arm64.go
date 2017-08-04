package main

// #include <linux/random.h>
//
// int rndaddentropy = RNDADDENTROPY;
//
import "C"

import (
	"errors"
)

// No standard RNG on arm64

func initRand() bool {
	return false
}

func rand() (uint64, error) {
	return 0, errors.New("No randomness available")
}
