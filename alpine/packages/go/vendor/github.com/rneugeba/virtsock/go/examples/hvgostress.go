package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"crypto/md5"
	"math/rand"
	"sync/atomic"

	"../hvsock"
)

var (
	clientStr   string
	serverMode  bool
	maxDataLen  int
	connections int
	sleepTime   int
	verbose     int
	exitOnError bool
	parallel    int
	svcid, _    = hvsock.GuidFromString("3049197C-9A4E-4FBF-9367-97F792F16994")

	connCounter int32
)

func init() {
	flag.StringVar(&clientStr, "c", "", "Client")
	flag.BoolVar(&serverMode, "s", false, "Start as a Server")
	flag.IntVar(&maxDataLen, "l", 64*1024, "Maximum Length of data")
	flag.IntVar(&connections, "i", 100, "Total number of connections")
	flag.IntVar(&sleepTime, "w", 0, "Sleep time in seconds between new connections")
	flag.IntVar(&parallel, "p", 1, "Run n connections in parallel")
	flag.BoolVar(&exitOnError, "e", false, "Exit when an error occurs")
	flag.IntVar(&verbose, "v", 0, "Set the verbosity level")

	rand.Seed(time.Now().UnixNano())
}

func main() {
	log.SetFlags(log.LstdFlags)
	flag.Parse()

	if verbose > 2 {
		hvsock.Debug = true
	}

	if serverMode {
		fmt.Printf("Starting server\n")
		server()
		return
	}

	// Client mode
	vmid := hvsock.GUID_ZERO
	var err error
	if strings.Contains(clientStr, "-") {
		vmid, err = hvsock.GuidFromString(clientStr)
		if err != nil {
			log.Fatalln("Can't parse GUID: ", clientStr)
		}
	} else if clientStr == "parent" {
		vmid = hvsock.GUID_PARENT
	} else {
		vmid = hvsock.GUID_LOOPBACK
	}

	if parallel <= 1 {
		// No parallelism, run in the main thread.
		fmt.Printf("Client connecting to %s\n", vmid.String())
		for i := 0; i < connections; i++ {
			client(vmid, i)
			time.Sleep(time.Duration(sleepTime) * time.Second)
		}
		return
	}

	// Parallel clients
	var wg sync.WaitGroup
	for i := 0; i < parallel; i++ {
		wg.Add(1)
		go parClient(&wg, vmid)
	}
	wg.Wait()
}

func server() {
	l, err := hvsock.Listen(hvsock.HypervAddr{VmId: hvsock.GUID_WILDCARD, ServiceId: svcid})
	if err != nil {
		log.Fatalln("Listen():", err)
	}
	defer func() {
		l.Close()
	}()

	connid := 0

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Fatalf("Accept(): %s\n", err)
		}

		prDebug("[%05d] accept(): %s -> %s \n", connid, conn.RemoteAddr(), conn.LocalAddr())
		go handleRequest(conn, connid)
		connid++
	}
}

func handleRequest(c net.Conn, connid int) {
	defer func() {
		prDebug("[%05d] Closing\n", connid)
		err := c.Close()
		if err != nil {
			prError("[%05d] Close(): %s\n", connid, err)
		}
	}()

	n, err := io.Copy(c, c)
	if err != nil {
		prError("[%05d] Copy(): %s", connid, err)
		return
	}
	prInfo("[%05d] Copied Bytes: %d\n", connid, n)

	if n == 0 {
		return
	}

	prDebug("[%05d] Sending BYE message\n", connid)

	// The '\n' is important as the client use ReadString()
	_, err = fmt.Fprintf(c, "Got %d bytes. Bye\n", n)
	if err != nil {
		prError("[%05d] Failed to send: %s", connid, err)
		return
	}
	prDebug("[%05d] Sent bye\n", connid)
}

func parClient(wg *sync.WaitGroup, vmid hvsock.GUID) {
	connid := int(atomic.AddInt32(&connCounter, 1))
	for connid < connections {
		client(vmid, connid)
		connid = int(atomic.AddInt32(&connCounter, 1))
		time.Sleep(time.Duration(sleepTime) * time.Second)
	}

	wg.Done()
}

func client(vmid hvsock.GUID, conid int) {
	sa := hvsock.HypervAddr{VmId: vmid, ServiceId: svcid}
	c, err := hvsock.Dial(sa)
	if err != nil {
		prError("[%05d] Failed to Dial: %s:%s %s\n", conid, sa.VmId.String(), sa.ServiceId.String(), err)
	}

	defer c.Close()

	// Create buffer with random data and random length.
	// Make sure the buffer is not zero-length
	buflen := rand.Intn(maxDataLen-1) + 1
	txbuf := randBuf(buflen)
	csum0 := md5.Sum(txbuf)

	prDebug("[%05d] TX: %d bytes, md5=%02x\n", conid, buflen, csum0)

	w := make(chan int)
	go func() {
		l, err := c.Write(txbuf)
		if err != nil {
			prError("[%05d] Failed to send: %s\n", conid, err)
		}
		if l != buflen {
			prError("[%05d] Failed to send enough data: %d\n", conid, l)
		}

		// Tell the other end that we are done
		c.CloseWrite()

		w <- l
	}()

	rxbuf := make([]byte, buflen)

	n, err := io.ReadFull(bufio.NewReader(c), rxbuf)
	if err != nil {
		prError("[%05d] Failed to receive: %s\n", conid, err)
		return
	}
	csum1 := md5.Sum(rxbuf)

	totalSent := <-w

	prInfo("[%05d] RX: %d bytes, md5=%02x (sent=%d)\n", conid, n, csum1, totalSent)
	if csum0 != csum1 {
		prError("[%05d] Checksums don't match", conid)
	}

	// Wait for Bye message
	message, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		prError("[%05d] Failed to receive bye: %s\n", conid, err)
	}
	prDebug("[%05d] From SVR: %s", conid, message)
}

func randBuf(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rand.Intn(255))
	}
	return b
}

func prError(format string, args ...interface{}) {
	if exitOnError {
		log.Fatalf(format, args...)
	} else {
		log.Printf(format, args...)
	}
}

func prInfo(format string, args ...interface{}) {
	if verbose > 0 {
		log.Printf(format, args...)
	}
}

func prDebug(format string, args ...interface{}) {
	if verbose > 1 {
		log.Printf(format, args...)
	}
}
