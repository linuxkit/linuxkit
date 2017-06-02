package main

import (
	"os"
)

func homeDir() string {
	return os.Getenv("USERPROFILE")
}
