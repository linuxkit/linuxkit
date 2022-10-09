package main

import (
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
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
	files, err := os.ReadDir(rootPath)
	if err != nil {
		log.Fatalf("Cannot read files in %s: %v", rootPath, err)
	}

	tmpdir, err := os.MkdirTemp("", filepath.Base(rootPath))
	if err != nil {
		log.Fatalf("Cannot create temporary directory: %v", err)
	}

	// need to set ourselves as a child subreaper or we cannot wait for runc as reparents to init
	if err := setSubreaper(1); err != nil {
		log.Fatalf("Cannot set as subreaper: %v", err)
	}

	status := 0

	logDir := path.Join(logDirBase, serviceType)
	varLogLink := path.Join(varLogDir, serviceType)

	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Cannot create log directory %s: %v", logDir, err)
	}

	logger := GetLog(logDir)

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

		stdoutLog := serviceType + "." + name + ".out"
		stdout, err := logger.Open(stdoutLog)
		if err != nil {
			log.Printf("Error opening stdout log connection: %v", err)
			status = 1
			continue
		}
		defer stdout.Close()

		stderrLog := serviceType + "." + name
		stderr, err := logger.Open(stderrLog)
		if err != nil {
			log.Printf("Error opening stderr log connection: %v", err)
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
		pf, err := os.ReadFile(pidfile)
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

		// ideally we want to use io.MultiWriter here, sending one stream to stdout/stderr, another to the log
		// however, this hangs if we do, due to a runc bug, see https://github.com/opencontainers/runc/issues/1721#issuecomment-366315563
		// once that is fixed, this can be cleaned up
		logger.Dump(stdoutLog)
		logger.Dump(stderrLog)
	}

	_ = os.RemoveAll(tmpdir)

	// make sure the link exists from /var/log/onboot -> /run/log/onboot
	logger.Symlink(varLogLink)

	return status
}

// setSubreaper copied directly from https://github.com/opencontainers/runc/blob/b23315bdd99c388f5d0dd3616188729c5a97484a/libcontainer/system/linux.go#L88
// to avoid version and vendor conflict issues
func setSubreaper(i int) error {
	return unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, uintptr(i), 0, 0, 0)
}
