package libproxy

import (
	"encoding/binary"
	"fmt"
	"io"
)

type handshake struct {
	// In future we could add flags here for feature negotiation.
	// The length is dynamic but it must be < 64k.
	payload []byte // uninterpreted
}

const handshakeMagic = "https://github.com/moby/vpnkit multiplexer protocol\n"

func unmarshalHandshake(r io.Reader) (*handshake, error) {
	magic := make([]byte, len(handshakeMagic))
	if err := binary.Read(r, binary.LittleEndian, &magic); err != nil {
		return nil, err
	}
	if string(magic) != handshakeMagic {
		return nil, fmt.Errorf("not a connection to a mulitplexer; received bad magic string '%s'", string(magic))
	}

	var length uint16
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		// it will be common to fail here with io.EOF
		return nil, err
	}
	payload := make([]byte, length)
	if err := binary.Read(r, binary.LittleEndian, &payload); err != nil {
		return nil, err
	}
	return &handshake{
		payload: payload,
	}, nil
}

func (h *handshake) Write(w io.Writer) error {
	magic := []byte(handshakeMagic)
	if err := binary.Write(w, binary.LittleEndian, magic); err != nil {
		return err
	}
	length := uint16(len(h.payload))
	if err := binary.Write(w, binary.LittleEndian, length); err != nil {
		return err
	}
	return binary.Write(w, binary.LittleEndian, h.payload)
}
