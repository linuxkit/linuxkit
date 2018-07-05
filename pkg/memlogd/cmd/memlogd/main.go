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
	"os/exec"
	"strings"
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
	conn net.Conn
	mode logMode
}

type connListener struct {
	conn      net.Conn
	output    chan *logEntry
	err       error
	exitOnEOF bool // exit instead of blocking if no more data in read buffer
}

func doLog(logCh chan logEntry, msg string) {
	logCh <- logEntry{time: time.Now(), source: "memlogd", msg: msg}
	return
}

func logQueryHandler(l *connListener) {
	defer l.conn.Close()

	for msg := range l.output {
		_, err := io.Copy(l.conn, strings.NewReader(msg.String()+"\n"))
		if err != nil {
			l.err = err
			return
		}
	}
}

func (msg *logEntry) String() string {
	return fmt.Sprintf("%s %s %s", msg.time.Format(time.RFC3339), msg.source, msg.msg)
}

func ringBufferHandler(ringSize, chanSize int, logCh chan logEntry, queryMsgChan chan queryMessage) {
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
				select {
				case l.output <- &msg:
				default:
					// channel is full so drop message
				}
			}
			if len(remove) > 0 { // remove listeners that returned errors
				for _, e := range remove {
					l = e.Value.(*connListener)
					fmt.Println("Removing connection, error: ", l.err)
					listeners.Remove(e)
				}
			}

		case msg := <-queryMsgChan:
			l := connListener{
				conn:      msg.conn,
				output:    make(chan *logEntry, chanSize),
				err:       nil,
				exitOnEOF: (msg.mode == logDump),
			}
			go logQueryHandler(&l)
			if msg.mode == logDumpFollow || msg.mode == logFollow {
				// register for future logs
				listeners.PushBack(&l)
			}
			if msg.mode == logDumpFollow || msg.mode == logDump {
				// fill with current data in buffer
				ring.Do(func(f interface{}) {
					if msg, ok := f.(logEntry); ok {
						select {
						case l.output <- &msg:
						default:
							// channel is full so drop message
						}
					}
				})
			}
			if msg.mode == logDump {
				close(l.output)
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
	var daemonize bool

	flag.StringVar(&socketQueryPath, "socket-query", "/var/run/memlogdq.sock", "unix domain socket for responding to log queries. Overridden by -fd-query")
	flag.StringVar(&socketLogPath, "socket-log", "/var/run/linuxkit-external-logging.sock", "unix domain socket to listen for new fds to add to log. Overridden by -fd-log")
	flag.IntVar(&passedLogFD, "fd-log", -1, "an existing SOCK_DGRAM socket for receiving fd's. Overrides -socket-log.")
	flag.IntVar(&passedQueryFD, "fd-query", -1, "an existing SOCK_STREAM for receiving log read requets. Overrides -socket-query.")
	flag.IntVar(&linesInBuffer, "max-lines", 5000, "Number of log lines to keep in memory")
	flag.IntVar(&lineMaxLength, "max-line-len", 1024, "Maximum line length recorded. Additional bytes are dropped.")
	flag.BoolVar(&daemonize, "daemonize", false, "Bind sockets and then daemonize.")
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

	if daemonize {
		child := exec.Command(os.Args[0],
			"-fd-log", "3", // connLogFd in ExtraFiles below
			"-fd-query", "4", // connQuery in ExtraFiles below
			"-max-lines", fmt.Sprintf("%d", linesInBuffer),
			"-max-line-len", fmt.Sprintf("%d", lineMaxLength),
		)
		connLogFile, err := connLogFd.File()
		if err != nil {
			log.Fatalf("The -fd-log cannot be represented as a *File: %s", err)
		}
		connQueryFile, err := connQuery.File()
		if err != nil {
			log.Fatalf("The -fd-query cannot be represented as a *File: %s", err)
		}
		child.ExtraFiles = append(child.ExtraFiles, connLogFile, connQueryFile)
		if err := child.Start(); err != nil {
			log.Fatalf("Failed to re-exec: %s", err)
		}
		os.Exit(0)
	}

	logCh := make(chan logEntry)
	fdMsgChan := make(chan fdMessage)
	queryMsgChan := make(chan queryMessage)

	go receiveFdHandler(connLogFd, logCh, fdMsgChan)
	go receiveQueryHandler(connQuery, logCh, queryMsgChan)
	go ringBufferHandler(linesInBuffer, linesInBuffer, logCh, queryMsgChan)

	doLog(logCh, "memlogd started")

	for true {
		select {
		case msg := <-fdMsgChan: // incoming fd
			go readLogFromFd(lineMaxLength, msg.fd, msg.name, logCh)
		}
	}
}
