package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	runcBinary   = "/usr/bin/runc"
	onbootPath   = "/containers/onboot"
	shutdownPath = "/containers/onshutdown"
)

func main() {
	// try to work out how we are being called
	command := os.Args[0]
	if len(os.Args) > 1 {
		command = os.Args[1]
	}
	var path = onbootPath
	switch {
	case strings.Contains(command, "boot"):
		path = onbootPath
	case strings.Contains(command, "shutdown"):
		path = shutdownPath
	}

	// do nothing if the path does not exist
	if _, err := os.Stat(path); err != nil && os.IsNotExist(err) {
		os.Exit(0)
	}

	// get files; note ReadDir already sorts them
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatalf("Cannot read files in %s: %v", path, err)
	}

	status := 0

	for _, file := range files {
		name := file.Name()
		fullPath := filepath.Join(path, name)
		if err := prepare(fullPath); err != nil {
			log.Printf("Error preparing %s: %v", name, err)
			status = 1
			continue
		}
		cmd := exec.Command(runcBinary, "run", "--bundle", fullPath, name)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Printf("Error running %s: %v", name, err)
			status = 1
		}
	}

	os.Exit(status)
}
