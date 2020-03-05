package libproxy

import (
	"io"
	"net"
	"strings"
)

// Conn defines a network connection
type Conn interface {
	net.Conn
	CloseWrite() error
}

// ProxyStream data between client and backend, until both are at EOF or quit is closed.
func ProxyStream(client, backend Conn, quit <-chan struct{}) error {
	event := make(chan int64)
	var broker = func(to, from Conn) {
		written, err := io.Copy(to, from)
		if err != nil && err != io.EOF && !errIsBeingClosed(err) {
			log.Println("error copying:", err)
		}
		err = to.CloseWrite()
		if err != nil && !errIsNotConnected(err) && !errIsBeingClosed(err) {
			log.Println("error CloseWrite to:", err)
		}
		event <- written
	}

	go broker(client, backend)
	go broker(backend, client)

	var transferred int64
	for i := 0; i < 2; i++ {
		select {
		case written := <-event:
			transferred += written
		case <-quit:
			// Interrupt the two brokers and "join" them.
			backend.Close()
			for ; i < 2; i++ {
				transferred += <-event
			}
			return nil
		}
	}
	backend.Close()
	return nil
}

func errIsNotConnected(err error) bool {
	return strings.HasSuffix(err.Error(), "is not connected")
}

func errIsConnectionRefused(err error) bool {
	return strings.HasSuffix(err.Error(), "connection refused")
}

func errIsBeingClosed(err error) bool {
	return strings.HasSuffix(err.Error(), "is being closed.")
}
