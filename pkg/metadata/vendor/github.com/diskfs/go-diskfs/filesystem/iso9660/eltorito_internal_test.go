package iso9660

import (
	"bytes"
	"testing"

	"github.com/diskfs/go-diskfs/partition/mbr"
	"github.com/diskfs/go-diskfs/util"
)

func TestElToritoGenerateCatalog(t *testing.T) {
	et := &ElTorito{
		BootCatalog:     "/boot.cat",
		HideBootCatalog: false,
		Platform:        EFI,
		Entries: []*ElToritoEntry{
			{Platform: BIOS, Emulation: HardDiskEmulation, BootFile: "/abc.img", HideBootFile: false, LoadSegment: 23, SystemType: mbr.Linux, size: 10, location: 100},
			{Platform: BIOS, Emulation: NoEmulation, BootFile: "/def.img", HideBootFile: false, LoadSegment: 0, SystemType: mbr.Fat32LBA, size: 20, location: 200},
			{Platform: EFI, Emulation: NoEmulation, BootFile: "/qrs.img", HideBootFile: false, LoadSegment: 0, SystemType: mbr.Fat16, size: 30, location: 300},
		},
	}
	// the catalog should look like
	// - validation entry
	// - initial/default entry
	// - header+entry for each subsequent
	//
	// we are NOT testing the conversions here as we do them elsewhere

	e := make([]byte, 0)
	e = append(e, et.validationEntry()...)
	e = append(e, et.Entries[0].entryBytes()...)
	e = append(e, et.Entries[1].headerBytes(false, 1)...)
	e = append(e, et.Entries[1].entryBytes()...)
	e = append(e, et.Entries[2].headerBytes(true, 1)...)
	e = append(e, et.Entries[2].entryBytes()...)

	b, err := et.generateCatalog()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if bytes.Compare(b, e) != 0 {
		t.Errorf("Mismatched bytes, actual then expected\n% x\n% x\n", b, e)
	}
}

func TestElToritoValidationEntry(t *testing.T) {
	et := &ElTorito{
		BootCatalog:     "/boot.cat",
		HideBootCatalog: false,
		Platform:        EFI,
	}
	b := et.validationEntry()
	e := make([]byte, 0x20)
	e[0] = 0x1
	e[1] = 0xef
	copy(e[4:0x1c], util.AppNameVersion)
	e[0x1e] = 0x55
	e[0x1f] = 0xaa

	// add the checksum - we calculated this manually
	e[0x1c] = 0x3c
	e[0x1d] = 0xd5
	if bytes.Compare(b, e) != 0 {
		t.Errorf("Mismatched bytes, actual then expected\n% x\n% x\n", b, e)
	}
}

func TestElToritoHeaderBytes(t *testing.T) {
	var (
		boot = "/abc.img"
	)
	e := &ElToritoEntry{
		Platform:     BIOS,
		Emulation:    HardDiskEmulation,
		BootFile:     boot,
		HideBootFile: false,
		LoadSegment:  23,
		SystemType:   mbr.Linux,
	}
	tests := []struct {
		last     bool
		entries  uint16
		expected []byte
	}{
		{true, 1, []byte{0x91, byte(BIOS), 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}},
		{false, 1, []byte{0x90, byte(BIOS), 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}},
		{true, 25, []byte{0x91, byte(BIOS), 0x19, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}},
		{false, 36, []byte{0x90, byte(BIOS), 0x24, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0}},
	}
	for _, tt := range tests {
		b := e.headerBytes(tt.last, tt.entries)
		if bytes.Compare(b, tt.expected) != 0 {
			t.Errorf("last (%v), entries (%d): mismatched result, actual then expected\n% x\n% x\n", tt.last, tt.entries, b, tt.expected)
		}
	}
}

func TestElToritoEntryBytes(t *testing.T) {
	var (
		boot = "/abc.img"
	)
	e := &ElToritoEntry{
		Platform:     BIOS,
		Emulation:    HardDiskEmulation,
		BootFile:     boot,
		HideBootFile: false,
		LoadSegment:  23,
		SystemType:   mbr.Linux,
		size:         2450,
		location:     193,
	}
	b := e.entryBytes()
	expected := make([]byte, 0x20)
	copy(expected, []byte{0x88, byte(HardDiskEmulation), 0x17, 0x0, byte(mbr.Linux), 0x0, 0x5, 0x0, 0xc1, 0x00})
	if bytes.Compare(b, expected) != 0 {
		t.Errorf("Mismatched bytes, actual then expected\n% x\n% x\n", b, expected)
	}
}
