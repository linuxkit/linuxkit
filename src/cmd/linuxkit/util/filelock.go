package util

import (
	"os"
)

type FileLock struct {
	file *os.File
}
