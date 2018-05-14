package main

// #include <linux/random.h>
//
// int rndaddentropy = RNDADDENTROPY;
//
import "C"

import (
	"errors"
)

// No standard RNG instruction on arm64
func initDRNG(ctx *rng) bool {
	ctx.disabled = true
	return false
}

func readDRNG(_ *rng) (uint64, error) {
	return 0, errors.New("No randomness available")
}
