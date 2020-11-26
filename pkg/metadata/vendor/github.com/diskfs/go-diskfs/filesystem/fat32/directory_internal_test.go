package fat32

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

// TestDirectoryEntriesFromBytes largely a duplicate of TestdirectoryEntryParseDirEntries
// it just loads it into the Directory structure
func TestDirectoryEntriesFromBytes(t *testing.T) {
	validDe, validBytes, err := getValidDirectoryEntries()
	if err != nil {
		t.Fatal(err)
	}
	// validBytes is ordered [][]byte - just string them all together
	b := make([]byte, 0)
	for _, b2 := range validBytes {
		b = append(b, b2...)
	}

	d := &Directory{}
	err = d.entriesFromBytes(b, nil)
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
			if *de != *validDe[i] {
				t.Errorf("%d: directoryEntry mismatch, actual then valid:", i)
				t.Log(de)
				t.Log(validDe[i])
			}
		}
	}

}

func TestDirectoryEntriesToBytes(t *testing.T) {
	validDe, validBytes, err := getValidDirectoryEntries()
	bytesPerCluster := 2048
	if err != nil {
		t.Fatal(err)
	}
	// validBytes is ordered [][]byte - just string them all together
	b := make([]byte, 0)
	for _, b2 := range validBytes {
		b = append(b, b2...)
	}
	d := &Directory{
		entries: validDe,
		directoryEntry: directoryEntry{
			filesystem: &FileSystem{
				bytesPerCluster: bytesPerCluster,
			},
		},
	}
	output, err := d.entriesToBytes(bytesPerCluster)
	switch {
	case err != nil:
		t.Errorf("unexpected non-nil error: %v", err)
	case output == nil:
		t.Errorf("unexpected nil bytes")
	case len(output) == 0:
		t.Errorf("unexpected 0 length byte slice")
	case len(output) != len(b):
		t.Errorf("mismatched byte slice length actual %d, expected %d", len(output), len(b))
	case bytes.Compare(output, b) != 0:
		t.Errorf("Mismatched output of bytes. Actual then expected:")
		t.Logf("%v", output)
		t.Logf("%v", b)
	}
}

func TestDirectoryCreateEntry(t *testing.T) {
	tests := []struct {
		name    string
		cluster uint32
		dir     bool
		de      *directoryEntry
	}{
		{"SHORT", 25, false, &directoryEntry{
			filenameShort:   "SHORT",
			fileExtension:   "",
			filenameLong:    "",
			isSubdirectory:  false,
			clusterLocation: 25,
		}},
		{"long", 55, false, &directoryEntry{
			filenameShort:   "LONG",
			fileExtension:   "",
			filenameLong:    "long",
			isSubdirectory:  false,
			clusterLocation: 55,
		}},
		{"long.txt", 99, true, &directoryEntry{
			filenameShort:   "LONG",
			fileExtension:   "TXT",
			filenameLong:    "long.txt",
			isSubdirectory:  true,
			clusterLocation: 99,
		}},
	}

	d := &Directory{}
	now := time.Now()
	for _, tt := range tests {
		output, err := d.createEntry(tt.name, tt.cluster, tt.dir)
		msg := fmt.Sprintf("createEntry(%s, %d, %t)", tt.name, tt.cluster, tt.dir)
		switch {
		case err != nil:
			t.Errorf("%s returned non-nil error: %v", msg, err)
		case output == nil:
			t.Errorf("%s returned nil directoryEntry", msg)
		case output.filenameLong != tt.de.filenameLong:
			t.Errorf("%s mismatched long filename actual %s vs expected %s", msg, output.filenameLong, tt.de.filenameLong)
		case output.filenameShort != tt.de.filenameShort:
			t.Errorf("%s mismatched short filename actual %s vs expected %s", msg, output.filenameShort, tt.de.filenameShort)
		case output.fileExtension != tt.de.fileExtension:
			t.Errorf("%s mismatched file extension actual %s vs expected %s", msg, output.fileExtension, tt.de.fileExtension)
		case output.clusterLocation != tt.de.clusterLocation:
			t.Errorf("%s mismatched cluster location actual %d vs expected %d", msg, output.clusterLocation, tt.de.clusterLocation)
		case output.isSubdirectory != tt.de.isSubdirectory:
			t.Errorf("%s mismatched subdir setting actual %t vs expected %t", msg, output.isSubdirectory, tt.de.isSubdirectory)
		// check create, modify and access times
		case output.createTime.Unix()-now.Unix() > 1000:
			t.Errorf("%s create time too far from current time, actual %v now %v", msg, output.createTime, now)
		case output.modifyTime.Unix()-now.Unix() > 1000:
			t.Errorf("%s modify time too far from current time, actual %v now %v", msg, output.modifyTime, now)
		case output.accessTime.Unix()-now.Unix() > 1000:
			t.Errorf("%s access time too far from current time, actual %v now %v", msg, output.accessTime, now)
		}
	}
}
