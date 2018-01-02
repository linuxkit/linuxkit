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
	flag.StringVar(&sysctlDir, "sysctlDir", "/proc/sys", "mount point for sysctls")
}

func sysctl(line []byte) error {
	sysctlLineTrimmed := strings.TrimSpace(string(line[:]))
	// skip any commented lines
	if len(sysctlLineTrimmed) >= 1 && (sysctlLineTrimmed[:1] == "#" || sysctlLineTrimmed[:1] == ";") {
		return nil
	}
	// parse line into a string of expected form X.Y.Z=VALUE
	sysctlLineKV := strings.Split(sysctlLineTrimmed, "=")
	if len(sysctlLineKV) != 2 {
		return fmt.Errorf("Cannot parse %s", sysctlLineTrimmed)
	}
	// trim any extra whitespace
	sysctlSetting, sysctlValue := strings.TrimSpace(sysctlLineKV[0]), strings.TrimSpace(sysctlLineKV[1])
	sysctlFile := filepath.Join(sysctlDir, filepath.Join(strings.FieldsFunc(sysctlSetting, splitKv)...))
	file, err := os.OpenFile(sysctlFile, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("Cannot open %s: %s", sysctlFile, err)
	}
	defer file.Close()
	_, err = file.Write([]byte(sysctlValue))
	if err != nil {
		return fmt.Errorf("Cannot write to %s: %s", sysctlFile, err)
	}
	return nil
}

func splitKv(r rune) bool {
	return r == '.' || r == '/'
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
				log.Println(fmt.Errorf("WARN: %v", err))
			}
		}
	}
}
