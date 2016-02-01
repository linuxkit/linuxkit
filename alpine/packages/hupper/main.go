package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

var (
	path       string
	huppidfile string
	pidfile    string
)

func init() {
	flag.StringVar(&path, "path", "/Database/branch/master/watch/com.docker.driver.amd64-linux.node/etc.node/docker.node/daemon.json.node/tree.live", "path of the file to watch")
	flag.StringVar(&huppidfile, "huppidfile", "/run/docker.pid", "pidfile for process to signal")
	flag.StringVar(&pidfile, "pidfile", "/run/hupper.pid", "my pidfile")
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	pid := os.Getpid()
	pidbytes := []byte(strconv.Itoa(pid))
	_ = ioutil.WriteFile(pidfile, pidbytes, 0644)

	watch, err := os.Open(path)
	if err != nil {
		log.Fatalln("Failed to open file", path, err)
	}
	// 43 bytes is the record size of the watch
	buf := make([]byte, 43)
	// initial state
	_, err = watch.Read(buf)
	if err != nil && err != io.EOF {
		log.Fatalln("Error reading watch file", err)
	}
	for {
		_, err := watch.Read(buf)
		if err != nil && err != io.EOF {
			log.Fatalln("Error reading watch file", err)
		}
		// a few changes eg debug do not require a daemon restart
		// however at present we cannot check changes, and most do
		restart := true
		if restart {
			cmd := exec.Command("service", "docker", "restart")
			// not much we can do if it does not restart
			_ = cmd.Run()
		} else {
			bytes, err := ioutil.ReadFile(huppidfile)
			if err != nil {
				continue
			}
			pidstring := string(bytes[:])
			pid, err := strconv.Atoi(pidstring)
			if err != nil {
				continue
			}
			syscall.Kill(pid, syscall.SIGHUP)
		}
	}
}
