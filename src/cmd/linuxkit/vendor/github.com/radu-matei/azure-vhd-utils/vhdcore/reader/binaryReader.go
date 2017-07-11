package reader

import (
	"encoding/binary"
	"fmt"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/common"
	"io"
)

// bufferSizeInBytes is the size of the buffer used by BinaryReader
//
const bufferSizeInBytes = 16

// ReadAtReader interface that composes io.ReaderAt and io.Reader interfaces.
//
type ReadAtReader interface {
	io.ReaderAt
	io.Reader
}

// BinaryReader is the reader which can be used to read values of primitive types from a reader
// The reader supports reading data stored both in little-endian or big-endian format.
//
type BinaryReader struct {
	buffer []byte
	order  binary.ByteOrder
	from   ReadAtReader
	Size   int64
}

// NewBinaryReader creates a new instance of BinaryReader, from is the underlying data source
// to read from, order is the byte order used to encode the data in the source, size is the
// length of the data source in bytes.
//
func NewBinaryReader(from ReadAtReader, order binary.ByteOrder, size int64) *BinaryReader {
	return &BinaryReader{
		buffer: make([]byte, bufferSizeInBytes),
		order:  order,
		from:   from,
		Size:   size,
	}
}

// ReadBytes reads exactly len(buf) bytes from r into buf. It returns the number of bytes
// copied and an error if fewer bytes were read. The error is EOF only if no bytes were
// read. If an EOF happens after reading some but not all the bytes, ReadBytes returns
// ErrUnexpectedEOF. On return, n == len(buf) if and only if err == nil.
//
func (b *BinaryReader) ReadBytes(offset int64, buf []byte) (int, error) {
	return b.from.ReadAt(buf, offset)
}

// ReadByte reads a byte from underlying source starting at byte offset off and returns it.
//
func (b *BinaryReader) ReadByte(offset int64) (byte, error) {
	if _, err := b.readToBuffer(1, offset); err != nil {
		return 0, err
	}

	return b.buffer[0], nil
}

// ReadBoolean reads a byte from underlying source starting at byte offset off and
// returns it as a bool.
//
func (b *BinaryReader) ReadBoolean(offset int64) (bool, error) {
	if _, err := b.readToBuffer(1, offset); err != nil {
		return false, err
	}
	return b.buffer[0] != 0, nil
}

// ReadUInt16 reads an encoded unsigned 2 byte integer from underlying source starting
// at byte offset off and return it as a uint16.
//
func (b *BinaryReader) ReadUInt16(offset int64) (uint16, error) {
	if _, err := b.readToBuffer(2, offset); err != nil {
		return 0, err
	}
	return b.order.Uint16(b.buffer), nil
}

// ReadInt16 reads an encoded signed 2 byte integer from underlying source starting
// at byte offset off returns it as a int16.
//
func (b *BinaryReader) ReadInt16(off int64) (int16, error) {
	if _, err := b.readToBuffer(2, off); err != nil {
		return 0, err
	}
	return int16(b.order.Uint16(b.buffer)), nil
}

// ReadUInt32 reads an encoded unsigned 4 byte integer from underlying source starting
// at byte offset off returns it as a uint32.
//
func (b *BinaryReader) ReadUInt32(off int64) (uint32, error) {
	if _, err := b.readToBuffer(4, off); err != nil {
		return 0, err
	}
	return b.order.Uint32(b.buffer), nil
}

// ReadInt32 reads an encoded signed 4 byte integer from underlying source starting
// at byte offset off and returns it as a int32.
//
func (b *BinaryReader) ReadInt32(off int64) (int32, error) {
	if _, err := b.readToBuffer(4, off); err != nil {
		return 0, err
	}
	return int32(b.order.Uint32(b.buffer)), nil
}

// ReadUInt64 reads an encoded unsigned 8 byte integer from underlying source starting
// at byte offset off and returns it as a uint64.
//
func (b *BinaryReader) ReadUInt64(off int64) (uint64, error) {
	if _, err := b.readToBuffer(8, off); err != nil {
		return 0, err
	}
	return b.order.Uint64(b.buffer), nil
}

// ReadInt64 reads an encoded signed 4 byte integer from underlying source starting
// at byte offset off and and returns it as a int64.
//
func (b *BinaryReader) ReadInt64(off int64) (int64, error) {
	if _, err := b.readToBuffer(8, off); err != nil {
		return 0, err
	}
	return int64(b.order.Uint64(b.buffer)), nil
}

// ReadUUID reads 16 byte character sequence from underlying source starting
// at byte offset off and returns it as a UUID.
//
func (b *BinaryReader) ReadUUID(off int64) (*common.UUID, error) {
	if _, err := b.readToBuffer(16, off); err != nil {
		return nil, err
	}

	return common.NewUUID(b.buffer)
}

// readToBuffer reads numBytes bytes from the source starting at byte offset off.
// This function uses ReadAt to read the bytes. It returns the number of bytes read
// and the error, if any.
// ReadAt always returns a non-nil error when n < len(numBytes). At end of file, that
// error is io.EOF.
//
func (b *BinaryReader) readToBuffer(numBytes int, off int64) (int, error) {
	if numBytes > bufferSizeInBytes {
		return 0, fmt.Errorf("Expected (0-%d) however found: %d", bufferSizeInBytes, numBytes)
	}

	return b.from.ReadAt(b.buffer[:numBytes], off)
}
