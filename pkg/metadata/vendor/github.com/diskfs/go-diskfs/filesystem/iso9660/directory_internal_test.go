package iso9660

import (
	"fmt"
	"testing"

	"github.com/diskfs/go-diskfs/testhelper"
)

// TestDirectoryEntriesFromBytes largely a duplicate of TestdirectoryEntryParseDirEntries
// it just loads it into the Directory structure
func TestDirectoryEntriesFromBytes(t *testing.T) {
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

	d := &Directory{}

	err = d.entriesFromBytes(b, fs)
	switch {
	case err != nil:
		t.Errorf("Unexpected non-nil error: %v", err)
	case d.entries == nil:
		t.Errorf("unexpected nil entries")
	case len(d.entries) != len(validDe):
		t.Errorf("mismatched entries length actual %d vs expected %d", len(d.entries), len(validDe))
	default:
		// run through them and see that they match
		for i, de := range d.entries {
			if !compareDirectoryEntries(de, validDe[i], false, true) {
				t.Errorf("%d: directoryEntry mismatch, actual then valid:", i)
				t.Logf("%#v\n", de)
				t.Logf("%#v\n", validDe[i])
			}
		}
	}
}

func TestDirectoryEntriesToBytes(t *testing.T) {
	blocksize := 2048
	fs := &FileSystem{
		blocksize: int64(blocksize),
	}
	validDe, _, b, _, err := getRockRidgeDirectoryEntries(fs, false)
	if err != nil {
		t.Fatal(err)
	}
	d := &Directory{
		entries: validDe,
		directoryEntry: directoryEntry{
			filesystem: fs,
		},
	}
	dirBytes, err := d.entriesToBytes([]uint32{19})
	// dirBytes is [][]byte where each entry is a block, the first being the dir itself, the rest CE blocks (if any)
	if err != nil {
		t.Fatalf("unexpected non-nil error: %v", err)
	}
	// null the date bytes out
	dirEntryBytes := clearDatesDirectoryBytes(dirBytes[0], blocksize)
	b = clearDatesDirectoryBytes(b, blocksize)
	switch {
	case dirEntryBytes == nil:
		t.Errorf("unexpected nil bytes")
	case len(dirEntryBytes) == 0:
		t.Errorf("unexpected 0 length byte slice")
	case len(dirEntryBytes) != len(b):
		t.Errorf("mismatched byte slice length actual %d, expected %d", len(dirEntryBytes), len(b))
	case len(dirEntryBytes)%blocksize != 0:
		t.Errorf("output size was %d which is not a perfect multiple of %d", len(dirEntryBytes), blocksize)
	}
}

func clearDatesDirectoryBytes(b []byte, blocksize int) []byte {
	if b == nil {
		return b
	}
	nullBytes := make([]byte, 7, 7)
	for i := 0; i < len(b); {
		// get the length of the current record
		dirlen := int(b[i])
		if dirlen == 0 {
			i += blocksize - blocksize%i
			continue
		}
		copy(b[i+18:i+18+7], nullBytes)
		i += dirlen
	}
	return b
}
