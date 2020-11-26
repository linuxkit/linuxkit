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

func TestASFPresencePongDecodeFromBytes(t *testing.T) {
	b, err := hex.DecodeString("000011be000000008100000000000000")
	if err != nil {
		t.Fatalf("Failed to decode ASF Presence Pong message")
	}

	pp := &ASFPresencePong{}
	if err := pp.DecodeFromBytes(b, gopacket.NilDecodeFeedback); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !bytes.Equal(pp.BaseLayer.Payload, []byte{}) {
		t.Errorf("payload is %v, want %v", pp.BaseLayer.Payload, b)
	}
	if !bytes.Equal(pp.BaseLayer.Contents, b) {
		t.Errorf("contents is %v, want %v", pp.BaseLayer.Contents, b)
	}
	if pp.Enterprise != ASFRMCPEnterprise {
		t.Errorf("want enterprise %v, got %v", ASFRMCPEnterprise, pp.Enterprise)
	}
	if !bytes.Equal(pp.OEM[:], make([]byte, 4)) {
		t.Errorf("want null OEM, got %v", pp.OEM[:])
	}
	if !pp.IPMI {
		t.Errorf("want IPMI, got false")
	}
	if !pp.ASFv1 {
		t.Errorf("want ASFv1, got false")
	}
	if pp.SecurityExtensions {
		t.Errorf("do not want security extensions, got true")
	}
	if pp.DASH {
		t.Errorf("do not want DASH, got true")
	}
}

func TestASFPresencePongSupportsDCMI(t *testing.T) {
	table := []struct {
		layer *ASFPresencePong
		want  bool
	}{
		{
			&ASFPresencePong{
				Enterprise: ASFRMCPEnterprise,
				IPMI:       true,
				ASFv1:      true,
			},
			false,
		},
		{
			&ASFPresencePong{
				Enterprise: ASFDCMIEnterprise,
				IPMI:       false,
				ASFv1:      true,
			},
			false,
		},
		{
			&ASFPresencePong{
				Enterprise: ASFDCMIEnterprise,
				IPMI:       true,
				ASFv1:      false,
			},
			false,
		},
		{
			&ASFPresencePong{
				Enterprise: ASFDCMIEnterprise,
				IPMI:       true,
				ASFv1:      true,
			},
			true,
		},
	}
	for _, test := range table {
		got := test.layer.SupportsDCMI()
		if got != test.want {
			t.Errorf("%v SupportsDCMI() = %v, want %v", test.layer, got, test.want)
		}
	}
}

func serializeASFPresencePong(pp *ASFPresencePong) ([]byte, error) {
	sb := gopacket.NewSerializeBuffer()
	err := pp.SerializeTo(sb, gopacket.SerializeOptions{})
	return sb.Bytes(), err
}

func TestASFPresencePongSerializeTo(t *testing.T) {
	table := []struct {
		layer *ASFPresencePong
		want  []byte
	}{
		{
			&ASFPresencePong{
				Enterprise: ASFRMCPEnterprise,
				IPMI:       true,
				ASFv1:      true,
			},
			[]byte{0, 0, 0x11, 0xbe, 0, 0, 0, 0, 0x81, 0, 0, 0, 0, 0, 0, 0},
		},
		{
			&ASFPresencePong{
				Enterprise:         1234,
				OEM:                [4]byte{1, 2, 3, 4},
				ASFv1:              true,
				SecurityExtensions: true,
				DASH:               true,
			},
			[]byte{0, 0, 0x4, 0xd2, 1, 2, 3, 4, 0x01, 0xa0, 0, 0, 0, 0, 0, 0},
		},
	}
	for _, test := range table {
		b, err := serializeASFPresencePong(test.layer)
		switch {
		case err != nil && test.want != nil:
			t.Errorf("serialize %v failed with %v, wanted %v", test.layer,
				err, test.want)
		case err == nil && !bytes.Equal(b, test.want):
			t.Errorf("serialize %v = %v, want %v", test.layer, b, test.want)
		}
	}
}
