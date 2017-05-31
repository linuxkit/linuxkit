/*
* Functions in this file are used to forward syslog messages to the
* host and must be quite careful about their own logging. In general
* error messages should go via the console log.Logger defined in this
* file.
 */
package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/rneugeba/virtsock/go/hvsock"
	"github.com/rneugeba/virtsock/go/vsock"
)

var (
	console     *log.Logger
	currentConn vConn

	alreadyConnectedOnce bool

	/* When writing we don't discover e.g. EPIPE until the _next_
	 * attempt to write. Therefore we keep a copy of the last
	 * message sent such that we can repeat it after an error.
	 *
	 * Note that this is imperfect since their can be multiple
	 * messages in flight at the point a connection collapses
	 * which will then be lost. This only handles the delayed
	 * notification of such an error to this code.
	 */
	lastMessage []byte
)

/* rfc5425 like scheme, see section 4.3 */
func rfc5425Write(conn vConn, buf []byte) error {

	msglen := fmt.Sprintf("%d ", len(buf))

	_, err := conn.Write([]byte(msglen))
	/* XXX todo, check for serious vs retriable errors */
	if err != nil {
		console.Printf("Error in length write: %s", err)
		return err
	}

	_, err = conn.Write(buf)
	/* XXX todo, check for serious vs retriable errors */
	if err != nil {
		console.Printf("Error in buf write: %s", err)
	}

	return err
}

func forwardSyslogDatagram(buf []byte, portstr string) error {
	for try := 0; try < 2; try++ {
		conn := currentConn
		if conn == nil {
			if strings.Contains(portstr, "-") {
				svcid, err := hvsock.GuidFromString(portstr)
				if err != nil {
					console.Fatalln("Failed to parse GUID", portstr, err)
				}

				conn, err = hvsock.Dial(hvsock.HypervAddr{VmId: hvsock.GUID_WILDCARD, ServiceId: svcid})
				if err != nil {
					console.Printf("Failed to dial hvsock port: %s", err)
					continue
				}
			} else {
				port, err := strconv.ParseUint(portstr, 10, 32)
				if err != nil {
					console.Fatalln("Can't convert %s to a uint.", portstr, err)
				}

				conn, err = vsock.Dial(vsock.VSOCK_CID_HOST, uint(port))
				if err != nil {
					console.Printf("Failed to dial vsock port %d: %s", port, err)
					continue
				}
			}

			conn.CloseRead()

			/*
			 * Only log on reconnection, not the initial connection since
			 * that is mostly uninteresting
			 */
			if alreadyConnectedOnce {
				console.Printf("Opened new conn to %s: %#v", portstr, conn)
			}
			alreadyConnectedOnce = true

			if lastMessage != nil {
				console.Printf("Replaying last message: %s", lastMessage)
				err := rfc5425Write(conn, lastMessage)
				if err != nil {
					conn.Close()
					continue
				}
				lastMessage = nil
			}

			currentConn = conn
		}

		err := rfc5425Write(conn, buf)
		if err != nil {
			currentConn.Close()
			currentConn = nil
			console.Printf("Failed to write: %s", string(buf))
			continue
		}

		if try > 0 {
			console.Printf("Forwarded on attempt %d: %s", try+1, string(buf))
		}

		// Keep a copy in case we get an EPIPE from the next write
		lastMessage = make([]byte, len(buf))
		copy(lastMessage, buf)

		return nil
	}

	lastMessage = nil // No point repeating this now
	return errors.New("Failed to send datagram, too many retries")
}

func handleSyslogForward(cfg string) {
	// logging to the default syslog while trying to do syslog
	// forwarding would result in infinite loops, so log all
	// messages in this callchain to the console instead.
	logFile, err := os.OpenFile("/dev/console", os.O_WRONLY, 0)
	if err != nil {
		/* What are the chances of this going anywhere useful... */
		log.Fatalln("Failed to open /dev/console for syslog forward logging", err)
	}

	console = log.New(logFile, "vsyslog: ", log.LstdFlags)

	s := strings.SplitN(cfg, ":", 2)
	if len(s) != 2 {
		console.Fatalf("Failed to parse: %s", cfg)
	}
	vsock := s[0]
	usock := s[1]

	err = os.Remove(usock)
	if err != nil && !os.IsNotExist(err) {
		console.Printf("Failed to remove %s: %s", usock, err)
		/* Try and carry on... */
	}

	l, err := net.ListenUnixgram("unixgram", &net.UnixAddr{Name: usock, Net: "unixgram"})
	if err != nil {
		console.Fatalf("Failed to listen to unixgram:%s: %s", usock, err)
	}

	var buf [4096]byte // Ugh, no way to peek at the next message's size in Go
	for {
		r, err := l.Read(buf[:])
		if err != nil {
			console.Fatalf("Failed to read: %s", err)
		}

		err = forwardSyslogDatagram(buf[:r], vsock)
		if err != nil {
			console.Printf("Failed to send log: %s: %s",
				err, string(buf[:r]))
		}
	}
}
