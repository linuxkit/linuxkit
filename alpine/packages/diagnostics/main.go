package main

import (
	"archive/tar"
	"bytes"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/rneugeba/virtsock/go/vsock"
	"github.com/rneugeba/virtsock/go/hvsock"
)

func run(timeout time.Duration, w *tar.Writer, command string, args ...string) {
	log.Printf("Running %s", command)
	c := exec.Command(command, args...)
	stdoutPipe, err := c.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to create stdout pipe: %#v", err)
	}
	stderrPipe, err := c.StderrPipe()
	if err != nil {
		log.Fatalf("Failed to create stderr pipe: %#v", err)
	}
	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	done := make(chan int)
	go func() {
		io.Copy(&stdoutBuffer, stdoutPipe)
		done <- 0
	}()
	go func() {
		io.Copy(&stderrBuffer, stderrPipe)
		done <- 0
	}()
	var timer *time.Timer
	timer = time.AfterFunc(timeout, func() {
		timer.Stop()
		if c.Process != nil {
			c.Process.Kill()
		}
	})
	_ = c.Run()
	<-done
	<-done
	timer.Stop()

	name := strings.Join(append([]string{path.Base(command)}, args...), " ")

	hdr := &tar.Header{
		Name: name + ".stdout",
		Mode: 0644,
		Size: int64(stdoutBuffer.Len()),
	}
	if err = w.WriteHeader(hdr); err != nil {
		log.Fatalln(err)
	}
	if _, err = w.Write(stdoutBuffer.Bytes()); err != nil {
		log.Fatalln(err)
	}
	hdr = &tar.Header{
		Name: name + ".stderr",
		Mode: 0644,
		Size: int64(stderrBuffer.Len()),
	}
	if err = w.WriteHeader(hdr); err != nil {
		log.Fatalln(err)
	}
	if _, err = w.Write(stderrBuffer.Bytes()); err != nil {
		log.Fatalln(err)
	}

}

func capture(w *tar.Writer) {
	t := 2 * time.Second

	run(t, w, "/bin/date")
	run(t, w, "/bin/uname", "-a")
	run(t, w, "/bin/ps", "uax")
	run(t, w, "/bin/netstat", "-tulpn")
	run(t, w, "/sbin/iptables-save")
	run(t, w, "/sbin/ifconfig", "-a")
	run(t, w, "/sbin/route", "-n")
	run(t, w, "/usr/sbin/brctl", "show")
	run(t, w, "/bin/dmesg")
	run(t, w, "/usr/bin/docker", "ps")
	run(t, w, "/usr/bin/tail", "-100", "/var/log/docker.log")
	run(t, w, "/usr/bin/tail", "-100", "/var/log/messages")
	run(t, w, "/usr/bin/tail", "-100", "/var/log/proxy-vsockd.log")
	run(t, w, "/usr/bin/tail", "-100", "/var/log/vsudd.log")
	run(t, w, "/bin/mount")
	run(t, w, "/bin/df")
	run(t, w, "/bin/ls", "-l", "/var")
	run(t, w, "/bin/ls", "-l", "/var/lib")
	run(t, w, "/bin/ls", "-l", "/var/lib/docker")
	run(t, w, "/usr/bin/diagnostics")
	run(t, w, "/bin/ping", "-w", "5", "8.8.8.8")
	run(t, w, "/bin/cat", "/etc/network/interfaces")
	run(t, w, "/bin/cat", "/etc/resolv.conf")
	run(t, w, "/bin/cat", "/etc/sysctl.conf")
	run(t, w, "/usr/bin/dig", "docker.com")
	run(t, w, "/usr/bin/wget", "-O", "-", "http://www.docker.com/")

	// Dump the database
	dbBase := "/Database/branch/master/ro"
	filepath.Walk(dbBase, func(path string, f os.FileInfo, err error) error {
		if f.Mode().IsRegular() {
			run(t, w, "/bin/cat", path)
		}
		return nil
	})
}

func main() {
	listeners := make([]net.Listener, 0)

	ip, err := net.Listen("tcp", ":62374")
	if err != nil {
		log.Printf("Failed to bind to TCP port 62374: %#v", err)
	} else {
		listeners = append(listeners, ip)
	}
	vsock, err := vsock.Listen(uint(62374))
	if err != nil {
		log.Printf("Failed to bind to vsock port 62374: %#v", err)
	} else {
		listeners = append(listeners, vsock)
	}
	svcid, _ := hvsock.GuidFromString("445BA2CB-E69B-4912-8B42-D7F494D007EA")
	hvsock, err := hvsock.Listen(hvsock.HypervAddr{VmId: hvsock.GUID_WILDCARD, ServiceId: svcid})
	if err != nil {
		log.Printf("Failed to bind to hvsock port: %#v", err)
	} else {
		listeners = append(listeners, hvsock)
	}

	for _, l := range listeners {
		go func(l net.Listener) {
			for {
				conn, err := l.Accept()
				if err != nil {
					log.Printf("Error accepting connection: %#v", err)
					return // no more listening
				}
				go func(conn net.Conn) {
					w := tar.NewWriter(conn)
					capture(w)
					if err := w.Close(); err != nil {
						log.Println(err)
					}
					conn.Close()
				}(conn)
			}
		}(l)
	}
	forever := make(chan int)
	<-forever
}
