package fat32

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

func getValidFSInfoSector() *FSInformationSector {
	return &FSInformationSector{
		freeDataClustersCount: 20007,
		lastAllocatedCluster:  126,
	}
}

func TestFsInformationSectorFromBytes(t *testing.T) {
	t.Run("mismatched length less than 512", func(t *testing.T) {
		b := make([]byte, 511, 512)
		fsis, err := fsInformationSectorFromBytes(b)
		if err == nil {
			t.Errorf("Did not return expected error")
		}
		if fsis != nil {
			t.Fatalf("Returned FSInformationSector was non-nil")
		}
		expected := fmt.Sprintf("Cannot read FAT32 FS Information Sector from %d bytes", len(b))
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}
	})
	t.Run("mismatched length greater than 512", func(t *testing.T) {
		b := make([]byte, 513, 513)
		fsis, err := fsInformationSectorFromBytes(b)
		if err == nil {
			t.Errorf("Did not return expected error")
		}
		if fsis != nil {
			t.Fatalf("Returned FSInformationSector was non-nil")
		}
		expected := fmt.Sprintf("Cannot read FAT32 FS Information Sector from %d bytes", len(b))
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}

	})
	t.Run("invalid start signature", func(t *testing.T) {
		input, err := ioutil.ReadFile(Fat32File)
		if err != nil {
			t.Fatalf("Error reading test fixture data from %s: %v", Fat32File, err)
		}
		b := input[512:1024]
		// now to pervert one key byte
		b[0] = 0xff
		fsis, err := fsInformationSectorFromBytes(b)
		if err == nil {
			t.Errorf("Did not return expected error")
		}
		if fsis != nil {
			t.Fatalf("Returned FSInformationSector was non-nil")
		}
		expected := fmt.Sprintf("Invalid signature at beginning of FAT 32 Filesystem Information Sector")
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}
	})
	t.Run("invalid middle signature", func(t *testing.T) {
		input, err := ioutil.ReadFile(Fat32File)
		if err != nil {
			t.Fatalf("Error reading test fixture data from %s: %v", Fat32File, err)
		}
		b := input[512:1024]
		// now to pervert one key byte
		b[484] = 0xff
		fsis, err := fsInformationSectorFromBytes(b)
		if err == nil {
			t.Errorf("Did not return expected error")
		}
		if fsis != nil {
			t.Fatalf("Returned FSInformationSector was non-nil")
		}
		expected := fmt.Sprintf("Invalid signature at middle of FAT 32 Filesystem Information Sector")
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}
	})
	t.Run("invalid end signature", func(t *testing.T) {
		input, err := ioutil.ReadFile(Fat32File)
		if err != nil {
			t.Fatalf("Error reading test fixture data from %s: %v", Fat32File, err)
		}
		b := input[512:1024]
		// now to pervert one key byte
		b[510] = 0xff
		fsis, err := fsInformationSectorFromBytes(b)
		if err == nil {
			t.Errorf("Did not return expected error")
		}
		if fsis != nil {
			t.Fatalf("Returned FSInformationSector was non-nil")
		}
		expected := fmt.Sprintf("Invalid signature at end of FAT 32 Filesystem Information Sector")
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}
	})
	t.Run("valid FS Information Sector", func(t *testing.T) {
		input, err := ioutil.ReadFile(Fat32File)
		if err != nil {
			t.Fatalf("Error reading test fixture data from %s: %v", Fat32File, err)
		}
		b := input[512:1024]
		fsis, err := fsInformationSectorFromBytes(b)
		if err != nil {
			t.Errorf("Return unexpected error: %v", err)
		}
		if fsis == nil {
			t.Fatalf("Returned FSInformationSector was nil unexpectedly")
		}
		valid := getValidFSInfoSector()
		if *valid != *fsis {
			t.Log(fsis)
			t.Log(valid)
			t.Fatalf("Mismatched FSInformationSector")
		}
	})
}

func TestInformationSectorToBytes(t *testing.T) {
	t.Run("valid FSInformationSector", func(t *testing.T) {
		fsis := getValidFSInfoSector()
		b, err := fsis.toBytes()
		if err != nil {
			t.Errorf("Error was not nil, instead %v", err)
		}
		if b == nil {
			t.Fatal("b was nil unexpectedly")
		}
		valid, err := ioutil.ReadFile(Fat32File)
		if err != nil {
			t.Fatalf("Error reading test fixture data from %s: %v", Fat32File, err)
		}
		validBytes := valid[512:1024]
		if bytes.Compare(validBytes, b) != 0 {
			t.Error("Mismatched bytes")
		}
	})
}
