package libproxy

import (
	//"encoding/binary"
	"errors"
	"net"
	//"strings"
	"sync"
	//"syscall"

	"github.com/Sirupsen/logrus"
)

type udpListener interface {
  ReadFromUDP(b []byte) (int, *net.UDPAddr, error)
	WriteToUDP(b []byte, addr *net.UDPAddr) (int, error)
	Close() error
}

type udpEncapsulator struct {
	conn     *net.Conn
	listener net.Listener
	m        *sync.Mutex
	r        *sync.Mutex
	w        *sync.Mutex
}

func (u *udpEncapsulator) getConn() (net.Conn, error) {
	u.m.Lock()
	defer u.m.Unlock()
	if u.conn != nil {
		return *u.conn, nil
	}
	conn, err := u.listener.Accept()
	if err != nil {
		logrus.Printf("Failed to accept connection: %#v", err)
		return nil, err
	}
	u.conn = &conn
	return conn, nil
}

func (u *udpEncapsulator) ReadFromUDP(b []byte) (int, *net.UDPAddr, error) {
	conn, err := u.getConn()
	if err != nil {
		return 0, nil, err
	}
	u.r.Lock()
	defer u.r.Unlock()
	datagram := &udpDatagram{payload: b}
	err = datagram.Unmarshal(conn)
	if err != nil {
		return 0, nil, err
	}
	return len(datagram.payload), &net.UDPAddr{IP: *datagram.IP, Port: datagram.Port, Zone: datagram.Zone}, nil
}

func (u *udpEncapsulator) WriteToUDP(b []byte, addr *net.UDPAddr) (int, error) {
	conn, err := u.getConn()
	if err != nil {
		return 0, err
	}
	u.w.Lock()
	defer u.w.Unlock()
	datagram := &udpDatagram{payload: b, IP: &addr.IP, Port: addr.Port, Zone: addr.Zone}
	return len(b), datagram.Marshal(conn)
}

func (u *udpEncapsulator) Close() error {
	if u.conn != nil {
		conn := *u.conn
		conn.Close()
	}
	u.listener.Close()
	return nil
}

func NewUDPListener(listener net.Listener) udpListener {
	var m sync.Mutex;
	return &udpEncapsulator{
		conn: nil,
		listener: listener,
		m: &m,
	}
}

type udpDatagram struct {
	payload []byte
	IP      *net.IP
	Port    int
	Zone    string
}

func (u *udpDatagram) Marshal(conn net.Conn) error {
	return errors.New("Marshal unimplemented")
}

func (u *udpDatagram) Unmarshal(conn net.Conn) error {
	return errors.New("Unmarshal unimplemented")
}
