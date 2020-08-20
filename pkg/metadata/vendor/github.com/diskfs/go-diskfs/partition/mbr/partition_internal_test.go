package mbr

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/diskfs/go-diskfs/testhelper"
)

const (
	mbrPartitionFile = "./testdata/mbr_partition.dat"
	partitionStart   = 2048
	partitionSize    = 20480
)

func TestFromBytes(t *testing.T) {
	t.Run("Short byte slice", func(t *testing.T) {
		b := make([]byte, partitionEntrySize-1, partitionEntrySize-1)
		rand.Read(b)
		partition, err := partitionFromBytes(b, logicalSectorSize, physicalSectorSize)
		if partition != nil {
			t.Error("should return nil partition")
		}
		if err == nil {
			t.Error("should not return nil error")
		}
		expected := fmt.Sprintf("Data for partition was %d bytes instead of expected %d", len(b), partitionEntrySize)
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}
	})
	t.Run("Long byte slice", func(t *testing.T) {
		b := make([]byte, partitionEntrySize+1, partitionEntrySize+1)
		rand.Read(b)
		partition, err := partitionFromBytes(b, logicalSectorSize, physicalSectorSize)
		if partition != nil {
			t.Error("should return nil partition")
		}
		if err == nil {
			t.Error("should not return nil error")
		}
		expected := fmt.Sprintf("Data for partition was %d bytes instead of expected %d", len(b), partitionEntrySize)
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}
	})
	t.Run("Invalid partition bootable code", func(t *testing.T) {
		b := make([]byte, partitionEntrySize, partitionEntrySize)
		rand.Read(b)
		b[0] = 0x67
		partition, err := partitionFromBytes(b, logicalSectorSize, physicalSectorSize)
		if partition != nil {
			t.Error("should return nil partition")
		}
		if err == nil {
			t.Error("should not return nil error")
		}
		expected := fmt.Sprintf("Invalid partition")
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}
	})
	t.Run("Valid partition", func(t *testing.T) {
		b, err := ioutil.ReadFile(mbrPartitionFile)
		if err != nil {
			t.Fatalf("Unable to read test fixture file %s: %v", mbrPartitionFile, err)
		}
		partition, err := partitionFromBytes(b, logicalSectorSize, physicalSectorSize)
		if partition == nil {
			t.Error("should not return nil partition")
		}
		if err != nil {
			t.Errorf("returned non-nil error: %v", err)
		}
		// check out data
		expected := Partition{
			Bootable:      false,
			StartHead:     0x20,
			StartSector:   0x21,
			StartCylinder: 0x00,
			Type:          Linux,
			EndHead:       0x31,
			EndSector:     0x18,
			EndCylinder:   0x00,
			Start:         partitionStart,
			Size:          partitionSize,
		}
		if partition == nil && &expected != nil || !partition.Equal(&expected) {
			t.Log(b)
			t.Errorf("actual partition was %v instead of expected %v", *partition, expected)
		}

	})
}

func TestToBytes(t *testing.T) {
	t.Run("Valid partition", func(t *testing.T) {
		partition := Partition{
			Bootable:      false,
			StartHead:     0,
			StartSector:   2,
			StartCylinder: 0,
			Type:          Linux,
			EndHead:       0,
			EndSector:     2,
			EndCylinder:   0,
			Start:         partitionStart,
			Size:          partitionSize,
		}
		b, err := partition.toBytes()
		if b == nil {
			t.Error("should not return nil bytes")
		}
		if err != nil {
			t.Error("should return nil error")
		}
		expected, err := ioutil.ReadFile(mbrPartitionFile)
		if err != nil {
			t.Fatalf("Unable to read test fixture file %s: %v", mbrPartitionFile, err)
		}
		if !PartitionEqualBytes(expected, b) {
			t.Errorf("returned byte %v instead of expected %v", b, expected)
		}
	})
}

func TestReadContents(t *testing.T) {
	t.Run("error reading file", func(t *testing.T) {
		partition := Partition{
			Bootable:      false,
			StartHead:     0,
			StartSector:   2,
			StartCylinder: 0,
			Type:          Linux,
			EndHead:       0,
			EndSector:     2,
			EndCylinder:   0,
			Start:         partitionStart,
			Size:          partitionSize,
		}
		var b bytes.Buffer
		writer := bufio.NewWriter(&b)
		expected := "Error reading from file"
		f := &testhelper.FileImpl{
			Reader: func(b []byte, offset int64) (int, error) {
				return 0, fmt.Errorf(expected)
			},
		}
		read, err := partition.ReadContents(f, writer)
		if read != 0 {
			t.Errorf("Returned %d bytes read instead of 0", read)
		}
		if err == nil {
			t.Errorf("Returned nil error instead of actual errors")
		}
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}
	})
	t.Run("successful read", func(t *testing.T) {
		partition := Partition{
			Bootable:      false,
			StartHead:     0,
			StartSector:   2,
			StartCylinder: 0,
			Type:          Linux,
			EndHead:       0,
			EndSector:     2,
			EndCylinder:   0,
			Start:         partitionStart,
			Size:          partitionSize,
		}
		var b bytes.Buffer
		writer := bufio.NewWriter(&b)
		size := 100
		b2 := make([]byte, size, size)
		rand.Read(b2)
		f := &testhelper.FileImpl{
			Reader: func(b []byte, offset int64) (int, error) {
				copy(b, b2)
				return size, io.EOF
			},
		}
		read, err := partition.ReadContents(f, writer)
		if read != int64(size) {
			t.Errorf("Returned %d bytes read instead of %d", read, size)
		}
		if err != nil {
			t.Errorf("Returned error instead of expected nil")
		}
		writer.Flush()
		if bytes.Compare(b.Bytes(), b2) != 0 {
			t.Errorf("Mismatched bytes data")
			t.Log(b.Bytes())
			t.Log(b2)
		}
	})
}

func TestWriteContents(t *testing.T) {
	t.Run("mismatched size", func(t *testing.T) {
		partition := Partition{
			Bootable:      false,
			StartHead:     0,
			StartSector:   2,
			StartCylinder: 0,
			Type:          Linux,
			EndHead:       0,
			EndSector:     2,
			EndCylinder:   0,
			Start:         partitionStart,
			Size:          partitionSize,
		}
		var b bytes.Buffer
		reader := bufio.NewReader(&b)
		expected := "Write 0 bytes to partition "
		f := &testhelper.FileImpl{}
		written, err := partition.WriteContents(f, reader)
		if written != 0 {
			t.Errorf("Returned %d bytes written instead of 0", written)
		}
		if err == nil {
			t.Errorf("Returned nil error instead of actual errors")
		}
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}
	})
	t.Run("error writing file", func(t *testing.T) {
		size := 512000
		partition := Partition{
			Bootable:      false,
			StartHead:     0,
			StartSector:   2,
			StartCylinder: 0,
			Type:          Linux,
			EndHead:       0,
			EndSector:     2,
			EndCylinder:   0,
			Start:         partitionStart,
			Size:          partitionSize,
		}
		b := make([]byte, size, size)
		rand.Read(b)
		reader := bytes.NewReader(b)
		expected := "Error writing to file"
		f := &testhelper.FileImpl{
			Writer: func(b []byte, offset int64) (int, error) {
				return 0, fmt.Errorf(expected)
			},
		}
		written, err := partition.WriteContents(f, reader)
		if written != 0 {
			t.Errorf("Returned %d bytes written instead of 0", written)
		}
		if err == nil {
			t.Errorf("Returned nil error instead of actual errors")
			return
		}
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}
	})
	t.Run("too large for partition", func(t *testing.T) {
		partition := Partition{
			Bootable:      false,
			StartHead:     0,
			StartSector:   2,
			StartCylinder: 0,
			Type:          Linux,
			EndHead:       0,
			EndSector:     2,
			EndCylinder:   0,
			Start:         partitionStart,
			Size:          1,
		}
		// make a byte array that is too big
		b := make([]byte, 2*512, 2*512)
		rand.Read(b)
		reader := bytes.NewReader(b)
		expected := "Requested to write at least"
		f := &testhelper.FileImpl{
			Writer: func(b []byte, offset int64) (int, error) {
				return len(b), nil
			},
		}
		// We have a size of 1 sector, or 512 bytes, but are trying to write 2*512.
		// It should write the first and fail on the second so we expect an error,
		// along with 512 bytes successfully written
		written, err := partition.WriteContents(f, reader)
		if written != 512 {
			t.Errorf("Returned %d bytes written instead of 512", written)
		}
		if err == nil {
			t.Errorf("Returned nil error instead of actual errors")
			return
		}
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}
	})

	t.Run("successful write", func(t *testing.T) {
		size := 512000
		sectorSize := size / 512
		partition := Partition{
			Bootable:      false,
			StartHead:     0,
			StartSector:   2,
			StartCylinder: 0,
			Type:          Linux,
			EndHead:       0,
			EndSector:     2,
			EndCylinder:   0,
			Start:         partitionStart,
			Size:          uint32(sectorSize),
		}
		b := make([]byte, size, size)
		rand.Read(b)
		b2 := make([]byte, 0, size)
		reader := bytes.NewReader(b)
		f := &testhelper.FileImpl{
			Writer: func(b []byte, offset int64) (int, error) {
				b2 = append(b2, b...)
				return len(b), nil
			},
		}
		written, err := partition.WriteContents(f, reader)
		if written != uint64(size) {
			t.Errorf("Returned %d bytes written instead of %d", written, size)
		}
		if err != nil {
			t.Errorf("Returned error instead of nil: %v", err)
			return
		}
		if bytes.Compare(b2, b) != 0 {
			t.Errorf("Bytes mismatch")
			t.Log(b)
			t.Log(b2)
		}
	})
}
