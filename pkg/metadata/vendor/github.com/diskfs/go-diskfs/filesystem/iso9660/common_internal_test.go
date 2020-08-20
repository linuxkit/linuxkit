package iso9660

import (
	"os"
	"testing"
	"time"
)

const (
	//ISO9660File = "./testdata/file.iso"
	ISO9660File   = "./testdata/9660.iso"
	RockRidgeFile = "./testdata/rockridge.iso"
	ISO9660Size   = 11018240
)

func GetTestFile(t *testing.T) (*File, string) {
	// we use the entry for FILENA01.;1 , which should have the content "filename_01" (without the quotes)
	// see ./testdata/README.md
	//
	// entry:
	// {recordSize:0x7a, extAttrSize:0x0, location:0x1422, size:0xb, creation:time.Time{wall:0x0, ext:0, loc:(*time.Location)(nil)}, isHidden:false, isSubdirectory:false, isAssociated:false, hasExtendedAttrs:false, hasOwnerGroupPermissions:false, hasMoreEntries:false, volumeSequence:0x0, filename:"FILENA01.;1"},
	// FileSystem implements the FileSystem interface
	file, err := os.Open(ISO9660File)
	if err != nil {
		t.Errorf("Could not read ISO9660 test file %s: %v", ISO9660File, err)
	}
	fs := &FileSystem{
		workspace: "",
		size:      ISO9660Size,
		start:     0,
		file:      file,
		blocksize: 2048,
	}
	de := &directoryEntry{
		extAttrSize: 0,
		location:    0x1473,
		size:        0x7,
		creation:    time.Now(),
		filesystem:  fs,
		filename:    "README.MD;1",
	}
	return &File{
		directoryEntry: de,
		isReadWrite:    false,
		isAppend:       false,
		offset:         0,
	}, "README\n"
}
