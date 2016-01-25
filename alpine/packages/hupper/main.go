package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"syscall"
)

var (
	path    string
	pidfile string
)

func init() {
	flag.StringVar(&path, "path", "/Database/branch/master/watch/com.docker.driver.amd64-linux.node/etc.node/docker.node/daemon.json.node/tree.live", "path of the file to watch")
	flag.StringVar(&pidfile, "pidfile", "/run/docker.pid", "pidfile for process to signal")
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	watch, err := os.Open(path)
	if err != nil {
		log.Fatalln("Failed to open file", path, err)
	}
	// 43 bytes is the record size of the watch
	buf := make([]byte, 43)
	for {
		n, err := watch.Read(buf)
		if err != nil {
			log.Fatalln("Error reading watch file", err)
		}
		if n == 0 {
			continue
		}
		bytes, err := ioutil.ReadFile(pidfile)
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
