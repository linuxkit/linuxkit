// Copyright 2016, Google, Inc. All rights reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

package layers

import (
	"reflect"
	"testing"

	"github.com/google/gopacket"
)

func TestTCPOptionKindString(t *testing.T) {
	testData := []struct {
		o *TCPOption
		s string
	}{
		{&TCPOption{
			OptionType:   TCPOptionKindNop,
			OptionLength: 1,
		},
			"TCPOption(NOP:)"},
		{&TCPOption{
			OptionType:   TCPOptionKindMSS,
			OptionLength: 4,
			OptionData:   []byte{0x12, 0x34},
		},
			"TCPOption(MSS:4660 0x1234)"},
		{&TCPOption{
			OptionType:   TCPOptionKindTimestamps,
			OptionLength: 10,
			OptionData:   []byte{0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x01},
		},
			"TCPOption(Timestamps:2/1 0x0000000200000001)"}}

	for _, tc := range testData {
		if s := tc.o.String(); s != tc.s {
			t.Errorf("expected %#v string to be %s, got %s", tc.o, tc.s, s)
		}
	}
}

func TestTCPSerializePadding(t *testing.T) {
	tcp := &TCP{}
	tcp.Options = append(tcp.Options, TCPOption{
		OptionType:   TCPOptionKindNop,
		OptionLength: 1,
	})
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true}
	err := gopacket.SerializeLayers(buf, opts, tcp)
	if err != nil {
		t.Fatal(err)
	}
	if len(buf.Bytes())%4 != 0 {
		t.Errorf("TCP data of len %d not padding to 32 bit boundary", len(buf.Bytes()))
	}
}

// testPacketTCPOptionDecode is the packet:
//   16:17:26.239051 IP 192.168.0.1.12345 > 192.168.0.2.54321: Flags [S], seq 3735928559:3735928563, win 0, options [mss 8192,eol], length 4
//   	0x0000:  0000 0000 0001 0000 0000 0001 0800 4500  ..............E.
//   	0x0010:  0034 0000 0000 8006 b970 c0a8 0001 c0a8  .4.......p......
//   	0x0020:  0002 3039 d431 dead beef 0000 0000 7002  ..09.1........p.
//   	0x0030:  0000 829c 0000 0204 2000 0000 0000 5465  ..............Te
//   	0x0040:  7374                                     st
var testPacketTCPOptionDecode = []byte{
	0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x08, 0x00, 0x45, 0x00,
	0x00, 0x34, 0x00, 0x00, 0x00, 0x00, 0x80, 0x06, 0xb9, 0x70, 0xc0, 0xa8, 0x00, 0x01, 0xc0, 0xa8,
	0x00, 0x02, 0x30, 0x39, 0xd4, 0x31, 0xde, 0xad, 0xbe, 0xef, 0x00, 0x00, 0x00, 0x00, 0x70, 0x02,
	0x00, 0x00, 0x82, 0x9c, 0x00, 0x00, 0x02, 0x04, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x54, 0x65,
	0x73, 0x74,
}

func TestPacketTCPOptionDecode(t *testing.T) {
	p := gopacket.NewPacket(testPacketTCPOptionDecode, LinkTypeEthernet, gopacket.Default)
	if p.ErrorLayer() != nil {
		t.Error("Failed to decode packet:", p.ErrorLayer().Error())
	}
	tcp := p.Layer(LayerTypeTCP).(*TCP)
	if tcp == nil {
		t.Error("Expected TCP layer, but got none")
	}

	expected := []TCPOption{
		{
			OptionType:   TCPOptionKindMSS,
			OptionLength: 4,
			OptionData:   []byte{32, 00},
		},
		{
			OptionType:   TCPOptionKindEndList,
			OptionLength: 1,
		},
	}

	if !reflect.DeepEqual(expected, tcp.Options) {
		t.Errorf("expected options to be %#v, but got %#v", expected, tcp.Options)
	}
}
