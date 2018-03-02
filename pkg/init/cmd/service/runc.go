package main

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"

	"github.com/containerd/containerd/sys"
	log "github.com/sirupsen/logrus"
)

const (
	runcBinary = "/usr/bin/runc"
	logDirBase = "/run/log/"
	varLogDir  = "/var/log"
)

func dumpFile(w io.Writer, filePath string) error {
	f, err := os.OpenFile(filePath, os.O_RDONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)

	return err
}

func runcInit(rootPath, serviceType string) int {
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

	logDir := path.Join(logDirBase, serviceType)
	varLogLink := path.Join(varLogDir, serviceType)

	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Cannot create log directory %s: %v", logDir, err)
	}

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

		// stream stdout and stderr to respective files
		// ideally we want to use io.MultiWriter here, sending one stream to stdout/stderr, another to the files
		// however, this hangs if we do, due to a runc bug, see https://github.com/opencontainers/runc/issues/1721#issuecomment-366315563
		// once that is fixed, this can be cleaned up
		stdoutFile := filepath.Join(logDir, serviceType+"."+name+".out.log")
		stdout, err := os.OpenFile(stdoutFile, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Printf("Error opening stdout log file: %v", err)
			status = 1
			continue
		}
		defer stdout.Close()

		stderrFile := filepath.Join(logDir, serviceType+"."+name+".err.log")
		stderr, err := os.OpenFile(stderrFile, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			log.Printf("Error opening stderr log file: %v", err)
			status = 1
			continue
		}
		defer stderr.Close()

		cmd.Stdout = stdout
		cmd.Stderr = stderr

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
		cmd.Stdout = stdout
		cmd.Stderr = stderr

		if err := cmd.Run(); err != nil {
			log.Printf("Error starting %s: %v", name, err)
			status = 1
			continue
		}

		_ = <-waitFor

		cleanup(path)
		_ = os.Remove(pidfile)

		// dump the log file outputs to os.Stdout/os.Stderr
		if err = dumpFile(os.Stdout, stdoutFile); err != nil {
			log.Printf("Error writing stdout of onboot service %s to console: %v", name, err)
		}
		if err = dumpFile(os.Stderr, stderrFile); err != nil {
			log.Printf("Error writing stderr of onboot service %s to console: %v", name, err)
		}
	}

	_ = os.RemoveAll(tmpdir)

	// make sure the link exists from /var/log/onboot -> /run/log/onboot
	if err := os.MkdirAll(varLogDir, 0755); err != nil {
		log.Printf("Error creating secondary log directory %s: %v", varLogDir, err)
	} else if err := os.Symlink(logDir, varLogLink); err != nil && !os.IsExist(err) {
		log.Printf("Error creating symlink from %s to %s: %v", varLogLink, logDir, err)
	}

	return status
}
