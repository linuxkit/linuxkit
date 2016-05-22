package libproxy

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"sync"

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
	length, err := datagram.Unmarshal(conn)
	if err != nil {
		return 0, nil, err
	}
	udpAddr := net.UDPAddr{IP: *datagram.IP, Port: int(datagram.Port), Zone: datagram.Zone}
	return length, &udpAddr, nil
}

func (u *udpEncapsulator) WriteToUDP(b []byte, addr *net.UDPAddr) (int, error) {
	conn, err := u.getConn()
	if err != nil {
		return 0, err
	}
	u.w.Lock()
	defer u.w.Unlock()
	datagram := &udpDatagram{payload: b, IP: &addr.IP, Port: uint16(addr.Port), Zone: addr.Zone}
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

func NewUDPConn(conn net.Conn) udpListener {
	var m sync.Mutex
	var r sync.Mutex
	var w sync.Mutex
	return &udpEncapsulator{
		conn:     &conn,
		listener: nil,
		m:        &m,
		r:        &r,
		w:        &w,
	}
}

func NewUDPListener(listener net.Listener) udpListener {
	var m sync.Mutex
	var r sync.Mutex
	var w sync.Mutex
	return &udpEncapsulator{
		conn:     nil,
		listener: listener,
		m:        &m,
		r:        &r,
		w:        &w,
	}
}

type udpDatagram struct {
	payload []byte
	IP      *net.IP
	Port    uint16
	Zone    string
}

func (u *udpDatagram) Marshal(conn net.Conn) error {
	// marshal the variable length header to a temporary buffer
	var header bytes.Buffer
	var length uint16
	length = uint16(len(*u.IP))
	if err := binary.Write(&header, binary.LittleEndian, &length); err != nil {
		return err
	}

	if err := binary.Write(&header, binary.LittleEndian, u.IP); err != nil {
		return err
	}

	if err := binary.Write(&header, binary.LittleEndian, &u.Port); err != nil {
		return err
	}

	length = uint16(len(u.Zone))
	if err := binary.Write(&header, binary.LittleEndian, &length); err != nil {
		return err
	}

	if err := binary.Write(&header, binary.LittleEndian, []byte(u.Zone)); err != nil {
		return nil
	}

	length = uint16(len(u.payload))
	if err := binary.Write(&header, binary.LittleEndian, &length); err != nil {
		return nil
	}

	length = uint16(2 + header.Len() + len(u.payload))
	if err := binary.Write(conn, binary.LittleEndian, &length); err != nil {
		return nil
	}
	_, err := io.Copy(conn, &header)
	if err != nil {
		return err
	}
	payload := bytes.NewBuffer(u.payload)
	_, err = io.Copy(conn, payload)
	if err != nil {
		return err
	}
	return nil
}

func (u *udpDatagram) Unmarshal(conn net.Conn) (int, error) {
	var length uint16
	// frame length
	if err := binary.Read(conn, binary.LittleEndian, &length); err != nil {
		return 0, err
	}
	if err := binary.Read(conn, binary.LittleEndian, &length); err != nil {
		return 0, err
	}
	var IP net.IP
	IP = make([]byte, length)
	if err := binary.Read(conn, binary.LittleEndian, &IP); err != nil {
		return 0, err
	}
	u.IP = &IP
	if err := binary.Read(conn, binary.LittleEndian, &u.Port); err != nil {
		return 0, err
	}
	if err := binary.Read(conn, binary.LittleEndian, &length); err != nil {
		return 0, err
	}
	Zone := make([]byte, length)
	if err := binary.Read(conn, binary.LittleEndian, &Zone); err != nil {
		return 0, err
	}
	u.Zone = string(Zone)
	if err := binary.Read(conn, binary.LittleEndian, &length); err != nil {
		return 0, err
	}
	_, err := io.ReadFull(conn, u.payload[0:length])
	if err != nil {
		return 0, err
	}
	return int(length), nil
}
