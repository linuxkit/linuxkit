package main

import (
	"strconv"
	"strings"
)

// This function parses the "size" parameter of a disk specification
// and returns the size in MB. The "size" parameter defaults to GB, but
// the unit can be explicitly set with either a G (for GB) or M (for
// MB). It returns the disk size in MB.
func getDiskSizeMB(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	sz := len(s)
	if strings.HasSuffix(s, "M") {
		return strconv.Atoi(s[:sz-1])
	}
	if strings.HasSuffix(s, "G") {
		s = s[:sz-1]
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	return 1024 * i, nil
}
