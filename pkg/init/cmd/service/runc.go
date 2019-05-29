package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"

	"github.com/opencontainers/runc/libcontainer/system"
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
	if err := system.SetSubreaper(1); err != nil {
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
		pidfile := filepath.Join(tmpdir, name)
		if err := runcInitService(logger, serviceType, name, pidfile, path); err != nil {
			log.Printf(err.Error())
			status = 1
		}
	}

	_ = os.RemoveAll(tmpdir)

	// make sure the link exists from /var/log/onboot -> /run/log/onboot
	logger.Symlink(varLogLink)

	return status
}

func runcInitService(logger Log, serviceType, name, pidfile, path string) error {
	runtimeConfig := getRuntimeConfig(path)

	if err := prepareFilesystem(path, runtimeConfig); err != nil {
		return fmt.Errorf("Error preparing %s: %v", name, err)
	}
	cmd := exec.Command(runcBinary, "create", "--bundle", path, "--pid-file", pidfile, name)

	stdoutLog := serviceType + "." + name + ".out"
	stdout, err := logger.Open(stdoutLog)
	if err != nil {
		return fmt.Errorf("Error opening stdout log connection: %v", err)
	}
	defer stdout.Close()

	stderrLog := serviceType + "." + name
	stderr, err := logger.Open(stderrLog)
	if err != nil {
		return fmt.Errorf("Error opening stderr log connection: %v", err)
	}
	defer stderr.Close()

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Error creating %s: %v", name, err)
	}
	pf, err := ioutil.ReadFile(pidfile)
	if err != nil {
		return fmt.Errorf("Cannot read pidfile: %v", err)
	}
	pid, err := strconv.Atoi(string(pf))
	if err != nil {
		return fmt.Errorf("Cannot parse pid from pidfile: %v", err)
	}

	if err := prepareProcess(pid, runtimeConfig); err != nil {
		return fmt.Errorf("Cannot prepare process: %v", err)
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
		return fmt.Errorf("Error starting %s: %v", name, err)
	}

	_ = <-waitFor

	cleanup(path)
	_ = os.Remove(pidfile)

	// ideally we want to use io.MultiWriter here, sending one stream to stdout/stderr, another to the log
	// however, this hangs if we do, due to a runc bug, see https://github.com/opencontainers/runc/issues/1721#issuecomment-366315563
	// once that is fixed, this can be cleaned up
	logger.Dump(stdoutLog)
	logger.Dump(stderrLog)

	return nil
}
