package vpnkit

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

// Protocol used by the exposed port.
type Protocol string

const (
	// TCP port is exposed
	TCP = Protocol("tcp")
	// UDP port is exposed
	UDP = Protocol("udp")
	// Unix domain socket is exposed
	Unix = Protocol("unix")
)

// Port describes a UDP, TCP port forward or a Unix domain socket forward.
type Port struct {
	Proto      Protocol `json:"proto,omitempty"`    // Proto is the protocol used by the exposed port.
	OutIP      net.IP   `json:"out_ip,omitempty"`   // OutIP is the external IP address.
	OutPort    uint16   `json:"out_port,omitempty"` // OutPort is the external port number.
	OutPath    string   `json:"out_path,omitempty"` // OutPath is the external Unix domain socket.
	InIP       net.IP   `json:"in_ip,omitempty"`    // InIP is the internal IP address.
	InPort     uint16   `json:"in_port,omitempty"`  // InPort is the internal port number.
	InPath     string   `json:"in_path,omitempty"`  // InPath is the internal Unix domain socket.
	Annotation string   `json:"annotation,omitempty"`
}

// String returns a human-readable string
func (p *Port) String() string {
	annotation := ""
	if p.Annotation != "" {
		annotation = p.Annotation + " "
	}
	if p.Proto == Unix {
		return fmt.Sprintf("%s%s forward from %s to %s", annotation, p.Proto, p.OutPath, p.InPath)
	}
	return fmt.Sprintf("%s%s forward from %s:%d to %s:%d", annotation, p.Proto, p.OutIP.String(), p.OutPort, p.InIP.String(), p.InPort)
}

// spec returns a string of the form proto:outIP:outPort:proto:inIP:inPort as
// understood by vpnkit
func (p *Port) spec() string {
	switch p.Proto {
	case "tcp", "udp":
		return fmt.Sprintf("%s:%s:%d:%s:%s:%d", p.Proto, p.OutIP.String(), p.OutPort, p.Proto, p.InIP.String(), p.InPort)
	case "unix":
		return fmt.Sprintf("unix:%s:unix:%s", base64.StdEncoding.EncodeToString([]byte(p.OutPath)), base64.StdEncoding.EncodeToString([]byte(p.InPath)))
	default:
		return "unknown protocol"
	}
}

func parse(name string) (*Port, error) {
	bits := strings.Split(name, ":")
	switch len(bits) {
	case 6:
		outProto := bits[0]
		outIP := net.ParseIP(bits[1])
		outPort, err := strconv.ParseUint(bits[2], 10, 16)
		if err != nil {
			return nil, err
		}
		inProto := bits[3]
		inIP := net.ParseIP(bits[4])
		inPort, err := strconv.ParseUint(bits[5], 10, 16)
		if err != nil {
			return nil, err
		}
		if outProto != inProto {
			return nil, errors.New("Failed to parse port: external proto is " + outProto + " but internal proto is " + inProto)
		}
		return &Port{Protocol(outProto), outIP, uint16(outPort), "", inIP, uint16(inPort), "", ""}, nil
	case 4:
		outProto := bits[0]
		outPathEnc := bits[1]
		outPath, err := base64.StdEncoding.DecodeString(outPathEnc)
		if err != nil {
			return nil, errors.New("Failed to base64 decode " + string(outPath))
		}
		inProto := bits[2]
		inPathEnc := bits[3]
		inPath, err := base64.StdEncoding.DecodeString(inPathEnc)
		if err != nil {
			return nil, errors.New("Failed to base64 decode " + string(inPath))
		}
		if outProto != "unix" || inProto != "unix" {
			return nil, errors.New("Failed to parse path: external proto is " + outProto + " and internal proto is " + inProto)
		}
		return &Port{Protocol(outProto), nil, uint16(0), string(outPath), nil, uint16(0), string(inPath), ""}, nil
	default:
		return nil, errors.New("Failed to parse port spec: " + name)
	}
}
