package main

import (
	"io"
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

		stdoutFile := filepath.Join("/var/log", "onboot-"+name+".out.log")

		stdoutFd, stdoutFdErr := os.OpenFile(stdoutFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		defer stdoutFd.Close()

		var stdoutCopyErr error

		if stdoutFdErr == nil {
			stdoutIn, _ := cmd.StdoutPipe()
			stdoutWriter := io.MultiWriter(os.Stdout, stdoutFd)
			go func() {
				_, stdoutCopyErr = io.Copy(stdoutWriter, stdoutIn)
			}()
		} else {
			log.Printf("Could not open %s for writing: %v", stdoutFile, stdoutFdErr)
			cmd.Stdout = os.Stdout
		}

		stderrFile := filepath.Join("/var/log", "onboot-"+name+".err.log")

		stderrFd, stderrFdErr := os.OpenFile(stderrFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		defer stderrFd.Close()

		var stderrCopyErr error

		if stderrFdErr == nil {
			stderrIn, _ := cmd.StderrPipe()
			stderrWriter := io.MultiWriter(os.Stderr, stderrFd)
			go func() {
				_, stderrCopyErr = io.Copy(stderrWriter, stderrIn)
			}()
		} else {
			log.Printf("Could not open %s for writing: %v", stderrFile, stderrFdErr)
			cmd.Stderr = os.Stderr
		}

		if err := cmd.Start(); err != nil {
			log.Printf("Error starting %s: %v", name, err)
			status = 1
			// skip cleanup on error for debug
			continue
		}

		if err := cmd.Wait(); err != nil {
			log.Printf("Error waiting for completion of %s: %v", name, err)
			status = 1
			continue
		}

		if stdoutFdErr == nil && stdoutCopyErr != nil {
			log.Printf("Could not stream stdout for %s: %v", name, stdoutCopyErr)
			status = 1
			continue
		}

		if stderrFdErr == nil && stderrCopyErr != nil {
			log.Printf("Could not stream stderr for %s: %v", name, stderrCopyErr)
			status = 1
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
