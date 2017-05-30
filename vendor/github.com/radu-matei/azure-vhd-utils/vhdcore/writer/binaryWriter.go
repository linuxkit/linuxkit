package writer

import (
	"encoding/binary"
	"io"
)

// bufferSizeInBytes is the size of the buffer used by BinaryWriter
//
const bufferSizeInBytes = 16

// BinaryWriter is the writer which can be used to write values of primitive types to a writer
// The writer supports writing data both in little-endian or big-endian format.
//
type BinaryWriter struct {
	buffer []byte
	order  binary.ByteOrder
	to     io.WriterAt
	Size   int64
}

// NewBinaryWriter creates a new instance of BinaryWriter, to is the underlying data source
// to write to, order is the byte order used to encode the data in the source, size is the
// length of the data source in bytes.
//
func NewBinaryWriter(to io.WriterAt, order binary.ByteOrder, size int64) *BinaryWriter {
	return &BinaryWriter{
		buffer: make([]byte, bufferSizeInBytes),
		order:  order,
		to:     to,
		Size:   size,
	}
}

// WriteBytes writes a byte slice to the underlying writer at offset off.
//
func (w *BinaryWriter) WriteBytes(off int64, value []byte) {
	w.to.WriteAt(value, off)
}

// WriteByte write a byte value to the underlying writer at offset off.
//
func (w *BinaryWriter) WriteByte(off int64, value byte) {
	w.buffer[0] = value
	w.to.WriteAt(w.buffer[:1], off)
}

// WriteBoolean write a boolean value to the underlying writer at offset off.
//
func (w *BinaryWriter) WriteBoolean(off int64, value bool) {
	if value {
		w.buffer[0] = 1
	} else {
		w.buffer[0] = 0
	}

	w.to.WriteAt(w.buffer[:1], off)
}

// WriteInt16 encodes an int16 and write it in the underlying writer at offset off.
//
func (w *BinaryWriter) WriteInt16(off int64, value int16) {
	w.order.PutUint16(w.buffer, uint16(value))
	w.to.WriteAt(w.buffer[:2], off)
}

// WriteUInt16 encodes an uint16 and write it in the underlying writer at offset off.
//
func (w *BinaryWriter) WriteUInt16(off int64, value uint16) {
	w.order.PutUint16(w.buffer, value)
	w.to.WriteAt(w.buffer[:2], off)
}

// WriteInt32 encodes an int32 and write it in the underlying writer at offset off.
//
func (w *BinaryWriter) WriteInt32(off int64, value int32) {
	w.order.PutUint32(w.buffer, uint32(value))
	w.to.WriteAt(w.buffer[:4], off)
}

// WriteUInt32 encodes an uint32 and write it in the underlying writer at offset off.
//
func (w *BinaryWriter) WriteUInt32(off int64, value uint32) {
	w.order.PutUint32(w.buffer, value)
	w.to.WriteAt(w.buffer[:4], off)
}

// WriteInt64 encodes an int64 and write it in the underlying writer at offset off.
//
func (w *BinaryWriter) WriteInt64(off int64, value int64) {
	w.order.PutUint64(w.buffer, uint64(value))
	w.to.WriteAt(w.buffer[:8], off)
}

// WriteUInt64 encodes an uint64 and write it in the underlying writer at offset off.
//
func (w *BinaryWriter) WriteUInt64(off int64, value uint64) {
	w.order.PutUint64(w.buffer, value)
	w.to.WriteAt(w.buffer[:8], off)
}

// WriteString writes a string to the underlying writer at offset off.
//
func (w *BinaryWriter) WriteString(off int64, value string) {
	w.to.WriteAt([]byte(value), off)
}
