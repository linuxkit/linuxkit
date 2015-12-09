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

var (
	path   string
	sock   string
	detach bool
)

func init() {
	flag.StringVar(&path, "path", "/9puds", "path of the 9P-mounted Unix domain socket tree")
	flag.StringVar(&sock, "sock", "/tmp/forwarded.sock", "path of the local Unix domain socket to forward to")
	flag.BoolVar(&detach, "detach", false, "detach from terminal")
}

func main() {
	log.SetFlags(0)
	flag.Parse()

	if detach {
		logFile, err := os.Create("/var/log/9pudc.log")
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

	eventsPath := path + "/events"
	events, err := os.Open(eventsPath)
	if err != nil {
		log.Fatalln("Failed to open file", eventsPath, err)
	}
	// 512 bytes is easily big enough to read a whole connection id
	buf := make([]byte, 512)
	for {
		n, err := events.Read(buf)
		if err != nil {
			log.Fatalln("Error reading events file", err)
		}
		id, err := strconv.Atoi(strings.TrimSpace(string(buf[0:n])))
		if err != nil {
			log.Fatalln("Failed to parse integer connection id", err)
		}
		go handleOne(id)
	}
}

func handleOne(id int) {
	log.Println(id, "handleOne")
	readPath := fmt.Sprintf("%s/connections/%d/read", path, id)
	// Remove will cause the server end to close
	defer func(){
		log.Println(id, "handleOne closing, removing", readPath)
		os.Remove(readPath)
	}()

	read, err := os.Open(readPath)
	if err != nil {
		// Fatal because this is a bug in the server implementation
		log.Fatalln("Failed to open read file", readPath, err)
	}
	defer read.Close()

	var conn *net.UnixConn
	// Cope with the server socket appearing up to 10s later
	for i := 0; i < 200; i++ {
		conn, err = net.DialUnix("unix", nil, &net.UnixAddr{sock, "unix"})
		if err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if err != nil {
		// If the forwarding program has broken then close and continue
		log.Println("Failed to connect to Unix domain socket after 10s", sock, err)
		return
	}

	w := make(chan int64)
	go func() {
		writePath := fmt.Sprintf("%s/connections/%d/write", path, id)
		write, err := os.OpenFile(writePath, os.O_WRONLY, 0666)
		if err != nil {
			// Fatal because this is a bug in the server implementation
			log.Fatalln("Failed to open write file", writePath, err)
		}
		log.Println(id, "copying from", sock, "to", writePath)
		n, err := io.Copy(write, conn)
		if err != nil {
			log.Println("error copying from Unix domain socket to 9P", err)
		}
		log.Println(id, "wrote", n, "bytes to", writePath)
		conn.CloseRead()
		write.Close()
		os.Remove(writePath)
		w <- n
	}()

	totalRead := int64(0)
	log.Println(id, "copying from", readPath, "to", sock)
	for {
		// EOF is used to signal a chunk/packet of data
		n, err := io.Copy(conn, read)
		totalRead = totalRead + n
		log.Println(id, "copied a packet of size", n, "bytes from stream")
		if err != nil {
			log.Println(id, "error copying from stream file to Unix domain socket:", err)
			break
		}
		if n == 0 {
			log.Println(id, "read zero-length chunk from stream file: treating as EOF")
			break
		}
	}
	conn.CloseWrite();

	log.Println(id, "waiting for writer to close")
	totalWritten := <-w
	log.Println(id, "read", totalRead, "written", totalWritten)
}
