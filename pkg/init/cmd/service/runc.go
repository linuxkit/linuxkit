package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/containerd/containerd/sys"
)

const (
	runcBinary = "/usr/bin/runc"
)

func runcInit(rootPath string) int {
	// do nothing if the path does not exist
	if _, err := os.Stat(rootPath); err != nil && os.IsNotExist(err) {
		return 0
	}

	// get files; note ReadDir already sorts them
	files, err := ioutil.ReadDir(rootPath)
	if err != nil {
		log.Fatalf("Cannot read files in %s: %v", rootPath, err)
	}

	tmpdir, err := ioutil.TempDir("", filepath.Base(rootPath))
	if err != nil {
		log.Fatalf("Cannot create temporary directory: %v", err)
	}

	// need to set ourselves as a child subreaper or we cannot wait for runc as reparents to init
	if err := sys.SetSubreaper(1); err != nil {
		log.Fatalf("Cannot set as subreaper: %v", err)
	}

	status := 0

	for _, file := range files {
		name := file.Name()
		path := filepath.Join(rootPath, name)

		runtimeConfig := getRuntimeConfig(path)

		if err := prepareFilesystem(path, runtimeConfig); err != nil {
			log.Printf("Error preparing %s: %v", name, err)
			status = 1
			continue
		}
		pidfile := filepath.Join(tmpdir, name)
		cmd := exec.Command(runcBinary, "create", "--bundle", path, "--pid-file", pidfile, name)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Printf("Error creating %s: %v", name, err)
			status = 1
			// skip cleanup on error for debug
			continue
		}
		pf, err := ioutil.ReadFile(pidfile)
		if err != nil {
			log.Printf("Cannot read pidfile: %v", err)
			status = 1
			continue
		}
		pid, err := strconv.Atoi(string(pf))
		if err != nil {
			log.Printf("Cannot parse pid from pidfile: %v", err)
			status = 1
			continue
		}

		if err := prepareProcess(pid, runtimeConfig); err != nil {
			log.Printf("Cannot prepare process: %v", err)
			status = 1
			continue
		}

		waitFor := make(chan *os.ProcessState)
		go func() {
			// never errors in Unix
			p, _ := os.FindProcess(pid)
			state, err := p.Wait()
			if err != nil {
				log.Printf("Process wait error: %v", err)
			}
			waitFor <- state
		}()

		cmd = exec.Command(runcBinary, "start", name)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			log.Printf("Error starting %s: %v", name, err)
			status = 1
			continue
		}

		_ = <-waitFor

		cleanup(path)
		_ = os.Remove(pidfile)
	}

	_ = os.RemoveAll(tmpdir)

	return status
}
