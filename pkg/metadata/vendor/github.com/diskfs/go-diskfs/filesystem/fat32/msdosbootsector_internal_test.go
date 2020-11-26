package fat32

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

func getValidMsDosBootSector() *msDosBootSector {
	return &msDosBootSector{
		biosParameterBlock: getValidDos71EBPB(),
		oemName:            "mkfs.fat",
		jumpInstruction:    [3]byte{0xeb, 0x58, 0x90},
		bootCode: []byte{0x0e, 0x1f, 0xbe, 0x77, 0x7c, 0xac,
			0x22, 0xc0, 0x74, 0x0b, 0x56, 0xb4, 0x0e, 0xbb, 0x07, 0x00, 0xcd, 0x10, 0x5e, 0xeb, 0xf0, 0x32,
			0xe4, 0xcd, 0x16, 0xcd, 0x19, 0xeb, 0xfe, 0x54, 0x68, 0x69, 0x73, 0x20, 0x69, 0x73, 0x20, 0x6e,
			0x6f, 0x74, 0x20, 0x61, 0x20, 0x62, 0x6f, 0x6f, 0x74, 0x61, 0x62, 0x6c, 0x65, 0x20, 0x64, 0x69,
			0x73, 0x6b, 0x2e, 0x20, 0x20, 0x50, 0x6c, 0x65, 0x61, 0x73, 0x65, 0x20, 0x69, 0x6e, 0x73, 0x65,
			0x72, 0x74, 0x20, 0x61, 0x20, 0x62, 0x6f, 0x6f, 0x74, 0x61, 0x62, 0x6c, 0x65, 0x20, 0x66, 0x6c,
			0x6f, 0x70, 0x70, 0x79, 0x20, 0x61, 0x6e, 0x64, 0x0d, 0x0a, 0x70, 0x72, 0x65, 0x73, 0x73, 0x20,
			0x61, 0x6e, 0x79, 0x20, 0x6b, 0x65, 0x79, 0x20, 0x74, 0x6f, 0x20, 0x74, 0x72, 0x79, 0x20, 0x61,
			0x67, 0x61, 0x69, 0x6e, 0x20, 0x2e, 0x2e, 0x2e, 0x20, 0x0d, 0x0a, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		},
	}
}

func TestMsDosBootSectorFromBytes(t *testing.T) {
	t.Run("mismatched length less than 512", func(t *testing.T) {
		b := make([]byte, 511, 512)
		bs, err := msDosBootSectorFromBytes(b)
		if err == nil {
			t.Errorf("Did not return expected error")
		}
		if bs != nil {
			t.Fatalf("Returned MsDosBootSector was non-nil")
		}
		expected := fmt.Sprintf("Cannot parse MS-DOS Boot Sector from %d bytes", len(b))
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}
	})
	t.Run("mismatched length greater than 512", func(t *testing.T) {
		b := make([]byte, 513, 513)
		bs, err := msDosBootSectorFromBytes(b)
		if err == nil {
			t.Errorf("Did not return expected error")
		}
		if bs != nil {
			t.Fatalf("Returned MsDosBootSector was non-nil")
		}
		expected := fmt.Sprintf("Cannot parse MS-DOS Boot Sector from %d bytes", len(b))
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}

	})
	t.Run("invalid Dos71EBPB", func(t *testing.T) {
		input, err := ioutil.ReadFile(Fat32File)
		if err != nil {
			t.Fatalf("Error reading test fixture data from %s: %v", Fat32File, err)
		}
		b := input[0:512]
		// now to pervert one key byte
		ebpbBytes := b[11:90]
		ebpbBytes[31] = 0xff
		bs, err := msDosBootSectorFromBytes(b)
		if err == nil {
			t.Errorf("Did not return expected error")
		}
		if bs != nil {
			t.Fatalf("Returned MsDosBootSector was non-nil")
		}
		expected := fmt.Sprintf("Could not read FAT32 BIOS Parameter Block from boot sector")
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}
	})
	t.Run("invalid signature", func(t *testing.T) {
		input, err := ioutil.ReadFile(Fat32File)
		if err != nil {
			t.Fatalf("Error reading test fixture data from %s: %v", Fat32File, err)
		}
		b := input[0:512]
		b[510] = 0x5e
		bs, err := msDosBootSectorFromBytes(b)
		if err == nil {
			t.Errorf("Did not return expected error")
		}
		if bs != nil {
			t.Fatalf("Returned MsDosBootSector was non-nil")
		}
		expected := fmt.Sprintf("Invalid signature in last 2 bytes of boot sector")
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}
	})
	t.Run("valid MsDosBootSector", func(t *testing.T) {
		input, err := ioutil.ReadFile(Fat32File)
		if err != nil {
			t.Fatalf("Error reading test fixture data from %s: %v", Fat32File, err)
		}
		b := input[0:512]
		bs, err := msDosBootSectorFromBytes(b)
		if err != nil {
			t.Errorf("Return unexpected error: %v", err)
		}
		if bs == nil {
			t.Fatalf("Returned MsDosBootSector was nil unexpectedly")
		}
		valid := getValidMsDosBootSector()
		if !bs.equal(valid) {
			t.Log(bs)
			t.Log(valid)
			t.Fatalf("Mismatched MS-DOS Boot Sector")
		}
	})
}

func TestMsDosBootSectorToBytes(t *testing.T) {
	t.Run("short OEM Name", func(t *testing.T) {
		name := "abc"
		bs := getValidMsDosBootSector()
		bs.oemName = name
		b, err := bs.toBytes()
		if err != nil {
			t.Errorf("Error was not nil, instead %v", err)
		}
		if b == nil {
			t.Fatal("b was nil unexpectedly")
		}
		// it should have passed it
		calculatedName := b[3:11]
		expectedName := []byte{97, 98, 99, 0x20, 0x20, 0x20, 0x20, 0x20}
		if bytes.Compare(calculatedName, expectedName) != 0 {
			t.Log(calculatedName)
			t.Log(expectedName)
			t.Fatal("did not fill short OEM name properly")
		}
	})
	t.Run("long OEM Name", func(t *testing.T) {
		bs := getValidMsDosBootSector()
		bs.oemName = "abcdefghijklmnop"
		b, err := bs.toBytes()
		if err == nil {
			t.Error("Error was nil unexpectedly")
		}
		if b != nil {
			t.Fatal("b was not nil")
		}
		expected := fmt.Sprintf("Cannot use OEM Name > 8 bytes")
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}
	})
	t.Run("non-ascii OEM Name", func(t *testing.T) {
		bs := getValidMsDosBootSector()
		bs.oemName = "\u0061\u6785"
		b, err := bs.toBytes()
		if err == nil {
			t.Error("Error was nil unexpectedly")
		}
		if b != nil {
			t.Fatal("b was not nil")
		}
		expected := fmt.Sprintf("Invalid OEM Name: non-ascii characters")
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}
	})
	t.Run("short boot code", func(t *testing.T) {
		bs := getValidMsDosBootSector()
		bs.bootCode = []byte{0x45, 0x56}
		b, err := bs.toBytes()
		if err != nil {
			t.Errorf("Error was not nil, instead %v", err)
		}
		if b == nil {
			t.Fatal("b was nil unexpectedly")
		}
		// it should have passed it
		calculatedBootCode := b[90:510]
		expectedBootCode := make([]byte, 420, 420)
		copy(expectedBootCode, bs.bootCode)
		if bytes.Compare(calculatedBootCode, expectedBootCode) != 0 {
			t.Log(calculatedBootCode)
			t.Log(expectedBootCode)
			t.Fatal("did not fill boot code properly")
		}
	})
	t.Run("long boot code", func(t *testing.T) {
		bs := getValidMsDosBootSector()
		bc := make([]byte, 600, 600)
		rand.Read(bc)
		bs.bootCode = bc
		b, err := bs.toBytes()
		if err == nil {
			t.Error("Error was nil unexpectedly")
		}
		if b != nil {
			t.Fatal("b was not nil unexpectedly")
		}
		expected := fmt.Sprintf("boot code too long")
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}
	})
	t.Run("valid Boot Sector", func(t *testing.T) {
		bs := getValidMsDosBootSector()
		b, err := bs.toBytes()
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
		validBytes := valid[:512]
		if bytes.Compare(validBytes, b) != 0 {
			t.Error("Mismatched bytes")
		}
	})
}
