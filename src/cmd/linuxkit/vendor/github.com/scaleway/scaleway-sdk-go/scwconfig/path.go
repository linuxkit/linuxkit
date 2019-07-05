package scwconfig

import (
	"errors"
	"os"
	"path/filepath"
)

const (
	unixHomeDirEnv    = "HOME"
	windowsHomeDirEnv = "USERPROFILE"
	xdgConfigDirEnv   = "XDG_CONFIG_HOME"

	defaultConfigFileName = "config.yaml"
)

var (
	// ErrNoHomeDir errors when no user directory is found
	ErrNoHomeDir = errors.New("user home directory not found")
)

func inConfigFile() string {
	v2path, exist := GetConfigV2FilePath()
	if exist {
		return "in config file " + v2path
	}
	return ""
}

// GetConfigV2FilePath returns the path to the Scaleway CLI config file
func GetConfigV2FilePath() (string, bool) {
	configDir, err := GetScwConfigDir()
	if err != nil {
		return "", false
	}
	return filepath.Join(configDir, defaultConfigFileName), true
}

// GetConfigV1FilePath returns the path to the Scaleway CLI config file
func GetConfigV1FilePath() (string, bool) {
	path, err := GetHomeDir()
	if err != nil {
		return "", false
	}
	return filepath.Join(path, ".scwrc"), true
}

// GetScwConfigDir returns the path to scw config folder
func GetScwConfigDir() (string, error) {
	if xdgPath := os.Getenv(xdgConfigDirEnv); xdgPath != "" {
		return filepath.Join(xdgPath, "scw"), nil
	}

	homeDir, err := GetHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".config", "scw"), nil
}

// GetHomeDir returns the path to your home directory
func GetHomeDir() (string, error) {
	switch {
	case os.Getenv(unixHomeDirEnv) != "":
		return os.Getenv(unixHomeDirEnv), nil
	case os.Getenv(windowsHomeDirEnv) != "":
		return os.Getenv(windowsHomeDirEnv), nil
	default:
		return "", ErrNoHomeDir
	}
}
