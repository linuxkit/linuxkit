package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"syscall"
	"time"
)

/* No way to teach net or syscall about vsock sockaddr, so go right to C */

/*
#include <sys/socket.h>
#include "include/uapi/linux/vm_sockets.h"
int bind_sockaddr_vm(int fd, const struct sockaddr_vm *sa_vm) {
    return bind(fd, (const struct sockaddr*)sa_vm, sizeof(*sa_vm));
}
int connect_sockaddr_vm(int fd, const struct sockaddr_vm *sa_vm) {
    return connect(fd, (const struct sockaddr*)sa_vm, sizeof(*sa_vm));
}
int accept_vm(int fd, struct sockaddr_vm *sa_vm, socklen_t *sa_vm_len) {
    return accept4(fd, (struct sockaddr *)sa_vm, sa_vm_len, 0);
}
*/
import "C"

const (
	AF_VSOCK            = 40
	VSOCK_CID_ANY       = 4294967295 /* 2^32-1 */
)

var (
	port   uint
	sock   string
	detach bool
)

func init() {
	flag.UintVar(&port, "port", 2376, "vsock port to forward")
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

	accept_fd, err := syscall.Socket(AF_VSOCK, syscall.SOCK_STREAM, 0)
	if err != nil {
		log.Fatal(err)
	}

	sa := C.struct_sockaddr_vm{}
	sa.svm_family = AF_VSOCK
	sa.svm_port = C.uint(port)
	sa.svm_cid = 3

	if ret := C.bind_sockaddr_vm(C.int(accept_fd), &sa); ret != 0 {
		log.Fatal(fmt.Sprintf("failed bind vsock connection to %08x.%08x, returned %d", sa.svm_cid, sa.svm_port, ret))
	}

	err = syscall.Listen(accept_fd, syscall.SOMAXCONN)
	if err != nil {
		log.Fatalln("Failed to listen to VSOCK", err)
	}

	log.Printf("Listening on fd %d", accept_fd)

	connid := 0

	for {
		var accept_sa C.struct_sockaddr_vm
		var accept_sa_len C.socklen_t

		connid++
		accept_sa_len = C.sizeof_struct_sockaddr_vm
		fd, err := C.accept_vm(C.int(accept_fd), &accept_sa, &accept_sa_len)
		if err != nil {
			log.Fatalln("Error accepting connection", err)
		}
		go handleOne(connid, int(fd), uint(accept_sa.svm_cid), uint(accept_sa.svm_port))
	}
}

func handleOne(connid int, fd int, cid, port uint) {
	vsock := os.NewFile(uintptr(fd), fmt.Sprintf("vsock:%d", fd))
	log.Printf("%d Accepted connection on fd %d from %08x.%08x", connid, fd, cid, port)

	defer syscall.Close(fd)

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
	defer docker.Close()

	if err != nil {
		// If the forwarding program has broken then close and continue
		log.Println(connid, "Failed to connect to Unix domain socket after 10s", sock, err)
		return
	}

	w := make(chan int64)
	go func() {
		n, err := io.Copy(vsock, docker)
		if err != nil {
			log.Println(connid, "error copying from docker to vsock:", err)
		}
		log.Println(connid, "copying from docker to vsock: ", n, "bytes done")

		err = docker.CloseRead()
		if err != nil {
			log.Println(connid, "error CloseRead on docker socket:", err)
		}
		err = syscall.Shutdown(fd, syscall.SHUT_WR)
		if err != nil {
			log.Println(connid, "error SHUT_WR on vsock:", err)
		}
		w <- n
	}()

	n, err := io.Copy(docker, vsock)
	if err != nil {
		log.Println(connid, "error copying from vsock to docker:", err)
	}
	log.Println(connid, "copying from vsock to docker: ", n, "bytes done")
	totalRead := n

	err = docker.CloseWrite()
	if err != nil {
		log.Println(connid, "error CloseWrite on docker socket:", err)
	}
	err = syscall.Shutdown(fd, syscall.SHUT_RD)
	if err != nil {
		log.Println(connid, "error SHUT_RD on vsock:", err)
	}

	totalWritten := <-w
	log.Println(connid, "Done. read:", totalRead, "written:", totalWritten)
}
