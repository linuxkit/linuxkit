package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	var socketLogPath string
	var socketQueryPath string
	var memlogdBundle string
	var pidFile string
	var detach bool
	flag.StringVar(&socketLogPath, "socket-log", "/var/run/linuxkit-external-logging.sock", "path to fd logging socket. Created and passed to logging container. Existing socket will be removed.")
	flag.StringVar(&socketQueryPath, "socket-query", "/var/run/memlogdq.sock", "path to query socket. Created and passed to logging container. Existing socket will be removed.")
	flag.StringVar(&memlogdBundle, "bundle", "/containers/init/memlogd", "runc bundle with memlogd")
	flag.StringVar(&pidFile, "pid-file", "/run/memlogd.pid", "path to pid file")
	flag.BoolVar(&detach, "detach", true, "detach from subprocess")
	flag.Parse()

	laddr := net.UnixAddr{socketLogPath, "unixgram"}
	os.Remove(laddr.Name) // remove existing socket
	lconn, err := net.ListenUnixgram("unixgram", &laddr)
	if err != nil {
		panic(err)
	}
	lfd, err := lconn.File()
	if err != nil {
		panic(err)
	}

	qaddr := net.UnixAddr{socketQueryPath, "unix"}
	os.Remove(qaddr.Name) // remove existing socket
	qconn, err := net.ListenUnix("unix", &qaddr)
	if err != nil {
		panic(err)
	}
	qfd, err := qconn.File()
	if err != nil {
		panic(err)
	}

	cmd := exec.Command("/sbin/start-stop-daemon", "--start", "--pidfile", pidFile,
		"--exec", "/usr/bin/runc", "--", "run", "--preserve-fds=2",
		"--bundle", memlogdBundle,
		"--pid-file", pidFile, "memlogd")
	log.Println(cmd.Args)
	cmd.ExtraFiles = append(cmd.ExtraFiles, lfd, qfd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	if detach {
		if err := cmd.Process.Release(); err != nil {
			panic(err)
		}
	} else {
		if err := cmd.Wait(); err != nil {
			if exitError, ok := err.(*exec.ExitError); ok {
				// exit with exit code from process
				status := exitError.Sys().(syscall.WaitStatus)
				os.Exit(status.ExitStatus())
			} else {
				// no exit code, report error and exit 1
				fmt.Println(err)
				os.Exit(1)
			}
		}
	}
}
