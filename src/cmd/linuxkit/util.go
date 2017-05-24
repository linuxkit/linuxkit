package main

import (
	"os"
	"strconv"
	"strings"
)

func getStringValue(envKey string, flagVal string, defaultVal string) string {
	var res string

	// If defined, take the env variable
	if _, ok := os.LookupEnv(envKey); ok {
		res = os.Getenv(envKey)
	}

	// If a flag is specified, this value takes precedence
	// Ignore cases where the flag carries the default value
	if flagVal != "" && flagVal != defaultVal {
		res = flagVal
	}

	// if we still don't have a value, use the default
	if res == "" {
		res = defaultVal
	}
	return res
}

func getIntValue(envKey string, flagVal int, defaultVal int) int {
	var res int

	// If defined, take the env variable
	if _, ok := os.LookupEnv(envKey); ok {
		var err error
		res, err = strconv.Atoi(os.Getenv(envKey))
		if err != nil {
			res = 0
		}
	}

	// If a flag is specified, this value takes precedence
	// Ignore cases where the flag carries the default value
	if flagVal > 0 {
		res = flagVal
	}

	// if we still don't have a value, use the default
	if res == 0 {
		res = defaultVal
	}
	return res
}

func getBoolValue(envKey string, flagVal bool) bool {
	var res bool

	// If defined, take the env variable
	if _, ok := os.LookupEnv(envKey); ok {
		switch os.Getenv(envKey) {
		case "":
			res = false
		case "0":
			res = false
		case "false":
			res = false
		case "FALSE":
			res = false
		case "1":
			res = true
		default:
			// catches "true", "TRUE" or anything else
			res = true

		}
	}

	// If a flag is specified, this value takes precedence
	if res != flagVal {
		res = flagVal
	}

	return res
}

func stringToIntArray(l string, sep string) ([]int, error) {
	var err error
	if l == "" {
		return []int{}, err
	}
	s := strings.Split(l, sep)
	i := make([]int, len(s))
	for idx := range s {
		if i[idx], err = strconv.Atoi(s[idx]); err != nil {
			return nil, err
		}
	}
	return i, nil
}

// Parse a string which is either a number in MB, or a number with
// either M (for Megabytes) or G (for GigaBytes) as a suffix and
// returns the number in MB. Return 0 if string is empty.
func getDiskSizeMB(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	sz := len(s)
	if strings.HasSuffix(s, "G") {
		i, err := strconv.Atoi(s[:sz-1])
		if err != nil {
			return 0, err
		}
		return i * 1024, nil
	}
	if strings.HasSuffix(s, "M") {
		s = s[:sz-1]
	}
	return strconv.Atoi(s)
}
