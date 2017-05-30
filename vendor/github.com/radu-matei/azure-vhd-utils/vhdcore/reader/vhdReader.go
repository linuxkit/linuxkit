package reader

import (
	"bytes"
	"encoding/binary"
	"time"
	"unsafe"

	"github.com/radu-matei/azure-vhd-utils/vhdcore"
)

// VhdReader is the reader used by various components responsible for reading different
// segments of VHD such as header, footer, BAT, block, bitmap and sector.
//
type VhdReader struct {
	*BinaryReader
}

// NewVhdReader creates new instance of the VhdReader, that reads from the underlying
// source, size is the size of the source in bytes.
//
func NewVhdReader(source ReadAtReader, size int64) *VhdReader {
	var order binary.ByteOrder
	if isLittleEndian() {
		order = binary.BigEndian
	} else {
		order = binary.LittleEndian
	}
	return &VhdReader{NewBinaryReader(source, order, size)}
}

// NewVhdReaderFromByteSlice creates a new instance of VhdReader, that uses the given
// byte slice as the underlying source to read from.
//
func NewVhdReaderFromByteSlice(b []byte) *VhdReader {
	source := bytes.NewReader(b)
	return NewVhdReader(source, int64(len(b)))
}

// ReadDateTime reads an encoded vhd timestamp from underlying source starting at byte
// offset off and return it as a time.Time.
//
func (r *VhdReader) ReadDateTime(off int64) (*time.Time, error) {
	d, err := r.ReadUInt32(off)
	if err != nil {
		return nil, err
	}
	vhdDateTime := vhdcore.NewVhdTimeStampFromSeconds(d).ToDateTime()
	return &vhdDateTime, nil
}

// isLittleEndian returns true if the host machine is little endian, false for
// big endian
//
func isLittleEndian() bool {
	var i int32 = 0x01020304
	u := unsafe.Pointer(&i)
	pb := (*byte)(u)
	b := *pb
	return (b == 0x04)
}
