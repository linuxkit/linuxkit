package main

import (
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

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
	v2, err := isCgroupV2()
	if err != nil {
		log.Fatalf("Cannot determine cgroup version: %v", err)
	}
	msg := "cgroup v1"
	if v2 {
		msg = "cgroup v2"
	}
	log.Printf("Using %s", msg)

	// did we choose to run in debug mode? If so, runc will be in debug, and all messages will go to stdout/stderr in addition to the log
	var runcDebugMode, runcConsoleMode bool
	dt, err := os.ReadFile("/proc/cmdline")
	if err != nil {
		log.Fatalf("error reading /proc/cmdline: %v", err)
	}

	debugLogger := log.New()
	debugLogger.Level = log.InfoLevel

	for _, s := range strings.Fields(string(dt)) {
		if s == "linuxkit.runc_debug=1" {
			runcDebugMode = true
			debugLogger.Level = log.DebugLevel
		}
		if s == "linuxkit.runc_console=1" {
			runcConsoleMode = true
		}
	}

	for _, file := range files {
		name := file.Name()
		path := filepath.Join(rootPath, name)
		log.Printf("%s %s: from %s", serviceType, name, path)

		runtimeConfig := getRuntimeConfig(path)

		if err := prepareFilesystem(path, runtimeConfig); err != nil {
			log.Printf("Error preparing %s: %v", name, err)
			status = 1
			continue
		}
		debugLogger.Debugf("%s %s: creating", serviceType, name)
		pidfile := filepath.Join(tmpdir, name)
		cmdArgs := []string{"create", "--bundle", path, "--pid-file", pidfile, name}
		if runcDebugMode {
			cmdArgs = append([]string{"--debug"}, cmdArgs...)
		}
		cmd := exec.Command(runcBinary, cmdArgs...)

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

		// if in console mode, send output to stdout/stderr instead of the log
		// do not try io.MultiWriter(os.Stdout, stdout) as console messages will hang.
		// it is not clear why, but since this is all for debugging anyways, it doesn't matter
		// much.
		if runcConsoleMode {
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}

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

		debugLogger.Debugf("%s %s: preparing", serviceType, name)
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

		debugLogger.Debugf("%s %s: starting", serviceType, name)
		cmdArgs = []string{"start", name}
		if runcDebugMode {
			cmdArgs = append([]string{"--debug"}, cmdArgs...)
		}
		cmd = exec.Command(runcBinary, cmdArgs...)
		cmd.Stdout = stdout
		cmd.Stderr = stderr

		if err := cmd.Run(); err != nil {
			log.Printf("Error starting %s: %v", name, err)
			status = 1
			continue
		}

		debugLogger.Debugf("%s %s: waiting for completion", serviceType, name)
		_ = <-waitFor

		debugLogger.Debugf("%s %s: cleaning up", serviceType, name)
		cleanup(path)
		_ = os.Remove(pidfile)

		// ideally we want to use io.MultiWriter here, sending one stream to stdout/stderr, another to the log
		// however, this hangs if we do, due to a runc bug, see https://github.com/opencontainers/runc/issues/1721#issuecomment-366315563
		// once that is fixed, this can be cleaned up
		logger.Dump(stdoutLog)
		logger.Dump(stderrLog)
		debugLogger.Debugf("%s %s: complete", serviceType, name)
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
