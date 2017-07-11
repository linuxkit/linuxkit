package writer

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
	"unsafe"

	"github.com/radu-matei/azure-vhd-utils/vhdcore"
)

// VhdWriter is the writer used by various components responsible for writing header and
// footer of the VHD.
//
type VhdWriter struct {
	*BinaryWriter
}

// NewVhdWriter creates new instance of the VhdWriter, that writes to the underlying target,
// size is the size of the target in bytes.
//
func NewVhdWriter(target io.WriterAt, size int64) *VhdWriter {
	var order binary.ByteOrder
	if isLittleEndian() {
		order = binary.BigEndian
	} else {
		order = binary.LittleEndian
	}
	return &VhdWriter{NewBinaryWriter(target, order, size)}
}

// NewVhdWriterFromByteSlice creates a new instance of VhdWriter, that uses the given byte
// slice as the underlying target to write to.
//
func NewVhdWriterFromByteSlice(b []byte) *VhdWriter {
	return NewVhdWriter(ByteSliceWriteAt(b), int64(len(b)))
}

// WriteTimeStamp writes vhd timestamp represented by the given time to underlying source
// starting at byte offset off.
//
func (r *VhdWriter) WriteTimeStamp(off int64, time *time.Time) {
	vhdTimeStamp := vhdcore.NewVhdTimeStamp(time)
	r.WriteUInt32(off, vhdTimeStamp.TotalSeconds)
}

// ByteSliceWriteAt is a type that satisfies io.WriteAt interface for byte slice.
//
type ByteSliceWriteAt []byte

// WriteAt copies len(b) bytes to the byte slice starting at byte offset off. It returns the number
// of bytes copied and an error, if any. WriteAt returns a non-nil error when n != len(b).
//
func (s ByteSliceWriteAt) WriteAt(b []byte, off int64) (n int, err error) {
	if off < 0 || off > int64(len(s)) {
		err = fmt.Errorf("Index %d is out of the boundary %d", off, len(s)-1)
		return
	}

	n = copy(s[off:], b)
	if n != len(b) {
		err = fmt.Errorf("Could write only %d bytes, as source is %d bytes and destination is %d bytes", n, len(b), len(s))
	}

	return
}

// isLittleEndian returns true if the host machine is little endian, false for big endian.
//
func isLittleEndian() bool {
	var i int32 = 0x01020304
	u := unsafe.Pointer(&i)
	pb := (*byte)(u)
	b := *pb
	return (b == 0x04)
}
