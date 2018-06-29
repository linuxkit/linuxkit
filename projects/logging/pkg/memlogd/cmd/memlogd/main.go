package main

import (
	"bufio"
	"bytes"
	"container/list"
	"container/ring"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"syscall"
	"time"
)

type logEntry struct {
	time   time.Time
	source string
	msg    string
}

type fdMessage struct {
	name string
	fd   int
}

type logMode byte

const (
	logDump logMode = iota
	logFollow
	logDumpFollow
)

type queryMessage struct {
	conn *net.UnixConn
	mode logMode
}

type connListener struct {
	conn      *net.UnixConn
	cond      *sync.Cond // condition and mutex used to notify listeners of more data
	buffer    bytes.Buffer
	err       error
	exitOnEOF bool // exit instead of blocking if no more data in read buffer
}

func doLog(logCh chan logEntry, msg string) {
	logCh <- logEntry{time: time.Now(), source: "memlogd", msg: msg}
	return
}

func logQueryHandler(l *connListener) {
	defer l.conn.Close()

	data := make([]byte, 0xffff)

	l.cond.L.Lock()
	for {
		var n, remaining int
		var rerr, werr error

		for rerr == nil && werr == nil {
			if n, rerr = l.buffer.Read(data); n == 0 { // process data before checking error
				break // exit read loop to wait for more data
			}
			l.cond.L.Unlock()

			remaining = n
			w := data
			for remaining > 0 && werr == nil {
				w = data[:remaining]
				n, werr = l.conn.Write(w)
				w = w[n:]
				remaining = remaining - n
			}

			l.cond.L.Lock()
		}

		// check errors
		if werr != nil {
			l.err = werr
			l.cond.L.Unlock()
			break
		}

		if rerr != nil && rerr != io.EOF { // EOF is ok, just wait for more data
			l.err = rerr
			l.cond.L.Unlock()
			break
		}
		if l.exitOnEOF && rerr == io.EOF { // ... unless we should exit on EOF
			l.err = nil
			l.cond.L.Unlock()
			break
		}
		l.cond.Wait() // unlock and wait for more data
	}
}

func (msg *logEntry) String() string {
	return fmt.Sprintf("%s %s %s", msg.time.Format(time.RFC3339), msg.source, msg.msg)
}

func ringBufferHandler(ringSize int, logCh chan logEntry, queryMsgChan chan queryMessage) {
	// Anything that interacts with the ring buffer goes through this handler
	ring := ring.New(ringSize)
	listeners := list.New()

	for {
		select {
		case msg := <-logCh:
			fmt.Printf("%s\n", msg.String())
			// add log entry
			ring.Value = msg
			ring = ring.Next()

			// send to listeners
			var l *connListener
			var remove []*list.Element
			for e := listeners.Front(); e != nil; e = e.Next() {
				l = e.Value.(*connListener)
				if l.err != nil {
					remove = append(remove, e)
					continue
				}
				l.cond.L.Lock()
				l.buffer.WriteString(fmt.Sprintf("%s\n", msg.String()))
				l.cond.L.Unlock()
				l.cond.Signal()
			}
			if len(remove) > 0 { // remove listeners that returned errors
				for _, e := range remove {
					l = e.Value.(*connListener)
					fmt.Println("Removing connection, error: ", l.err)
					listeners.Remove(e)
				}
			}

		case msg := <-queryMsgChan:
			l := connListener{conn: msg.conn, cond: sync.NewCond(&sync.Mutex{}), err: nil, exitOnEOF: (msg.mode == logDump)}
			listeners.PushBack(&l)
			go logQueryHandler(&l)
			if msg.mode == logDumpFollow || msg.mode == logDump {
				l.cond.L.Lock()
				// fill with current data in buffer
				ring.Do(func(f interface{}) {
					if msg, ok := f.(logEntry); ok {
						s := fmt.Sprintf("%s\n", msg.String())
						l.buffer.WriteString(s)
					}
				})
				l.cond.L.Unlock()
				l.cond.Signal() // signal handler that more data is available
			}
		}
	}
}

func receiveQueryHandler(l *net.UnixListener, logCh chan logEntry, queryMsgChan chan queryMessage) {
	for {
		var conn *net.UnixConn
		var err error
		if conn, err = l.AcceptUnix(); err != nil {
			doLog(logCh, fmt.Sprintf("Connection error %s", err))
			continue
		}
		mode := make([]byte, 1)
		n, err := conn.Read(mode)
		if err != nil || n != 1 {
			doLog(logCh, fmt.Sprintf("No mode received: %s", err))
		}
		queryMsgChan <- queryMessage{conn, logMode(mode[0])}
	}
}

func receiveFdHandler(conn *net.UnixConn, logCh chan logEntry, fdMsgChan chan fdMessage) {
	oob := make([]byte, 512)
	b := make([]byte, 512)

	for {
		n, oobn, _, _, err := conn.ReadMsgUnix(b, oob)
		if err != nil {
			doLog(logCh, fmt.Sprintf("ERROR: Unable to read oob data: %s", err.Error()))
			continue
		}

		if oobn == 0 {
			continue
		}

		oobmsgs, err := syscall.ParseSocketControlMessage(oob[:oobn])
		if err != nil {
			doLog(logCh, fmt.Sprintf("ERROR: Failed to parse socket control message: %s", err.Error()))
			continue
		}

		for _, oobmsg := range oobmsgs {
			r, err := syscall.ParseUnixRights(&oobmsg)
			if err != nil {
				doLog(logCh, fmt.Sprintf("ERROR: Failed to parse UNIX rights in oob data: %s", err.Error()))
				continue
			}
			for _, fd := range r {
				name := ""
				if n > 0 {
					name = string(b[:n])
				}
				fdMsgChan <- fdMessage{name: name, fd: fd}
			}
		}
	}
}

func readLogFromFd(maxLineLen int, fd int, source string, logCh chan logEntry) {
	f := os.NewFile(uintptr(fd), "")
	defer f.Close()

	r := bufio.NewReader(f)
	l, isPrefix, err := r.ReadLine()
	var buffer bytes.Buffer

	for err == nil {
		buffer.Write(l)
		for isPrefix {
			l, isPrefix, err = r.ReadLine()
			if err != nil {
				break
			}
			if buffer.Len() < maxLineLen {
				buffer.Write(l)
			}
		}
		if buffer.Len() > maxLineLen {
			buffer.Truncate(maxLineLen)
		}
		logCh <- logEntry{time: time.Now(), source: source, msg: buffer.String()}
		buffer.Reset()

		l, isPrefix, err = r.ReadLine()
	}
}

func main() {
	var err error

	var socketQueryPath string
	var passedQueryFD int
	var socketLogPath string
	var passedLogFD int
	var linesInBuffer int
	var lineMaxLength int

	flag.StringVar(&socketQueryPath, "socket-query", "/tmp/memlogdq.sock", "unix domain socket for responding to log queries. Overridden by -fd-query")
	flag.StringVar(&socketLogPath, "socket-log", "/tmp/memlogd.sock", "unix domain socket to listen for new fds to add to log. Overridden by -fd-log")
	flag.IntVar(&passedLogFD, "fd-log", -1, "an existing SOCK_DGRAM socket for receiving fd's. Overrides -socket-log.")
	flag.IntVar(&passedQueryFD, "fd-query", -1, "an existing SOCK_STREAM for receiving log read requets. Overrides -socket-query.")
	flag.IntVar(&linesInBuffer, "max-lines", 5000, "Number of log lines to keep in memory")
	flag.IntVar(&lineMaxLength, "max-line-len", 1024, "Maximum line length recorded. Additional bytes are dropped.")

	flag.Parse()

	var connLogFd *net.UnixConn
	if passedLogFD == -1 { // no fd on command line, use socket path
		addr := net.UnixAddr{
			Name: socketLogPath,
			Net:  "unixgram",
		}
		if connLogFd, err = net.ListenUnixgram("unixgram", &addr); err != nil {
			log.Fatal("Unable to open socket: ", err)
		}
		defer os.Remove(addr.Name)
	} else { // use given fd
		var f net.Conn
		if f, err = net.FileConn(os.NewFile(uintptr(passedLogFD), "")); err != nil {
			log.Fatal("Unable to open fd: ", err)
		}
		connLogFd = f.(*net.UnixConn)
	}
	defer connLogFd.Close()

	var connQuery *net.UnixListener
	if passedQueryFD == -1 { // no fd on command line, use socket path
		addr := net.UnixAddr{
			Name: socketQueryPath,
			Net:  "unix",
		}
		if connQuery, err = net.ListenUnix("unix", &addr); err != nil {
			log.Fatal("Unable to open socket: ", err)
		}
		defer os.Remove(addr.Name)
	} else { // use given fd
		var f net.Listener
		if f, err = net.FileListener(os.NewFile(uintptr(passedQueryFD), "")); err != nil {
			log.Fatal("Unable to open fd: ", err)
		}
		connQuery = f.(*net.UnixListener)
	}
	defer connQuery.Close()

	logCh := make(chan logEntry)
	fdMsgChan := make(chan fdMessage)
	queryMsgChan := make(chan queryMessage)

	go receiveFdHandler(connLogFd, logCh, fdMsgChan)
	go receiveQueryHandler(connQuery, logCh, queryMsgChan)
	go ringBufferHandler(linesInBuffer, logCh, queryMsgChan)

	doLog(logCh, "memlogd started")

	for true {
		select {
		case msg := <-fdMsgChan: // incoming fd
			go readLogFromFd(lineMaxLength, msg.fd, msg.name, logCh)
		}
	}
}
