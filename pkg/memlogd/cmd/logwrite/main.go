package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"syscall"
)

func getLogFileSocketPair() (*os.File, int) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		panic(err)
	}

	localFd := fds[0]
	remoteFd := fds[1]

	localLogFile := os.NewFile(uintptr(localFd), "")
	return localLogFile, remoteFd
}

func sendFD(conn *net.UnixConn, remoteAddr *net.UnixAddr, source string, fd int) error {
	oobs := syscall.UnixRights(fd)
	_, _, err := conn.WriteMsgUnix([]byte(source), oobs, remoteAddr)
	return err
}

func main() {
	var err error
	var ok bool

	var serverSocket string
	var name string

	flag.StringVar(&serverSocket, "socket", "/var/run/linuxkit-external-logging.sock", "socket to pass fd's to memlogd")
	flag.StringVar(&name, "n", "", "name of sender, defaults to first argument if left blank")
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		log.Fatal("no command specified")
	}

	if name == "" {
		name = args[0]
	}

	localStdoutLog, remoteStdoutFd := getLogFileSocketPair()
	localStderrLog, remoteStderrFd := getLogFileSocketPair()

	var outSocket int
	if outSocket, err = syscall.Socket(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0); err != nil {
		log.Fatal("Unable to create socket: ", err)
	}

	var outFile net.Conn
	if outFile, err = net.FileConn(os.NewFile(uintptr(outSocket), "")); err != nil {
		log.Fatal(err)
	}

	var conn *net.UnixConn
	if conn, ok = outFile.(*net.UnixConn); !ok {
		log.Fatal("Internal error, invalid cast.")
	}

	raddr := net.UnixAddr{Name: serverSocket, Net: "unixgram"}

	if err = sendFD(conn, &raddr, name+".stdout", remoteStdoutFd); err != nil {
		log.Fatal("fd stdout send failed: ", err)
	}

	if err = sendFD(conn, &raddr, name+".stderr", remoteStderrFd); err != nil {
		log.Fatal("fd stderr send failed: ", err)
	}

	cmd := exec.Command(args[0], args[1:]...)
	outStderr := io.MultiWriter(localStderrLog, os.Stderr)
	outStdout := io.MultiWriter(localStdoutLog, os.Stdout)
	cmd.Stderr = outStderr
	cmd.Stdout = outStdout
	if err = cmd.Run(); err != nil {
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
