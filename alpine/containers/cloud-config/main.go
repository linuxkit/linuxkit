package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var (
	configDir string
	sysctlDir string
)

func init() {
	flag.StringVar(&configDir, "configDir", "/etc/sysctl.d", "directory with config files")
	flag.StringVar(&sysctlDir, "sysctlLocalDir", "/cloud_sysctl", "directory to set sysctl values")
}

func sysctl(line []byte) error {
	// parse line into a string of expected form X.Y.Z=VALUE
	sysctlLineKV := strings.Split(string(line[:]), "=")
	// trim any extra whitespace
	sysctlSetting, sysctlValue := strings.Trim(sysctlLineKV[0], " "), strings.Trim(sysctlLineKV[1], " ")
	sysctlFile := filepath.Join(sysctlDir, filepath.Join(strings.Split(sysctlSetting, ".")...))
	err := os.MkdirAll(filepath.Dir(sysctlFile), 0)
	if err != nil {
		return fmt.Errorf("Cannot open directory for %s: %s", sysctlFile, err)
	}
	file, err := os.OpenFile(sysctlFile, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("Cannot open %s: %s", sysctlFile, err)
	}
	defer file.Close()
	_, err = file.Write([]byte(sysctlValue))
	if err != nil {
		return fmt.Errorf("Cannot write to %s: %s", sysctlFile, err)
	}
	fmt.Printf("*** successfully set %s to %s ***\n", sysctlFile, sysctlValue)
	return nil
}

func main() {
	flag.Parse()

	files, err := ioutil.ReadDir(configDir)
	if err != nil {
		log.Fatalf("Cannot read directory %s: %s", configDir, err)
	}

	for _, file := range files {
		contents, err := ioutil.ReadFile(filepath.Join(configDir, file.Name()))
		if err != nil {
			log.Fatalf("Cannot read file %s: %s", file.Name(), err)
		}
		lines := bytes.Split(contents, []byte("\n"))
		for _, line := range lines {
			if len(line) == 0 {
				continue
			}
			err = sysctl(line)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
