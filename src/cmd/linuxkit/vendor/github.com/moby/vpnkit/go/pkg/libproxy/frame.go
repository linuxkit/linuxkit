package libproxy

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

// Proto is the protocol of the flow
type Proto uint8

const (
	// TCP flow
	TCP Proto = 1
	// UDP flow
	UDP Proto = 2
	// Unix domain socket flow
	Unix Proto = 3
)

// Destination refers to a listening TCP or UDP service
type Destination struct {
	Proto Proto
	IP    net.IP
	Port  uint16
	Path  string
}

func (d Destination) String() string {
	switch d.Proto {
	case TCP:
		return fmt.Sprintf("TCP:%s:%d", d.IP.String(), d.Port)
	case UDP:
		return fmt.Sprintf("UDP:%s:%d", d.IP.String(), d.Port)
	case Unix:
		return fmt.Sprintf("Unix:%s", d.Path)
	}
	return "Unknown"
}

// Read header which describes TCP/UDP and destination IP:port
func unmarshalDestination(r io.Reader) (Destination, error) {
	d := Destination{}
	if err := binary.Read(r, binary.LittleEndian, &d.Proto); err != nil {
		return d, err
	}
	switch d.Proto {
	case TCP, UDP:
		var length uint16
		// IP length
		if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
			return d, err
		}
		d.IP = make([]byte, length)
		if err := binary.Read(r, binary.LittleEndian, &d.IP); err != nil {
			return d, err
		}
		if err := binary.Read(r, binary.LittleEndian, &d.Port); err != nil {
			return d, err
		}
	case Unix:
		var length uint16
		// String length
		if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
			return d, err
		}
		path := make([]byte, length)
		if err := binary.Read(r, binary.LittleEndian, &path); err != nil {
			return d, err
		}
		d.Path = string(path)
	}
	return d, nil
}

func (d Destination) Write(w io.Writer) error {
	if err := binary.Write(w, binary.LittleEndian, d.Proto); err != nil {
		return err
	}
	switch d.Proto {
	case TCP, UDP:
		b := []byte(d.IP)
		length := uint16(len(b))
		if err := binary.Write(w, binary.LittleEndian, length); err != nil {
			return err
		}
		if err := binary.Write(w, binary.LittleEndian, b); err != nil {
			return err
		}
		if err := binary.Write(w, binary.LittleEndian, d.Port); err != nil {
			return err
		}
	case Unix:
		b := []byte(d.Path)
		length := uint16(len(b))
		if err := binary.Write(w, binary.LittleEndian, length); err != nil {
			return err
		}
		if err := binary.Write(w, binary.LittleEndian, b); err != nil {
			return err
		}
	}
	return nil
}

// Size returns the marshalled size in bytes
func (d Destination) Size() int {
	switch d.Proto {
	case TCP, UDP:
		return 1 + 2 + len(d.IP) + 2
	case Unix:
		return 1 + 2 + len(d.Path)
	}
	return 0
}

// Connection indicates whether the connection will use multiplexing or not.
type Connection int8

func (c Connection) String() string {
	switch c {
	case Dedicated:
		return "Dedicated"
	case Multiplexed:
		return "Multiplexed"
	default:
		return "Unknown"
	}
}

const (
	// Dedicated means this connection will not use multiplexing
	Dedicated Connection = iota + 1
	// Multiplexed means this connection will contain labelled sub-connections mixed together
	Multiplexed
)

// OpenFrame requests to connect to a proxy backend
type OpenFrame struct {
	Connection  Connection // Connection describes whether the opened connection should be dedicated or multiplexed
	Destination Destination
}

func unmarshalOpen(r io.Reader) (*OpenFrame, error) {
	o := &OpenFrame{}
	if err := binary.Read(r, binary.LittleEndian, &o.Connection); err != nil {
		return nil, err
	}
	d, err := unmarshalDestination(r)
	if err != nil {
		return nil, err
	}
	o.Destination = d
	return o, nil
}

func (o *OpenFrame) Write(w io.Writer) error {
	if err := binary.Write(w, binary.LittleEndian, o.Connection); err != nil {
		return err
	}
	return o.Destination.Write(w)
}

// Size returns the marshalled size of the Open message
func (o *OpenFrame) Size() int {
	return 1 + o.Destination.Size()
}

// CloseFrame requests to disconnect from a proxy backend
type CloseFrame struct {
}

// ShutdownFrame requests to close the write channel to a proxy backend
type ShutdownFrame struct {
}

// DataFrame is the header of a frame containing user data
type DataFrame struct {
	payloadlen uint32
}

func unmarshalData(r io.Reader) (*DataFrame, error) {
	d := &DataFrame{}
	err := binary.Read(r, binary.LittleEndian, &d.payloadlen)
	return d, err
}

func (d *DataFrame) Write(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, d.payloadlen)
}

// Size returns the marshalled size of the data payload header
func (d *DataFrame) Size() int {
	return 4
}

// WindowFrame is a window advertisement message
type WindowFrame struct {
	seq uint64
}

func unmarshalWindow(r io.Reader) (*WindowFrame, error) {
	w := &WindowFrame{}
	err := binary.Read(r, binary.LittleEndian, &w.seq)
	return w, err
}

func (win *WindowFrame) Write(w io.Writer) error {
	return binary.Write(w, binary.LittleEndian, win.seq)
}

// Size returned the marshalled size of the Window payload
func (win *WindowFrame) Size() int {
	return 8
}

// Command is the action requested by a message.
type Command int8

const (
	// Open requests to open a connection to a backend service.
	Open Command = iota + 1
	// Close requests and then acknowledges the close of a sub-connection
	Close
	// Shutdown indicates that no more data will be written in this direction
	Shutdown
	// Data is a payload of a connection/sub-connection
	Data
	// Window is permission to send and consume buffer space
	Window
)

// Frame is the low-level message sent to the multiplexer
type Frame struct {
	Command  Command // Command is the action erquested
	ID       uint32  // Id of the sub-connection, managed by the client
	open     *OpenFrame
	close    *CloseFrame
	shutdown *ShutdownFrame
	window   *WindowFrame
	data     *DataFrame
}

func unmarshalFrame(r io.Reader) (*Frame, error) {
	f := &Frame{}
	var totallen uint16
	if err := binary.Read(r, binary.LittleEndian, &totallen); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &f.Command); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &f.ID); err != nil {
		return nil, err
	}
	switch f.Command {
	case Open:
		o, err := unmarshalOpen(r)
		if err != nil {
			return nil, err
		}
		f.open = o
	case Close:
		// no payload
	case Shutdown:
		// no payload
	case Window:
		w, err := unmarshalWindow(r)
		if err != nil {
			return nil, err
		}
		f.window = w
	case Data:
		d, err := unmarshalData(r)
		if err != nil {
			return nil, err
		}
		f.data = d
	}
	return f, nil
}

func (f *Frame) Write(w io.Writer) error {
	frameLen := uint16(f.Size())
	if err := binary.Write(w, binary.LittleEndian, frameLen); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, f.Command); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, f.ID); err != nil {
		return err
	}
	switch f.Command {
	case Open:
		if err := f.open.Write(w); err != nil {
			return err
		}
	case Close:
		// no payload
	case Shutdown:
		// no payload
	case Window:
		if err := f.window.Write(w); err != nil {
			return err
		}
	case Data:
		if err := f.data.Write(w); err != nil {
			return err
		}
	}
	return nil
}

// Size returns the marshalled size of the frame
func (f *Frame) Size() int {
	// include 2 for the preceeding length field
	len := 2 + 1 + 4
	switch f.Command {
	case Open:
		len = len + f.open.Size()
	case Close:
		// no payload
	case Shutdown:
		// no payload
	case Window:
		len = len + f.window.Size()
	case Data:
		len = len + f.data.Size()
	}
	return len
}

func (f *Frame) String() string {
	switch f.Command {
	case Open:
		return fmt.Sprintf("%d Open %s %s", f.ID, f.open.Connection.String(), f.open.Destination.String())
	case Close:
		return fmt.Sprintf("%d Close", f.ID)
	case Shutdown:
		return fmt.Sprintf("%d Shutdown", f.ID)
	case Window:
		return fmt.Sprintf("%d Window %d", f.ID, f.window.seq)
	case Data:
		return fmt.Sprintf("%d Data length %d", f.ID, f.data.payloadlen)
	default:
		return "unknown"
	}
}

// Window returns the payload of the frame, if it has Command = Window.
func (f *Frame) Window() (*WindowFrame, error) {
	if f.Command != Window {
		return nil, errors.New("Frame is not a Window()")
	}
	return f.window, nil
}

// Open returns the payload of the frame, if it has Command = Open
func (f *Frame) Open() (*OpenFrame, error) {
	if f.Command != Open {
		return nil, errors.New("Frame is not an Open()")
	}
	return f.open, nil
}

// Data returns the payload of the frame, if it has Command = Data
func (f *Frame) Data() (*DataFrame, error) {
	if f.Command != Data {
		return nil, errors.New("Frame is not Data()")
	}
	return f.data, nil
}

// Payload returns the payload of the frame.
func (f *Frame) Payload() interface{} {
	switch f.Command {
	case Open:
		return f.open
	case Close:
		return f.close
	case Shutdown:
		return f.shutdown
	case Window:
		return f.window
	case Data:
		return f.data
	default:
		return nil
	}
}

// NewWindow creates a Window message
func NewWindow(ID uint32, seq uint64) *Frame {
	return &Frame{
		Command: Window,
		ID:      ID,
		window: &WindowFrame{
			seq: seq,
		},
	}
}

// NewOpen creates an open message
func NewOpen(ID uint32, d Destination) *Frame {
	return &Frame{
		Command: Open,
		ID:      ID,
		open: &OpenFrame{
			Connection:  Multiplexed,
			Destination: d,
		},
	}
}

// NewData creates a data header frame
func NewData(ID, payloadlen uint32) *Frame {
	return &Frame{
		Command: Data,
		ID:      ID,
		data: &DataFrame{
			payloadlen: payloadlen,
		},
	}
}

// NewShutdown creates a shutdown frame
func NewShutdown(ID uint32) *Frame {
	return &Frame{
		Command: Shutdown,
		ID:      ID,
	}
}

// NewClose creates a close frame
func NewClose(ID uint32) *Frame {
	return &Frame{
		Command: Close,
		ID:      ID,
	}
}
