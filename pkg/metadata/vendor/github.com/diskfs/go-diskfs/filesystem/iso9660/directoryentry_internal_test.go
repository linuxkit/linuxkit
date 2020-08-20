package iso9660

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/diskfs/go-diskfs/testhelper"
)

var (
	timeBytesTests = []struct {
		b   []byte
		rfc string
	}{
		// see reference at https://wiki.osdev.org/ISO_9660#Directories
		{[]byte{80, 1, 2, 14, 35, 36, 0}, "1980-01-02T14:35:36+00:00"},
		{[]byte{95, 11, 25, 0, 16, 7, 8}, "1995-11-25T00:16:07+02:00"},
		{[]byte{101, 6, 30, 12, 0, 0, 0xe6}, "2001-06-30T12:00:00-06:30"},
	}
)

func compareDirectoryEntries(a, b *directoryEntry, compareDates, compareExtensions bool) bool {
	now := time.Now()
	// copy values so we do not mess up the originals
	c := &directoryEntry{}
	d := &directoryEntry{}
	*c = *a
	*d = *b

	if !compareDates {
		// unify fields we let be equal
		c.creation = now
		d.creation = now
	}

	cExt := c.extensions
	dExt := d.extensions
	c.extensions = nil
	d.extensions = nil

	shallowMatch := reflect.DeepEqual(*c, *d)
	extMatch := true
	switch {
	case !compareExtensions:
		extMatch = true
	case len(cExt) != len(dExt):
		extMatch = false
	default:
		// compare them
		for i, e := range cExt {
			if e.Signature() != dExt[i].Signature() || e.Length() != dExt[i].Length() || e.Version() != dExt[i].Version() || bytes.Compare(e.Data(), dExt[i].Data()) != 0 {
				extMatch = false
				break
			}
		}
	}
	return shallowMatch && extMatch
}
func directoryEntryBytesNullDate(a []byte) []byte {
	now := make([]byte, 7, 7)
	a1 := make([]byte, len(a))
	copy(a1[18:18+7], now)
	return a1
}

// get9660DirectoryEntries get a list of valid directory entries for path /
// returns:
// slice of entries, slice of byte slices for each, entire bytes for all, map of location to byte slice
func get9660DirectoryEntries(f *FileSystem) ([]*directoryEntry, [][][]byte, []byte, map[int][]byte, error) {
	blocksize := 2048
	rootSector := 18
	ceSector := 19
	// read correct bytes off of disk
	input, err := ioutil.ReadFile(ISO9660File)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("Error reading data from iso9660 test fixture %s: %v", ISO9660File, err)
	}

	// start of root directory in file.iso - sector 18
	// sector 0-15 - system area
	// sector 16 - Primary Volume Descriptor
	// sector 17 - Volume Descriptor Set Terimnator
	// sector 18 - / (root) directory
	// sector 19 - Continuation Area for root directory first entry
	// sector 20 - /abc directory
	// sector 21 - /bar directory
	// sector 22 - /foo directory
	// sector 23 - /foo directory
	// sector 24 - /foo directory
	// sector 25 - /foo directory
	// sector 26 - /foo directory
	// sector 27 - L path table
	// sector 28 - M path table
	// sector 33-2592 - /ABC/LARGEFILE
	// sector 2593-5152 - /BAR/LARGEFILE
	// sector 5153 - /FOO/FILENA01
	//  ..
	// sector 5228 - /FOO/FILENA75
	// sector 5229 - /README.MD
	startRoot := rootSector * blocksize // start of root directory in file.iso

	// one block, since we know it is just one block
	allBytes := input[startRoot : startRoot+blocksize]

	startCe := ceSector * blocksize               // start of ce area in file.iso
	ceBytes := input[startCe : startCe+blocksize] // CE block
	// cut the CE block down to just the CE bytes
	ceBytes = ceBytes[:237]

	t1 := time.Now()
	entries := []*directoryEntry{
		&directoryEntry{
			extAttrSize:              0,
			location:                 0x12,
			size:                     0x800,
			creation:                 t1,
			isHidden:                 false,
			isSubdirectory:           true,
			isAssociated:             false,
			hasExtendedAttrs:         false,
			hasOwnerGroupPermissions: false,
			hasMoreEntries:           false,
			volumeSequence:           1,
			filename:                 "",
			isSelf:                   true,
			filesystem:               f,
		},
		&directoryEntry{
			extAttrSize:              0,
			location:                 0x12,
			size:                     0x800,
			creation:                 t1,
			isHidden:                 false,
			isSubdirectory:           true,
			isAssociated:             false,
			hasExtendedAttrs:         false,
			hasOwnerGroupPermissions: false,
			hasMoreEntries:           false,
			volumeSequence:           1,
			filename:                 "",
			isParent:                 true,
			filesystem:               f,
		},
		&directoryEntry{
			extAttrSize:              0,
			location:                 0x13,
			size:                     0x800,
			creation:                 t1,
			isHidden:                 false,
			isSubdirectory:           true,
			isAssociated:             false,
			hasExtendedAttrs:         false,
			hasOwnerGroupPermissions: false,
			hasMoreEntries:           false,
			volumeSequence:           1,
			filename:                 "ABC",
			filesystem:               f,
		},
		&directoryEntry{
			extAttrSize:              0,
			location:                 0x14,
			size:                     0x800,
			creation:                 t1,
			isHidden:                 false,
			isSubdirectory:           true,
			isAssociated:             false,
			hasExtendedAttrs:         false,
			hasOwnerGroupPermissions: false,
			hasMoreEntries:           false,
			volumeSequence:           1,
			filename:                 "BAR",
			filesystem:               f,
		},
		&directoryEntry{
			extAttrSize:              0,
			location:                 0x15,
			size:                     0x800,
			creation:                 t1,
			isHidden:                 false,
			isSubdirectory:           true,
			isAssociated:             false,
			hasExtendedAttrs:         false,
			hasOwnerGroupPermissions: false,
			hasMoreEntries:           false,
			volumeSequence:           1,
			filename:                 "DEEP",
			filesystem:               f,
		},
		&directoryEntry{
			extAttrSize:              0,
			location:                 0x21,
			size:                     0x1000,
			creation:                 t1,
			isHidden:                 false,
			isSubdirectory:           true,
			isAssociated:             false,
			hasExtendedAttrs:         false,
			hasOwnerGroupPermissions: false,
			hasMoreEntries:           false,
			volumeSequence:           1,
			filename:                 "FOO",
			filesystem:               f,
		},
		&directoryEntry{
			extAttrSize:              0,
			location:                 0x1473,
			size:                     0x7,
			creation:                 t1,
			isHidden:                 false,
			isSubdirectory:           false,
			isAssociated:             false,
			hasExtendedAttrs:         false,
			hasOwnerGroupPermissions: false,
			hasMoreEntries:           false,
			volumeSequence:           1,
			filename:                 "README.MD;1",
			filesystem:               f,
		},
	}

	// b is a slice of byte slices; each one represents the raw directory entry bytes for a directoryEntry
	//  parallels the directory Entries above
	// each entry is a slice of slices: the first is the raw data for the entry
	//  the second and any subsequent are continuation areas
	b := make([][][]byte, 0, 8)
	read := 0
	for range entries {
		recordSize := int(allBytes[read])
		// do we have a 0 point? if so, move ahead until we pass it at the end of the block
		if recordSize == 0x00 {
			read += (blocksize - read%blocksize)
		}
		b2 := make([][]byte, 0)
		b2 = append(b2, allBytes[read:read+recordSize])
		b = append(b, b2)
		read += recordSize
	}
	// the first one has a continuation entry
	b[0] = append(b[0], ceBytes)

	byteMap := map[int][]byte{
		rootSector: allBytes, ceSector: ceBytes,
	}
	return entries, b, allBytes, byteMap, nil
}

// getRockRidgeDirectoryEntries get a list of valid directory entries for path /
// returns:
// slice of entries, slice of byte slices for each, entire bytes for all, map of location to byte slice
func getRockRidgeDirectoryEntries(f *FileSystem, includeRelocated bool) ([]*directoryEntry, [][][]byte, []byte, map[int][]byte, error) {
	blocksize := 2048
	rootSector := 18
	ceSector := 19
	// read correct bytes off of disk
	input, err := ioutil.ReadFile(RockRidgeFile)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("Error reading data from iso9660 test fixture %s: %v", ISO9660File, err)
	}

	// start of root directory in file.iso - sector 18
	// sector 0-15 - system area
	// sector 16 - Primary Volume Descriptor
	// sector 17 - Volume Descriptor Set Terimnator
	// sector 18 - / (root) directory
	// sector 19 - Continuation Area for root directory first entry
	// sector 20 - /abc directory
	// sector 21 - /bar directory
	// sector 22 - /foo directory
	// sector 23 - /foo directory
	// sector 24 - /foo directory
	// sector 25 - /foo directory
	// sector 26 - /foo directory
	// sector 27 - L path table
	// sector 28 - M path table
	// sector 33-2592 - /ABC/LARGEFILE
	// sector 2593-5152 - /BAR/LARGEFILE
	// sector 5153 - /FOO/FILENA01
	//  ..
	// sector 5228 - /FOO/FILENA75
	// sector 5229 - /README.MD
	startRoot := rootSector * blocksize // start of root directory in file.iso

	// one block, since we know it is just one block
	allBytes := input[startRoot : startRoot+blocksize]

	startCe := ceSector * blocksize               // start of ce area in file.iso
	ceBytes := input[startCe : startCe+blocksize] // CE block
	// cut the CE block down to just the CE bytes
	ceBytes = ceBytes[:237]

	t1 := time.Now()
	entries := []*directoryEntry{
		&directoryEntry{
			extAttrSize:              0,
			location:                 0x12,
			size:                     0x800,
			creation:                 t1,
			isHidden:                 false,
			isSubdirectory:           true,
			isAssociated:             false,
			hasExtendedAttrs:         false,
			hasOwnerGroupPermissions: false,
			hasMoreEntries:           false,
			volumeSequence:           1,
			filename:                 "",
			isSelf:                   true,
			filesystem:               f,
			extensions: []directoryEntrySystemUseExtension{
				directoryEntrySystemUseExtensionSharingProtocolIndicator{0},
				rockRidgePosixAttributes{mode: 0755 | os.ModeDir, linkCount: 1, uid: 0, gid: 0, length: 36},
				rockRidgeTimestamps{longForm: false, stamps: []rockRidgeTimestamp{
					{timestampType: rockRidgeTimestampModify, time: bytesToTime([]byte{0x76, 0x9, 0x1b, 0xb, 0x2f, 0x15, 0x0})},
					{timestampType: rockRidgeTimestampAccess, time: bytesToTime([]byte{0x76, 0x9, 0x1b, 0xb, 0x2f, 0x29, 0x0})},
					{timestampType: rockRidgeTimestampAttribute, time: bytesToTime([]byte{0x76, 0x9, 0x1b, 0xb, 0x2f, 0x15, 0x0})},
				},
				},
				directoryEntrySystemUseExtensionReference{extensionVersion: 1, id: "RRIP_1991A", descriptor: "THE ROCK RIDGE INTERCHANGE PROTOCOL PROVIDES SUPPORT FOR POSIX FILE SYSTEM SEMANTICS", source: "PLEASE CONTACT DISC PUBLISHER FOR SPECIFICATION SOURCE.  SEE PUBLISHER IDENTIFIER IN PRIMARY VOLUME DESCRIPTOR FOR CONTACT INFORMATION."},
			},
		},
		&directoryEntry{
			extAttrSize:              0,
			location:                 0x12,
			size:                     0x800,
			creation:                 t1,
			isHidden:                 false,
			isSubdirectory:           true,
			isAssociated:             false,
			hasExtendedAttrs:         false,
			hasOwnerGroupPermissions: false,
			hasMoreEntries:           false,
			volumeSequence:           1,
			filename:                 "",
			isParent:                 true,
			filesystem:               f,
			extensions: []directoryEntrySystemUseExtension{
				rockRidgePosixAttributes{mode: 0755 | os.ModeDir, linkCount: 1, uid: 0, gid: 0, length: 36},
				rockRidgeTimestamps{longForm: false, stamps: []rockRidgeTimestamp{
					{timestampType: rockRidgeTimestampModify, time: bytesToTime([]byte{0x76, 0x9, 0x1b, 0xb, 0x2f, 0x15, 0x0})},
					{timestampType: rockRidgeTimestampAccess, time: bytesToTime([]byte{0x76, 0x9, 0x1b, 0xb, 0x2f, 0x29, 0x0})},
					{timestampType: rockRidgeTimestampAttribute, time: bytesToTime([]byte{0x76, 0x9, 0x1b, 0xb, 0x2f, 0x15, 0x0})},
				},
				},
			},
		},
		&directoryEntry{
			extAttrSize:              0,
			location:                 0x14,
			size:                     0x800,
			creation:                 t1,
			isHidden:                 false,
			isSubdirectory:           true,
			isAssociated:             false,
			hasExtendedAttrs:         false,
			hasOwnerGroupPermissions: false,
			hasMoreEntries:           false,
			volumeSequence:           1,
			filename:                 "ABC",
			filesystem:               f,
			extensions: []directoryEntrySystemUseExtension{
				rockRidgePosixAttributes{mode: 0755 | os.ModeDir, linkCount: 1, uid: 0, gid: 0, length: 36},
				rockRidgeTimestamps{longForm: false, stamps: []rockRidgeTimestamp{
					{timestampType: rockRidgeTimestampModify, time: bytesToTime([]byte{0x76, 0x9, 0x1b, 0xb, 0x2f, 0x1e, 0x0})},
					{timestampType: rockRidgeTimestampAccess, time: bytesToTime([]byte{0x76, 0x9, 0x1b, 0xb, 0x2f, 0x29, 0x0})},
					{timestampType: rockRidgeTimestampAttribute, time: bytesToTime([]byte{0x76, 0x9, 0x1b, 0xb, 0x2f, 0x1e, 0x0})},
				},
				},
				rockRidgeName{name: "abc"},
			},
		},
		&directoryEntry{
			extAttrSize:              0,
			location:                 0x15,
			size:                     0x800,
			creation:                 t1,
			isHidden:                 false,
			isSubdirectory:           true,
			isAssociated:             false,
			hasExtendedAttrs:         false,
			hasOwnerGroupPermissions: false,
			hasMoreEntries:           false,
			volumeSequence:           1,
			filename:                 "BAR",
			filesystem:               f,
			extensions: []directoryEntrySystemUseExtension{
				rockRidgePosixAttributes{mode: 0755 | os.ModeDir, linkCount: 1, uid: 0, gid: 0, length: 36},
				rockRidgeTimestamps{longForm: false, stamps: []rockRidgeTimestamp{
					{timestampType: rockRidgeTimestampModify, time: bytesToTime([]byte{0x76, 0x9, 0x1b, 0xb, 0x2f, 0x1a, 0x0})},
					{timestampType: rockRidgeTimestampAccess, time: bytesToTime([]byte{0x76, 0x9, 0x1b, 0xb, 0x2f, 0x29, 0x0})},
					{timestampType: rockRidgeTimestampAttribute, time: bytesToTime([]byte{0x76, 0x9, 0x1b, 0xb, 0x2f, 0x1a, 0x0})},
				},
				},
				rockRidgeName{name: "bar"},
			},
		},
		&directoryEntry{
			extAttrSize:              0x0,
			location:                 0x16,
			size:                     0x800,
			creation:                 t1,
			isHidden:                 false,
			isSubdirectory:           true,
			isAssociated:             false,
			hasExtendedAttrs:         false,
			hasOwnerGroupPermissions: false,
			hasMoreEntries:           false,
			volumeSequence:           0x1,
			filesystem:               f,
			filename:                 "DEEP",
			extensions: []directoryEntrySystemUseExtension{
				rockRidgePosixAttributes{mode: 0755 | os.ModeDir, linkCount: 1, uid: 0, gid: 0, length: 36},
				rockRidgeTimestamps{longForm: false, stamps: []rockRidgeTimestamp{
					{timestampType: rockRidgeTimestampModify, time: bytesToTime([]byte{0x76, 0x0A, 0x13, 0x08, 0x0D, 0x32, 0x00})},
					{timestampType: rockRidgeTimestampAccess, time: bytesToTime([]byte{0x76, 0x0A, 0x13, 0x08, 0x0D, 0x32, 0x00})},
					{timestampType: rockRidgeTimestampAttribute, time: bytesToTime([]byte{0x76, 0x0A, 0x13, 0x08, 0x0D, 0x32, 0x00})},
				}},
				rockRidgeName{name: "deep"},
			},
		},
		&directoryEntry{
			extAttrSize:              0,
			location:                 0x1d,
			size:                     0x2800,
			creation:                 t1,
			isHidden:                 false,
			isSubdirectory:           true,
			isAssociated:             false,
			hasExtendedAttrs:         false,
			hasOwnerGroupPermissions: false,
			hasMoreEntries:           false,
			volumeSequence:           1,
			filename:                 "FOO",
			filesystem:               f,
			extensions: []directoryEntrySystemUseExtension{
				rockRidgePosixAttributes{mode: 0755 | os.ModeDir, linkCount: 1, uid: 0, gid: 0, length: 36},
				rockRidgeTimestamps{longForm: false, stamps: []rockRidgeTimestamp{
					{timestampType: rockRidgeTimestampModify, time: bytesToTime([]byte{0x76, 0x9, 0x1b, 0xb, 0x2f, 0x28, 0x0})},
					{timestampType: rockRidgeTimestampAccess, time: bytesToTime([]byte{0x76, 0x9, 0x1b, 0xb, 0x2f, 0x29, 0x0})},
					{timestampType: rockRidgeTimestampAttribute, time: bytesToTime([]byte{0x76, 0x9, 0x1b, 0xb, 0x2f, 0x28, 0x0})},
				},
				},
				rockRidgeName{name: "foo"},
			},
		},
		&directoryEntry{
			extAttrSize:              0,
			location:                 0x29,
			size:                     0x0,
			creation:                 t1,
			isHidden:                 false,
			isSubdirectory:           false,
			isAssociated:             false,
			hasExtendedAttrs:         false,
			hasOwnerGroupPermissions: false,
			hasMoreEntries:           false,
			volumeSequence:           1,
			filename:                 "LINK.;1",
			filesystem:               f,
			extensions: []directoryEntrySystemUseExtension{
				rockRidgePosixAttributes{mode: 0777 | os.ModeSymlink, linkCount: 1, uid: 0, gid: 0, length: 36},
				rockRidgeTimestamps{longForm: false, stamps: []rockRidgeTimestamp{
					{timestampType: rockRidgeTimestampModify, time: bytesToTime([]byte{0x76, 0xa, 0x13, 0x8, 0xd, 0x32, 0x0})},
					{timestampType: rockRidgeTimestampAccess, time: bytesToTime([]byte{0x76, 0xa, 0x13, 0x8, 0xd, 0x32, 0x0})},
					{timestampType: rockRidgeTimestampAttribute, time: bytesToTime([]byte{0x76, 0xa, 0x13, 0x8, 0xd, 0x32, 0x0})},
				},
				},
				rockRidgeName{name: "link"},
				rockRidgeSymlink{name: "/a/b/c/d/ef/g/h"},
			},
		},
		&directoryEntry{
			extAttrSize:              0,
			location:                 0x1476,
			size:                     0x7,
			creation:                 t1,
			isHidden:                 false,
			isSubdirectory:           false,
			isAssociated:             false,
			hasExtendedAttrs:         false,
			hasOwnerGroupPermissions: false,
			hasMoreEntries:           false,
			volumeSequence:           1,
			filename:                 "README.MD;1",
			filesystem:               f,
			extensions: []directoryEntrySystemUseExtension{
				rockRidgePosixAttributes{mode: 0644, linkCount: 1, uid: 0, gid: 0, length: 36},
				rockRidgeTimestamps{longForm: false, stamps: []rockRidgeTimestamp{
					{timestampType: rockRidgeTimestampModify, time: bytesToTime([]byte{0x76, 0xa, 0x13, 0x8, 0xd, 0x32, 0x0})},
					{timestampType: rockRidgeTimestampAccess, time: bytesToTime([]byte{0x76, 0xa, 0x13, 0x8, 0xd, 0x33, 0x0})},
					{timestampType: rockRidgeTimestampAttribute, time: bytesToTime([]byte{0x76, 0xa, 0x13, 0x8, 0xd, 0x32, 0x0})},
				},
				},
				rockRidgeName{name: "README.md"},
			},
		},
	}
	if includeRelocated {
		entries = append(entries,
			&directoryEntry{
				extAttrSize:              0,
				location:                 0x22,
				size:                     0x800,
				creation:                 t1,
				isHidden:                 false,
				isSubdirectory:           true,
				isAssociated:             false,
				hasExtendedAttrs:         false,
				hasOwnerGroupPermissions: false,
				hasMoreEntries:           false,
				volumeSequence:           1,
				filename:                 "G",
				filesystem:               f,
				extensions: []directoryEntrySystemUseExtension{
					rockRidgePosixAttributes{mode: 0755 | os.ModeDir, linkCount: 1, uid: 0, gid: 0, length: 36},
					rockRidgeTimestamps{longForm: false, stamps: []rockRidgeTimestamp{
						{timestampType: rockRidgeTimestampModify, time: bytesToTime([]byte{0x76, 0xa, 0x13, 0x8, 0xd, 0x32, 0x0})},
						{timestampType: rockRidgeTimestampAccess, time: bytesToTime([]byte{0x76, 0xa, 0x13, 0x8, 0xd, 0x32, 0x0})},
						{timestampType: rockRidgeTimestampAttribute, time: bytesToTime([]byte{0x76, 0xa, 0x13, 0x8, 0xd, 0x32, 0x0})},
					},
					},
					rockRidgeRelocatedDirectory{},
					rockRidgeName{name: "g"},
				},
			})
	}

	// b is a slice of byte slices; each one represents the raw directory entry bytes for a directoryEntry
	//  parallels the directory Entries above
	// each entry is a slice of slices: the first is the raw data for the entry
	//  the second and any subsequent are continuation areas
	b := make([][][]byte, 0, 8)
	read := 0
	for range entries {
		recordSize := int(allBytes[read])
		// do we have a 0 point? if so, move ahead until we pass it at the end of the block
		if recordSize == 0x00 {
			read += (blocksize - read%blocksize)
		}
		b2 := make([][]byte, 0)
		b2 = append(b2, allBytes[read:read+recordSize])
		b = append(b, b2)
		read += recordSize
	}
	// the first one has a continuation entry
	b[0] = append(b[0], ceBytes)

	byteMap := map[int][]byte{
		rootSector: allBytes, ceSector: ceBytes,
	}
	return entries, b, allBytes, byteMap, nil
}

func getValidDirectoryEntriesExtended(fs *FileSystem) ([]*directoryEntry, [][]byte, []byte, error) {
	// these are taken from the file ./testdata/9660.iso, see ./testdata/README.md
	blocksize := 2048
	fooSector := 0x21
	t1, _ := time.Parse(time.RFC3339, "2017-11-26T07:53:16Z")
	sizes := []int{0x34, 0x34, 0x2c}
	entries := []*directoryEntry{
		{extAttrSize: 0x0, location: 0x21, size: 0x1000, creation: t1, isHidden: false, isSubdirectory: true, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "", isSelf: true},
		{extAttrSize: 0x0, location: 0x12, size: 0x800, creation: t1, isHidden: false, isSubdirectory: true, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "", isParent: true},
		{extAttrSize: 0x0, location: 0x1427, size: 0xb, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA00.;1"},
		{extAttrSize: 0x0, location: 0x1428, size: 0xb, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA01.;1"},
		{extAttrSize: 0x0, location: 0x1429, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA02.;1"},
		{extAttrSize: 0x0, location: 0x142a, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA03.;1"},
		{extAttrSize: 0x0, location: 0x142b, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA04.;1"},
		{extAttrSize: 0x0, location: 0x142c, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA05.;1"},
		{extAttrSize: 0x0, location: 0x142d, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA06.;1"},
		{extAttrSize: 0x0, location: 0x142e, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA07.;1"},
		{extAttrSize: 0x0, location: 0x142f, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA08.;1"},
		{extAttrSize: 0x0, location: 0x1430, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA09.;1"},
		{extAttrSize: 0x0, location: 0x1431, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA10.;1"},
		{extAttrSize: 0x0, location: 0x1432, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA11.;1"},
		{extAttrSize: 0x0, location: 0x1433, size: 0xb, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA12.;1"},
		{extAttrSize: 0x0, location: 0x1434, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA13.;1"},
		{extAttrSize: 0x0, location: 0x1435, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA14.;1"},
		{extAttrSize: 0x0, location: 0x1436, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA15.;1"},
		{extAttrSize: 0x0, location: 0x1437, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA16.;1"},
		{extAttrSize: 0x0, location: 0x1438, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA17.;1"},
		{extAttrSize: 0x0, location: 0x1439, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA18.;1"},
		{extAttrSize: 0x0, location: 0x143a, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA19.;1"},
		{extAttrSize: 0x0, location: 0x143b, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA20.;1"},
		{extAttrSize: 0x0, location: 0x143c, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA21.;1"},
		{extAttrSize: 0x0, location: 0x143d, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA22.;1"},
		{extAttrSize: 0x0, location: 0x143e, size: 0xb, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA23.;1"},
		{extAttrSize: 0x0, location: 0x143f, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA24.;1"},
		{extAttrSize: 0x0, location: 0x1440, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA25.;1"},
		{extAttrSize: 0x0, location: 0x1441, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA26.;1"},
		{extAttrSize: 0x0, location: 0x1442, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA27.;1"},
		{extAttrSize: 0x0, location: 0x1443, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA28.;1"},
		{extAttrSize: 0x0, location: 0x1444, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA29.;1"},
		{extAttrSize: 0x0, location: 0x1445, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA30.;1"},
		{extAttrSize: 0x0, location: 0x1446, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA31.;1"},
		{extAttrSize: 0x0, location: 0x1447, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA32.;1"},
		{extAttrSize: 0x0, location: 0x1448, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA33.;1"},
		{extAttrSize: 0x0, location: 0x1449, size: 0xb, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA34.;1"},
		{extAttrSize: 0x0, location: 0x144a, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA35.;1"},
		{extAttrSize: 0x0, location: 0x144b, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA36.;1"},
		{extAttrSize: 0x0, location: 0x144c, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA37.;1"},
		{extAttrSize: 0x0, location: 0x144d, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA38.;1"},
		{extAttrSize: 0x0, location: 0x144e, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA39.;1"},
		{extAttrSize: 0x0, location: 0x144f, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA40.;1"},
		{extAttrSize: 0x0, location: 0x1450, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA41.;1"},
		{extAttrSize: 0x0, location: 0x1451, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA42.;1"},
		{extAttrSize: 0x0, location: 0x1452, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA43.;1"},
		{extAttrSize: 0x0, location: 0x1453, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA44.;1"},
		{extAttrSize: 0x0, location: 0x1454, size: 0xb, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA45.;1"},
		{extAttrSize: 0x0, location: 0x1455, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA46.;1"},
		{extAttrSize: 0x0, location: 0x1456, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA47.;1"},
		{extAttrSize: 0x0, location: 0x1457, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA48.;1"},
		{extAttrSize: 0x0, location: 0x1458, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA49.;1"},
		{extAttrSize: 0x0, location: 0x1459, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA50.;1"},
		{extAttrSize: 0x0, location: 0x145a, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA51.;1"},
		{extAttrSize: 0x0, location: 0x145b, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA52.;1"},
		{extAttrSize: 0x0, location: 0x145c, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA53.;1"},
		{extAttrSize: 0x0, location: 0x145d, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA54.;1"},
		{extAttrSize: 0x0, location: 0x145e, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA55.;1"},
		{extAttrSize: 0x0, location: 0x145f, size: 0xb, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA56.;1"},
		{extAttrSize: 0x0, location: 0x1460, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA57.;1"},
		{extAttrSize: 0x0, location: 0x1461, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA58.;1"},
		{extAttrSize: 0x0, location: 0x1462, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA59.;1"},
		{extAttrSize: 0x0, location: 0x1463, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA60.;1"},
		{extAttrSize: 0x0, location: 0x1464, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA61.;1"},
		{extAttrSize: 0x0, location: 0x1465, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA62.;1"},
		{extAttrSize: 0x0, location: 0x1466, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA63.;1"},
		{extAttrSize: 0x0, location: 0x1467, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA64.;1"},
		{extAttrSize: 0x0, location: 0x1468, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA65.;1"},
		{extAttrSize: 0x0, location: 0x1469, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA66.;1"},
		{extAttrSize: 0x0, location: 0x146a, size: 0xb, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA67.;1"},
		{extAttrSize: 0x0, location: 0x146b, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA68.;1"},
		{extAttrSize: 0x0, location: 0x146c, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA69.;1"},
		{extAttrSize: 0x0, location: 0x146d, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA70.;1"},
		{extAttrSize: 0x0, location: 0x146e, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA71.;1"},
		{extAttrSize: 0x0, location: 0x146f, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA72.;1"},
		{extAttrSize: 0x0, location: 0x1470, size: 0xc, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA73.;1"},
		{extAttrSize: 0x0, location: 0x1471, size: 0xb, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA74.;1"},
		{extAttrSize: 0x0, location: 0x1472, size: 0xb, creation: t1, isHidden: false, isSubdirectory: false, isAssociated: false, hasExtendedAttrs: false, hasOwnerGroupPermissions: false, hasMoreEntries: false, volumeSequence: 0x1, filename: "FILENA75.;1"},
	}

	for _, e := range entries {
		e.filesystem = fs
	}
	// read correct bytes off of disk
	input, err := ioutil.ReadFile(ISO9660File)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("Error reading data from iso9660 test fixture %s: %v", ISO9660File, err)
	}

	start := fooSector * blocksize // start of /foo directory in file.iso

	// five blocks, since we know it is five blocks
	allBytes := input[start : start+2*blocksize]
	b := make([][]byte, 0, len(entries))
	read := 0
	for i := range entries {
		var recordSize int
		if i < len(sizes) {
			recordSize = sizes[1]
		} else {
			recordSize = sizes[len(sizes)-1]
		}
		// do we have a 0 point? if so, move ahead until we pass it at the end of the block
		if allBytes[read] == 0x00 {
			read += (blocksize - read%blocksize)
		}
		b = append(b, allBytes[read:read+recordSize])
		read += recordSize
	}
	return entries, b, allBytes, nil
}

func TestBytesToTime(t *testing.T) {
	for _, tt := range timeBytesTests {
		output := bytesToTime(tt.b)
		expected, err := time.Parse(time.RFC3339, tt.rfc)
		if err != nil {
			t.Fatalf("Error parsing expected date: %v", err)
		}
		if !expected.Equal(output) {
			t.Errorf("bytesToTime(%d) expected output %v, actual %v", tt.b, expected, output)
		}
	}
}

func TestTimeToBytes(t *testing.T) {
	for _, tt := range timeBytesTests {
		input, err := time.Parse(time.RFC3339, tt.rfc)
		if err != nil {
			t.Fatalf("Error parsing input date: %v", err)
		}
		b := timeToBytes(input)
		if bytes.Compare(b, tt.b) != 0 {
			t.Errorf("timeToBytes(%v) expected output %x, actual %x", tt.rfc, tt.b, b)
		}
	}

}

func TestDirectoryEntryStringToASCIIBytes(t *testing.T) {
	tests := []struct {
		input  string
		output []byte
		err    error
	}{
		{"abc", []byte{0x61, 0x62, 0x63}, nil},
		{"abcdefg", []byte{0x61, 0x62, 0x63, 0x64, 0x65, 0x66, 0x67}, nil},
		{"abcdef\u2318", nil, fmt.Errorf("Non-ASCII character in name: %s", "abcdef\u2318")},
	}
	for _, tt := range tests {
		output, err := stringToASCIIBytes(tt.input)
		if bytes.Compare(output, tt.output) != 0 {
			t.Errorf("stringToASCIIBytes(%s) expected output %v, actual %v", tt.input, tt.output, output)
		}
		if (err != nil && tt.err == nil) || (err == nil && tt.err != nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())) {
			t.Errorf("mismatched err expected, actual: %v, %v", tt.err, err)
		}
	}

}

func TestDirectoryEntryUCaseValid(t *testing.T) {
	tests := []struct {
		input  string
		output string
	}{
		{"abc", "ABC"},
		{"ABC", "ABC"},
		{"aBC", "ABC"},
		{"a15D", "A15D"},
		{"A BC", "ABC"},
		{"A..-a*)82y12112bb", "A_A__82Y12112BB"},
	}
	for _, tt := range tests {
		output := uCaseValid(tt.input)
		if output != tt.output {
			t.Errorf("uCaseValid(%s) expected %s actual %s", tt.input, tt.output, output)
		}
	}
}

func TestDirectoryEntryParseDirEntries(t *testing.T) {
	fs := &FileSystem{blocksize: 2048, suspEnabled: true, suspExtensions: []suspExtension{getRockRidgeExtension("RRIP_1991A")}}
	validDe, _, b, byteMap, err := getRockRidgeDirectoryEntries(fs, false)
	if err != nil {
		t.Fatal(err)
	}
	// need this to read the continuation area
	f := &testhelper.FileImpl{
		Reader: func(b []byte, offset int64) (int, error) {
			location := int(offset / fs.blocksize)
			if b2, ok := byteMap[location]; ok {
				copy(b, b2)
				if len(b2) < len(b) {
					return len(b2), nil
				}
				return len(b), nil
			}
			return 0, fmt.Errorf("Unknown area to read %d", offset)
		},
	}
	fs.file = f
	tests := []struct {
		de  []*directoryEntry
		b   []byte
		err error
	}{
		{validDe, b, nil},
	}

	for _, tt := range tests {
		output, err := parseDirEntries(tt.b, fs)
		switch {
		case (err != nil && tt.err == nil) || (err == nil && tt.err != nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())):
			t.Errorf("mismatched err actual, then expected:\n%v\n%v", err, tt.err)
		case (output == nil && tt.de != nil) || (tt.de == nil && output != nil):
			t.Errorf("parseDirEntries() DirectoryEntry mismatched nil actual, expected %v %v", output, tt.de)
		case len(output) != len(tt.de):
			t.Errorf("parseDirEntries() DirectoryEntry mismatched length actual, expected %d %d", len(output), len(tt.de))
		default:
			for i, de := range output {
				if !compareDirectoryEntries(de, tt.de[i], false, true) {
					t.Errorf("%d: parseDirEntries() DirectoryEntry mismatch, actual then expected:", i)
					t.Logf("%#v\n", de)
					t.Logf("%#v\n", tt.de[i])
				}
			}
		}
	}

}

func TestDirectoryEntryToBytes(t *testing.T) {
	fs := &FileSystem{
		blocksize: int64(2048),
	}
	validDe, validBytes, _, _, err := getRockRidgeDirectoryEntries(fs, true)
	if err != nil {
		t.Fatal(err)
	}
	for i, de := range validDe[0:1] {
		b, err := de.toBytes(false, []uint32{19})
		switch {
		case err != nil:
			t.Errorf("Error converting directory entry to bytes: %v", err)
			t.Logf("%v", de)
		case int(b[0][0]) != len(b[0]):
			t.Errorf("Reported size as %d but had %d bytes", b[0], len(b))
		default:
			// compare the actual dir entry
			if bytes.Compare(directoryEntryBytesNullDate(b[0]), directoryEntryBytesNullDate(validBytes[i][0])) != 0 {
				t.Errorf("%d: Mismatched entry bytes %s, actual vs expected", i, de.filename)
				t.Log(b[0])
				t.Log(validBytes[i])
			}
			// compare the continuation entries
			if len(validBytes[i]) != len(b) {
				t.Errorf("%d: Mismatched number of continuation entries actual %d expected %d", i, len(b)-1, len(validBytes[i])-1)
			}
			for j, e := range validBytes[i][1:] {
				if bytes.Compare(e, b[j+1]) != 0 {
					t.Errorf("%d: mismatched continuation entry bytes, actual then expected", i)
					t.Log(b[j+1])
					t.Log(e)
				}
			}
		}
	}
}

func TestDirectoryEntryGetLocation(t *testing.T) {
	// directoryEntryGetLocation(p string) (uint32, uint32, error) {
	tests := []struct {
		input  string
		output uint32
		err    error
	}{
		{"/", 0x12, nil},
		{"/ABC", 0x13, nil},
		{"/FOO", 0x21, nil},
		{"/NOTHERE", 0, nil},
	}

	f, err := os.Open(ISO9660File)
	if err != nil {
		t.Fatalf("Could not open iso testing file %s: %v", ISO9660File, err)
	}
	// the root directory entry
	root := &directoryEntry{
		extAttrSize:              0,
		location:                 0x12,
		size:                     0x800,
		creation:                 time.Now(),
		isHidden:                 false,
		isSubdirectory:           true,
		isAssociated:             false,
		hasExtendedAttrs:         false,
		hasOwnerGroupPermissions: false,
		hasMoreEntries:           false,
		volumeSequence:           1,
		filename:                 string(0x00),
		filesystem:               &FileSystem{blocksize: 2048, file: f},
	}

	for _, tt := range tests {
		// root directory entry needs a filesystem or this will error out
		output, _, err := root.getLocation(tt.input)
		if output != tt.output {
			t.Errorf("directoryEntry.getLocation(%s) expected output %d, actual %d", tt.input, tt.output, output)
		}
		if (err != nil && tt.err == nil) || (err == nil && tt.err != nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())) {
			t.Errorf("mismatched err expected, actual: %v, %v", tt.err, err)
		}
	}
}
