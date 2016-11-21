package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"log/syslog"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/rneugeba/virtsock/go/hvsock"
	"github.com/rneugeba/virtsock/go/vsock"
)

type forward struct {
	vsock string
	net   string // "unix" or "unixgram"
	usock string
}

type forwards []forward

var (
	inForwards forwards
	detach     bool
	useHVsock  bool
	syslogFwd  string
	pidfile    string
)

type vConn interface {
	net.Conn
	CloseRead() error
	CloseWrite() error
}

func (f *forwards) String() string {
	return "Forwards"
}

func (f *forwards) Set(value string) error {
	s := strings.SplitN(value, ":", 3)
	if len(s) != 3 {
		return fmt.Errorf("Failed to parse: %s", value)
	}
	var newF forward
	newF.vsock = s[0]
	newF.net = s[1]
	newF.usock = s[2]
	*f = append(*f, newF)
	return nil
}

func init() {
	flag.Var(&inForwards, "inport", "incoming port to forward")
	flag.StringVar(&syslogFwd, "syslog", "", "enable syslog forwarding")
	flag.BoolVar(&detach, "detach", false, "detach from terminal")
	flag.StringVar(&pidfile, "pidfile", "", "pid file")
}

func main() {
	log.SetFlags(log.LstdFlags)
	flag.Parse()

	if pidfile != "" {
		file, err := os.OpenFile(pidfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			log.Fatalln("Failed to open pidfile", err)
		}
		_, err = fmt.Fprintf(file, "%d", os.Getpid())
		file.Close()
		if err != nil {
			log.Fatalln("Failed to write pid", err)
		}
	}

	if detach {
		syslog, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "vsudd")
		if err != nil {
			log.Fatalln("Failed to open syslog", err)
		}

		null, err := os.OpenFile("/dev/null", os.O_RDWR, 0)
		if err != nil {
			log.Fatalln("Failed to open /dev/null", err)
		}

		/* Don't do this above since we aren't yet forwarding
		/* syslog (if we've been asked to) so the above error
		/* reporting wants to go via the default path
		/* (stdio). */

		log.SetOutput(syslog)
		log.SetFlags(0)

		fd := null.Fd()
		syscall.Dup2(int(fd), int(os.Stdin.Fd()))
		syscall.Dup2(int(fd), int(os.Stdout.Fd()))
		syscall.Dup2(int(fd), int(os.Stderr.Fd()))
	}

	var wg sync.WaitGroup

	if syslogFwd != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			handleSyslogForward(syslogFwd)
		}()
	}

	connid := 0

	for _, inF := range inForwards {
		var portstr = inF.vsock
		var network = inF.net
		var usock = inF.usock

		var l net.Listener

		if network != "unix" {
			log.Fatalf("cannot forward incoming port to %s:%s", network, usock)
		}

		log.Printf("incoming port forward from %s to %s", portstr, usock)

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
				log.Fatalf("Failed to bind to vsock port %d: %s", port, err)
			}
			log.Printf("Listening on port %s", portstr)
			useHVsock = false
		}

		wg.Add(1)

		go func() {
			defer wg.Done()
			for {
				connid++
				conn, err := l.Accept()
				if err != nil {
					log.Printf("Error accepting connection: %s", err)
					return // no more listening
				}
				log.Printf("Connection %d to: %s from: %s\n", connid, portstr, conn.RemoteAddr())

				go handleOneIn(connid, conn.(vConn), usock)
			}
		}()
	}

	wg.Wait()
}

func handleOneIn(connid int, conn vConn, sock string) {
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

	docker, err = net.DialUnix("unix", nil, &net.UnixAddr{sock, "unix"})

	if err != nil {
		// If the forwarding program has broken then close and continue
		log.Println(connid, "Failed to connect to Unix domain socket", sock, err)
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
