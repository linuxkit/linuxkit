package main

import (
	"log"
)

func main() {
	if err := newCmd().Execute(); err != nil {
		log.Fatalf("error during command execution: %v", err)
	}
}
