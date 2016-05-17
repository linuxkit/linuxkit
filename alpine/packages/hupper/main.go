package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
)

var (
	paths      stringSlice
	huppidfile string
	pidfile    string
)

type stringSlice []string

func (s *stringSlice) String() string {
	return fmt.Sprintf("%s", *s)
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func init() {
	flag.Var(&paths, "path", "paths of the files to watch")
	flag.StringVar(&huppidfile, "huppidfile", "", "pidfile for process to signal")
	flag.StringVar(&pidfile, "pidfile", "", "my pidfile")
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	if len(paths) < 1 {
		log.Fatal("watch path not set")
	}

	if huppidfile == "" {
		log.Fatal("huppidfile not set")
	}

	if pidfile != "" {
		pid := os.Getpid()
		pidbytes := []byte(strconv.Itoa(pid))
		_ = ioutil.WriteFile(pidfile, pidbytes, 0644)
	}

	var wg sync.WaitGroup
	wg.Add(len(paths))

	for _, path := range paths {
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

		go func() {
			for {
				_, err := watch.Read(buf)
				if err != nil && err != io.EOF {
					log.Fatalln("Error reading watch file", err)
				}
				if err == io.EOF {
					continue
				}
				// a few changes eg debug do not require a daemon restart
				// however at present we cannot check changes, and most do
				restart := true
				if restart {
					log.Println("Restarting docker")
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
		}()
	}
	wg.Wait()
}
