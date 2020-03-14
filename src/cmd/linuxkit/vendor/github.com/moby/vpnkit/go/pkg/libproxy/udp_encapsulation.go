package libproxy

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"time"
)

// UDPListener defines a listener interface to read, write and close a UDP connection
type UDPListener interface {
	ReadFromUDP(b []byte) (int, *net.UDPAddr, error)
	WriteToUDP(b []byte, addr *net.UDPAddr) (int, error)
	Close() error
}

// UDPEncapsulator implements net.Conn and reads and writes UDP datagrams framed within a stream connection
type uDPEncapsulator interface {
	Conn
	ReadFromUDP(b []byte) (int, *net.UDPAddr, error)
	WriteToUDP(b []byte, addr *net.UDPAddr) (int, error)
}

// udpEncapsulator encapsulates a UDP connection and listener
type udpEncapsulator struct {
	conn net.Conn
	m    *sync.Mutex
	r    *sync.Mutex
	w    *sync.Mutex
	addr *net.UDPAddr
}

// ReadFromUDP reads the bytestream from a udpEncapsulator, returning the
// number of bytes read and the unpacked UDPAddr struct
func (u *udpEncapsulator) ReadFromUDP(b []byte) (int, *net.UDPAddr, error) {
	u.r.Lock()
	defer u.r.Unlock()
	datagram := &udpDatagram{payload: b}
	length, err := datagram.Unmarshal(u.conn)
	if err != nil {
		return 0, nil, err
	}
	udpAddr := net.UDPAddr{IP: *datagram.IP, Port: int(datagram.Port), Zone: datagram.Zone}
	return length, &udpAddr, nil
}

// WriteToUDP writes a bytestream to a specified UDPAddr, returning the number
// of bytes successfully written
func (u *udpEncapsulator) WriteToUDP(b []byte, addr *net.UDPAddr) (int, error) {
	u.w.Lock()
	defer u.w.Unlock()
	datagram := &udpDatagram{payload: b, IP: &addr.IP, Port: uint16(addr.Port), Zone: addr.Zone}
	return len(b), datagram.Marshal(u.conn)
}

// Close closes the connection in the udpEncapsulator
func (u *udpEncapsulator) Close() error {
	return u.conn.Close()
}

func (u *udpEncapsulator) CloseWrite() error {
	return nil
}

func (u *udpEncapsulator) Read(b []byte) (int, error) {
	n, _, err := u.ReadFromUDP(b)
	return n, err
}

func (u *udpEncapsulator) Write(b []byte) (int, error) {
	return u.WriteToUDP(b, &net.UDPAddr{})
}

func (u *udpEncapsulator) LocalAddr() net.Addr {
	return u.conn.LocalAddr()
}

func (u *udpEncapsulator) RemoteAddr() net.Addr {
	return u.conn.RemoteAddr()
}

func (u *udpEncapsulator) SetDeadline(t time.Time) error {
	return u.conn.SetDeadline(t)
}

func (u *udpEncapsulator) SetReadDeadline(t time.Time) error {
	return u.conn.SetReadDeadline(t)
}

func (u *udpEncapsulator) SetWriteDeadline(t time.Time) error {
	return u.conn.SetWriteDeadline(t)
}

func (u *udpEncapsulator) Connect(a *net.UDPAddr) {
	u.addr = a
}

// newUDPConn initializes a new UDP connection
func newUDPConn(conn net.Conn) uDPEncapsulator {
	var m sync.Mutex
	var r sync.Mutex
	var w sync.Mutex
	return &udpEncapsulator{
		conn: conn,
		m:    &m,
		r:    &r,
		w:    &w,
	}
}

type udpDatagram struct {
	payload []byte
	IP      *net.IP
	Port    uint16
	Zone    string
}

// Marshal marshals data from the udpDatagram to the provided connection
func (u *udpDatagram) Marshal(w io.Writer) error {
	// marshal the variable length header to a temporary buffer
	var header bytes.Buffer
	length := uint16(len(*u.IP))
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
	if err := binary.Write(w, binary.LittleEndian, &length); err != nil {
		return nil
	}
	_, err := io.Copy(w, &header)
	if err != nil {
		return err
	}
	payload := bytes.NewBuffer(u.payload)
	_, err = io.Copy(w, payload)
	if err != nil {
		return err
	}
	return nil
}

// Unmarshal unmarshals data from the connection to the udpDatagram
func (u *udpDatagram) Unmarshal(r io.Reader) (int, error) {
	var length uint16
	// frame length
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return 0, err
	}
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return 0, err
	}
	var IP net.IP
	IP = make([]byte, length)
	if err := binary.Read(r, binary.LittleEndian, &IP); err != nil {
		return 0, err
	}
	u.IP = &IP
	if err := binary.Read(r, binary.LittleEndian, &u.Port); err != nil {
		return 0, err
	}
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return 0, err
	}
	Zone := make([]byte, length)
	if err := binary.Read(r, binary.LittleEndian, &Zone); err != nil {
		return 0, err
	}
	u.Zone = string(Zone)
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return 0, err
	}
	_, err := io.ReadFull(r, u.payload[0:length])
	if err != nil {
		return 0, err
	}
	return int(length), nil
}
