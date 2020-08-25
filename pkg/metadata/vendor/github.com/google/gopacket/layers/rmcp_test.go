// Copyright 2019 The GoPacket Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file in the root of the source tree.

package layers

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/google/gopacket"
)

func TestRMCPDecodeFromBytes(t *testing.T) {
	b, err := hex.DecodeString("0600ff06")
	if err != nil {
		t.Fatalf("Failed to decode RMCP message")
	}

	rmcp := &RMCP{}
	if err := rmcp.DecodeFromBytes(b, gopacket.NilDecodeFeedback); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !bytes.Equal(rmcp.BaseLayer.Payload, []byte{}) {
		t.Errorf("payload is %v, want %v", rmcp.BaseLayer.Payload, b)
	}
	if !bytes.Equal(rmcp.BaseLayer.Contents, b) {
		t.Errorf("contents is %v, want %v", rmcp.BaseLayer.Contents, b)
	}
	if rmcp.Version != RMCPVersion1 {
		t.Errorf("version is %v, want %v", rmcp.Version, RMCPVersion1)
	}
	if rmcp.Sequence != 0xFF {
		t.Errorf("sequence is %v, want %v", rmcp.Sequence, 0xFF)
	}
	if rmcp.Ack {
		t.Errorf("ack is true, want false")
	}
	if rmcp.Class != RMCPClassASF {
		t.Errorf("class is %v, want %v", rmcp.Class, RMCPClassASF)
	}
}

func serializeRMCP(rmcp *RMCP) ([]byte, error) {
	sb := gopacket.NewSerializeBuffer()
	err := rmcp.SerializeTo(sb, gopacket.SerializeOptions{})
	return sb.Bytes(), err
}

func TestRMCPTestSerializeTo(t *testing.T) {
	table := []struct {
		layer *RMCP
		want  []byte
	}{
		{
			&RMCP{
				Version:  RMCPVersion1,
				Sequence: 1,
				Ack:      false,
				Class:    RMCPClassASF,
			},
			[]byte{0x6, 0x0, 0x1, 0x6},
		},
		{
			&RMCP{
				Version:  RMCPVersion1,
				Sequence: 0xFF,
				Ack:      true,
				Class:    RMCPClassIPMI,
			},
			[]byte{0x6, 0x0, 0xFF, 0x87},
		},
	}
	for _, test := range table {
		b, err := serializeRMCP(test.layer)
		switch {
		case err != nil && test.want != nil:
			t.Errorf("serialize %v failed with %v, wanted %v", test.layer,
				err, test.want)
		case err == nil && !bytes.Equal(b, test.want):
			t.Errorf("serialize %v = %v, want %v", test.layer, b, test.want)
		}
	}
}
