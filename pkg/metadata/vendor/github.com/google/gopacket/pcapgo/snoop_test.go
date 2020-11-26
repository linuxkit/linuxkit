// Copyright 2019 The GoPacket Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.
package pcapgo

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
	"time"
)

var (
	spHeader = []byte{
		0x73, 0x6E, 0x6F, 0x6F, 0x70, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x02,
		0x00, 0x00, 0x00, 0x04,
	}

	pack = []byte{
		0x00, 0x00, 0x00, 0x2A, 0x00, 0x00, 0x00, 0x2A, 0x00, 0x00, 0x00, 0x44, 0x00, 0x00, 0x00, 0x00, 0x5C, 0xBE, 0xB8, 0x4C, 0x00, 0x0C, 0xB1, 0x47,
		0x7c, 0x5a, 0x1c, 0x49, 0x3c, 0xd1, 0x1e, 0x65, 0x50, 0x7f, 0xb9, 0xca, 0x08, 0x06, 0x00, 0x01, 0x08, 0x00, 0x06, 0x04, 0x00, 0x01, 0x1e, 0x65,
		0x50, 0x7f, 0xb9, 0xca, 0x0a, 0x00, 0x33, 0x68, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0a, 0x00, 0x33, 0x01, 0x00, 0x00,
	}
)

func OpenHandlePack() (buf []byte, handle *SnoopReader, err error) {
	buf = make([]byte, len(spHeader)+len(pack))
	copy(buf, append(spHeader, pack...))
	handle, err = NewSnoopReader(bytes.NewReader(buf))
	return buf, handle, err
}

func equalError(t *testing.T, err error, eq error) {
	if err.Error() != eq.Error() {
		t.Error(err)
	}
}

func equalNil(t *testing.T, err error) {
	if err != nil {
		t.Error(err)
	}
}

func equal(t *testing.T, expected, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		t.Error(fmt.Errorf("Not equal: \nexpected: %s\nactual  : %s", expected, actual))
	}
}

func TestReadHeader(t *testing.T) {
	_, err := NewSnoopReader(bytes.NewReader(spHeader))
	equalNil(t, err)
}

func TestBadHeader(t *testing.T) {
	buf := make([]byte, len(spHeader))
	copy(buf, spHeader)
	buf[6] = 0xff
	_, err := NewSnoopReader(bytes.NewReader(buf))
	equalError(t, err, fmt.Errorf("%s: %s", unknownMagic, "736e6f6f7000ff00"))

	buf[6] = 0x00
	buf[11] = 0x03
	_, err = NewSnoopReader(bytes.NewReader(buf))
	equalError(t, err, fmt.Errorf("%s: %d", unknownVersion, 3))

	buf[11] = 0x02
	buf[15] = 0x0b // linktype 11 is undefined
	_, err = NewSnoopReader(bytes.NewReader(buf))
	equalError(t, err, fmt.Errorf("%s, Code:%d", unkownLinkType, 11))

	buf[15] = 0x04
}

func TestReadPacket(t *testing.T) {
	_, handle, err := OpenHandlePack()
	equalNil(t, err)

	_, _, err = handle.ReadPacketData()
	equalNil(t, err)
}

func TestZeroCopy(t *testing.T) {
	_, handle, err := OpenHandlePack()
	equalNil(t, err)

	var cnt int
	for cnt = 0; ; cnt++ {
		_, _, err := handle.ZeroCopyReadPacketData()
		if err != nil {
			equalError(t, err, fmt.Errorf("EOF"))
			break
		}
	}
	if cnt != 1 {
		t.Error(err)
	}
}

func TestPacketHeader(t *testing.T) {
	_, handle, err := OpenHandlePack()
	equalNil(t, err)
	_, ci, err := handle.ReadPacketData()
	equalNil(t, err)

	equal(t, ci.CaptureLength, 42)
	equal(t, ci.Length, 42)
	equal(t, ci.Timestamp, time.Date(2019, 04, 23, 07, 01, 32, 831815*1000, time.UTC)) //with nanosec

}

func TestBadPacketHeader(t *testing.T) {
	buf, handle, err := OpenHandlePack()
	equalNil(t, err)
	buf[23] = 0x2C
	_, _, err = handle.ReadPacketData()
	equalError(t, err, fmt.Errorf(originalLenExceeded))
	buf[23] = 0x2A
}

func TestBigPacketData(t *testing.T) {
	buf, handle, err := OpenHandlePack()
	equalNil(t, err)
	// increase OriginalLen
	buf[19] = 0x00
	buf[18] = 0x11
	// increase includedLen
	buf[23] = 0x00
	buf[22] = 0x11
	_, _, err = handle.ReadPacketData()
	equalError(t, err, fmt.Errorf(captureLenExceeded))
	buf[23] = 0x44
	buf[22] = 0x00
	buf[19] = 0x44
	buf[18] = 0x00
}

func TestLinkType(t *testing.T) {
	_, handle, err := OpenHandlePack()
	equalNil(t, err)
	_, err = handle.LinkType()
	equalNil(t, err)
}

func TestNotOverlapBuf(t *testing.T) {
	buf := make([]byte, len(spHeader)+len(pack)*2)
	packs := append(spHeader, pack...)
	copy(buf, append(packs, pack...))
	handle, err := NewSnoopReader(bytes.NewReader(buf))
	equalNil(t, err)
	overlap, _, err := handle.ReadPacketData()
	equalNil(t, err)
	overlap2, _, err := handle.ReadPacketData()
	overlap[30] = 0xff
	if overlap[30] == overlap2[30] {
		t.Error(fmt.Errorf("Should not be: %x", overlap[30]))
	}

}

func GeneratePacks(num int) []byte {
	buf := make([]byte, len(spHeader)+(len(pack)*num))
	packs := append(spHeader, pack...)
	for i := 1; i < num; i++ {
		packs = append(packs, pack...)
	}
	copy(buf, packs)
	return buf
}
func BenchmarkReadPacketData(b *testing.B) {
	buf := GeneratePacks(100)
	handle, _ := NewSnoopReader(bytes.NewReader(buf))
	for n := 0; n < b.N; n++ {
		_, _, _ = handle.ReadPacketData()
	}
}

func BenchmarkZeroCopyReadPacketData(b *testing.B) {
	buf := GeneratePacks(100)
	handle, _ := NewSnoopReader(bytes.NewReader(buf))
	for n := 0; n < b.N; n++ {
		_, _, _ = handle.ZeroCopyReadPacketData()
	}
}
