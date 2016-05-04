package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

/* No way to teach net or syscall about vsock sockaddr, so go right to C */

/*
#include <stdio.h>
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

struct sockaddr_hv {
	unsigned short shv_family;
	unsigned short reserved;
	unsigned char  shv_vm_id[16];
	unsigned char  shv_service_id[16];
};
int bind_sockaddr_hv(int fd, const struct sockaddr_hv *sa_hv) {
    return bind(fd, (const struct sockaddr*)sa_hv, sizeof(*sa_hv));
}
int connect_sockaddr_hv(int fd, const struct sockaddr_hv *sa_hv) {
    return connect(fd, (const struct sockaddr*)sa_hv, sizeof(*sa_hv));
}
int accept_hv(int fd, struct sockaddr_hv *sa_hv, socklen_t *sa_hv_len) {
    return accept4(fd, (struct sockaddr *)sa_hv, sa_hv_len, 0);
}
*/
import "C"

type GUID [16]byte

const (
	AF_VSOCK      = 40
	VSOCK_CID_ANY = 4294967295 /* 2^32-1 */

	AF_HYPERV     = 42
	SHV_PROTO_RAW = 1
)

var (
	portstr string
	sock    string
	detach  bool
	SHV_VMID_GUEST = GUID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
)

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

	if strings.Contains(portstr, "-") {
		guid, err := guidFromString(portstr)
		if err != nil {
			log.Fatalln("Failed to parse GUID", portstr, err)
		}
		hvsockListen(guid)
	} else {
		port, err := strconv.ParseUint(portstr, 10, 32)
		if err != nil {
			log.Fatalln("Can't convert %s to a uint.", portstr, err)
		}
		vsockListen(uint(port))
	}
}

func hvsockListen(port GUID) {
	accept_fd, err := syscall.Socket(AF_HYPERV, syscall.SOCK_STREAM, SHV_PROTO_RAW)
	if err != nil {
		log.Fatal(err)
	}

	sa := C.struct_sockaddr_hv{}
	sa.shv_family = AF_HYPERV
	sa.reserved = 0
	/* TODO: Turn this into a function */
	for i := 0; i < 16; i++ {
		sa.shv_vm_id[i] = C.uchar(SHV_VMID_GUEST[i])
	}
	for i := 0; i < 16; i++ {
		sa.shv_service_id[i] = C.uchar(port[i])
	}

	if ret := C.bind_sockaddr_hv(C.int(accept_fd), &sa); ret != 0 {
		log.Fatal(fmt.Sprintf("failed bind hvsock connection to %s.%s, returned %d",
			SHV_VMID_GUEST.toString(), port.toString(), ret))
	}

	err = syscall.Listen(accept_fd, syscall.SOMAXCONN)
	if err != nil {
		log.Fatalln("Failed to listen to VSOCK", err)
	}

	log.Printf("Listening on fd %d", accept_fd)

	connid := 0

	for {
		var accept_sa C.struct_sockaddr_hv
		var accept_sa_len C.socklen_t

		connid++
		accept_sa_len = C.sizeof_struct_sockaddr_hv
		fd, err := C.accept_hv(C.int(accept_fd), &accept_sa, &accept_sa_len)
		if err != nil {
			log.Fatalln("Error accepting connection", err)
		}

		accept_vm_id := guidFromC(accept_sa.shv_vm_id)
		accept_svc_id := guidFromC(accept_sa.shv_service_id)
		log.Printf("%d Accepted connection on fd %d from %s.%s",
			connid, fd, accept_vm_id.toString(), accept_svc_id.toString())
		go handleOne(connid, int(fd))
	}
}

func vsockListen(port uint) {
	accept_fd, err := syscall.Socket(AF_VSOCK, syscall.SOCK_STREAM, 0)
	if err != nil {
		log.Fatal(err)
	}

	sa := C.struct_sockaddr_vm{}
	sa.svm_family = AF_VSOCK
	sa.svm_port = C.uint(port)
	sa.svm_cid = VSOCK_CID_ANY

	if ret := C.bind_sockaddr_vm(C.int(accept_fd), &sa); ret != 0 {
		log.Fatal(fmt.Sprintf("failed bind vsock connection to %08x.%08x, returned %d",
			sa.svm_cid, sa.svm_port, ret))
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
		log.Printf("%d Accepted connection on fd %d from %08x.%08x",
			connid, fd, uint(accept_sa.svm_cid), uint(accept_sa.svm_port))
		go handleOne(connid, int(fd))
	}
}

func handleOne(connid int, fd int) {
	vsock := os.NewFile(uintptr(fd), fmt.Sprintf("vsock:%d", fd))

	defer func() {
		if err := vsock.Close(); err != nil {
			log.Println(connid, "Error closing", vsock, ":", err)
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
		n, err := io.Copy(vsock, docker)
		if err != nil {
			log.Println(connid, "error copying from docker to vsock:", err)
		}

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

func (g *GUID) toString() string {
	/* XXX This assume little endian */
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		g[3], g[2], g[1], g[0],
		g[5], g[4],
		g[7], g[6],
		g[8], g[9],
		g[10], g[11], g[12], g[13], g[14], g[15])
}

func guidFromString(s string) (GUID, error) {
	var g GUID
	var err error
	_, err = fmt.Sscanf(s, "%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		&g[3], &g[2], &g[1], &g[0],
		&g[5], &g[4],
		&g[7], &g[6],
		&g[8], &g[9],
		&g[10], &g[11], &g[12], &g[13], &g[14], &g[15])
	return g, err
}

func guidFromC(cg [16]C.uchar) GUID {
       var g GUID
       for i := 0; i < 16; i++ {
               g[i] = byte(cg[i])
       }
       return g
}
