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

func TestASFDecodeFromBytes(t *testing.T) {
	b, err := hex.DecodeString("000011be4000001000000000000000")
	if err != nil {
		t.Fatalf("Failed to decode ASF message")
	}

	asf := &ASF{}
	if err := asf.DecodeFromBytes(b, gopacket.NilDecodeFeedback); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !bytes.Equal(asf.BaseLayer.Contents, b[:8]) {
		t.Errorf("contents is %v, want %v", asf.BaseLayer.Contents, b[:8])
	}
	if !bytes.Equal(asf.BaseLayer.Payload, b[8:]) {
		t.Errorf("payload is %v, want %v", asf.BaseLayer.Payload, b[8:])
	}
	if asf.Enterprise != ASFRMCPEnterprise {
		t.Errorf("enterprise is %v, want %v", asf.Enterprise, ASFRMCPEnterprise)
	}
	if asf.Type != ASFDataIdentifierPresencePong.Type {
		t.Errorf("type is %v, want %v", asf.Type, ASFDataIdentifierPresencePong)
	}
	if asf.Tag != 0 {
		t.Errorf("tag is %v, want 0", asf.Tag)
	}
	if asf.Length != 16 {
		t.Errorf("length is %v, want 16", asf.Length)
	}
}

func serializeASF(asf *ASF) ([]byte, error) {
	sb := gopacket.NewSerializeBuffer()
	err := asf.SerializeTo(sb, gopacket.SerializeOptions{
		FixLengths: true,
	})
	return sb.Bytes(), err
}

func TestASFSerializeTo(t *testing.T) {
	table := []struct {
		layer *ASF
		want  []byte
	}{
		{
			&ASF{
				ASFDataIdentifier: ASFDataIdentifierPresencePing,
			},
			[]byte{0, 0, 0x11, 0xbe, 0x80, 0, 0, 0},
		},
		{
			&ASF{
				ASFDataIdentifier: ASFDataIdentifierPresencePong,
				// ensures length is being overridden - should be encoded to 0
				// as there is no payload
				Length: 1,
			},
			[]byte{0, 0, 0x11, 0xbe, 0x40, 0, 0, 0},
		},
	}
	for _, test := range table {
		b, err := serializeASF(test.layer)
		switch {
		case err != nil && test.want != nil:
			t.Errorf("serialize %v failed with %v, wanted %v", test.layer,
				err, test.want)
		case err == nil && !bytes.Equal(b, test.want):
			t.Errorf("serialize %v = %v, want %v", test.layer, b, test.want)
		}
	}
}
