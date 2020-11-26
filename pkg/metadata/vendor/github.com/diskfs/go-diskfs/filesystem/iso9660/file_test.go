package iso9660_test

import (
	"io"
	"testing"

	"github.com/diskfs/go-diskfs/filesystem/iso9660"
)

func TestFileRead(t *testing.T) {
	// pretty simple: never should be able to write as it is a read-only filesystem
	// we use
	f, content := iso9660.GetTestFile(t)

	b := make([]byte, 20, 20)
	read, err := f.Read(b)
	if read != 0 && err != io.EOF {
		t.Errorf("received unexpected error when reading: %v", err)
	}
	if read != len(content) {
		t.Errorf("read %d bytes instead of expected %d", read, len(content))
	}
	bString := string(b[:read])
	if bString != content {
		t.Errorf("Mismatched content:\nActual: '%s'\nExpected: '%s'", bString, content)
	}
}

func TestFileWrite(t *testing.T) {
	// pretty simple: never should be able to write as it is a read-only filesystem
	f := &iso9660.File{}
	b := make([]byte, 8, 8)
	written, err := f.Write(b)
	if err == nil {
		t.Errorf("received no error when should have been prevented from writing")
	}
	if written != 0 {
		t.Errorf("wrote %d bytes instead of expected %d", written, 0)
	}
}
