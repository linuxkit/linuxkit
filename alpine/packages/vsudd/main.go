package main

import (
	"flag"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rneugeba/virtsock/go/hvsock"
	"github.com/rneugeba/virtsock/go/vsock"
)

var (
	portstr   string
	sock      string
	detach    bool
	useHVsock bool
)

type vConn interface {
	net.Conn
	CloseRead() error
	CloseWrite() error
}

func init() {
	flag.StringVar(&portstr, "port", "2376", "vsock port to forward")
	flag.StringVar(&sock, "sock", "/var/run/docker.sock", "path of the local Unix domain socket to forward to")
	flag.BoolVar(&detach, "detach", false, "detach from terminal")
}

func main() {
	log.SetFlags(log.LstdFlags)
	flag.Parse()

	if detach {
		logFile, err := os.Create("/var/log/vsudd.log")
		if err != nil {
			log.Fatalln("Failed to open log file", err)
		}
		log.SetOutput(logFile)
		null, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
		if err != nil {
			log.Fatalln("Failed to open /dev/null", err)
		}
		fd := null.Fd()
		syscall.Dup2(int(fd), int(os.Stdin.Fd()))
		syscall.Dup2(int(fd), int(os.Stdout.Fd()))
		syscall.Dup2(int(fd), int(os.Stderr.Fd()))
	}

	var l net.Listener
	if strings.Contains(portstr, "-") {
		svcid, err := hvsock.GuidFromString(portstr)
		if err != nil {
			log.Fatalln("Failed to parse GUID", portstr, err)
		}
		l, err = hvsock.Listen(hvsock.HypervAddr{VmId: hvsock.GUID_WILDCARD, ServiceId: svcid})
		if err != nil {
			log.Fatalf("Failed to bind to hvsock port: %s", err)
		}
		log.Printf("Listening on ServiceId %s", svcid)
		useHVsock = true
	} else {
		port, err := strconv.ParseUint(portstr, 10, 32)
		if err != nil {
			log.Fatalln("Can't convert %s to a uint.", portstr, err)
		}
		l, err = vsock.Listen(uint(port))
		if err != nil {
			log.Fatalf("Failed to bind to vsock port %u: %s", port, err)
		}
		log.Printf("Listening on port %u", port)
		useHVsock = false
	}

	connid := 0
	for {
		connid++
		conn, err := l.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %s", err)
			return // no more listening
		}
		log.Printf("Connection %d from: %s\n", connid, conn.RemoteAddr())

		go handleOne(connid, conn.(vConn))
	}
}

func handleOne(connid int, conn vConn) {
	defer func() {
		if err := conn.Close(); err != nil {
			// On windows we get an EINVAL when the other end already closed
			// Don't bother spilling this into the logs
			if !(useHVsock && err == syscall.EINVAL) {
				log.Println(connid, "Error closing", conn, ":", err)
			}
		}
	}()

	var docker *net.UnixConn
	var err error

	// Cope with the server socket appearing up to 10s later
	for i := 0; i < 200; i++ {
		docker, err = net.DialUnix("unix", nil, &net.UnixAddr{sock, "unix"})
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		// If the forwarding program has broken then close and continue
		log.Println(connid, "Failed to connect to Unix domain socket after 10s", sock, err)
		return
	}
	defer func() {
		if err := docker.Close(); err != nil {
			log.Println(connid, "Error closing", docker, ":", err)
		}
	}()

	w := make(chan int64)
	go func() {
		n, err := io.Copy(conn, docker)
		if err != nil {
			log.Println(connid, "error copying from docker to vsock:", err)
		}

		err = docker.CloseRead()
		if err != nil {
			log.Println(connid, "error CloseRead on docker socket:", err)
		}
		err = conn.CloseWrite()
		if err != nil {
			log.Println(connid, "error CloseWrite on vsock:", err)
		}
		w <- n
	}()

	n, err := io.Copy(docker, conn)
	if err != nil {
		log.Println(connid, "error copying from vsock to docker:", err)
	}
	totalRead := n

	err = docker.CloseWrite()
	if err != nil {
		log.Println(connid, "error CloseWrite on docker socket:", err)
	}
	err = conn.CloseRead()
	if err != nil {
		log.Println(connid, "error CloseRead on vsock:", err)
	}

	totalWritten := <-w
	log.Println(connid, "Done. read:", totalRead, "written:", totalWritten)
}
