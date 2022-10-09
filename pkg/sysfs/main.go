package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	configDir string
	sysfsDir  string
)

func init() {
	flag.StringVar(&configDir, "configDir", "/etc/sysfs.d", "directory with config files")
	flag.StringVar(&sysfsDir, "sysfsDir", "/sys", "mount point for sysfs")
}

func sysfs(line []byte) error {
	// parse line into a string of expected form X/Y/Z=VALUE
	sysfsLineKV := strings.Split(string(line[:]), "=")
	if len(sysfsLineKV) != 2 {
		if len(sysfsLineKV) >= 1 && len(sysfsLineKV[0]) >= 1 && strings.Trim(sysfsLineKV[0], " ")[:1] == "#" {
			return nil
		}
		return fmt.Errorf("Cannot parse %s", string(line))
	}
	// trim any extra whitespace
	sysfsSetting, sysfsValue := strings.Trim(sysfsLineKV[0], " "), strings.Trim(sysfsLineKV[1], " ")
	sysfsFile := filepath.Join(sysfsDir, sysfsSetting)
	file, err := os.OpenFile(sysfsFile, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("Cannot open %s: %s", sysfsFile, err)
	}
	defer file.Close()
	_, err = file.Write([]byte(sysfsValue))
	if err != nil {
		return fmt.Errorf("Cannot write to %s: %s", sysfsFile, err)
	}
	return nil
}

func main() {
	flag.Parse()

	files, err := os.ReadDir(configDir)
	if err != nil {
		log.Fatalf("Cannot read directory %s: %s", configDir, err)
	}

	for _, file := range files {
		contents, err := os.ReadFile(filepath.Join(configDir, file.Name()))
		if err != nil {
			log.Fatalf("Cannot read file %s: %s", file.Name(), err)
		}
		lines := bytes.Split(contents, []byte("\n"))
		for _, line := range lines {
			if len(line) == 0 {
				continue
			}
			err = sysfs(line)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
