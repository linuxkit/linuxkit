package iso9660

import (
	"os"
	"testing"
)

func TestIso9660ReadDirectory(t *testing.T) {
	// will use the file.iso fixture to test an actual directory
	// \ (root directory) should be in one block
	// \FOO should be in multiple blocks
	file, err := os.Open(ISO9660File)
	defer file.Close()
	if err != nil {
		t.Fatalf("Could not open file %s to read: %v", ISO9660File, err)
	}
	// FileSystem implements the FileSystem interface
	pathTable, _, _, err := get9660PathTable()
	if err != nil {
		t.Fatalf("Could not get path table: %v", err)
	}
	fs := &FileSystem{
		workspace: "", // we only ever call readDirectory with no workspace
		size:      ISO9660Size,
		start:     0,
		file:      file,
		blocksize: 2048,
		pathTable: pathTable,
	}
	validDe, _, _, _, err := get9660DirectoryEntries(fs)
	if err != nil {
		t.Fatalf("Unable to read valid directory entries: %v", err)
	}
	validDeExtended, _, _, err := getValidDirectoryEntriesExtended(fs)
	if err != nil {
		t.Fatalf("Unable to read valid directory entries extended: %v", err)
	}
	fs.rootDir = validDe[0] // validDe contains root directory entries, first one is the root itself

	tests := []struct {
		path    string
		entries []*directoryEntry
	}{
		{`\`, validDe},
		{"/", validDe},
		{`\FOO`, validDeExtended},
		{`/FOO`, validDeExtended},
	}
	for _, tt := range tests {
		entries, err := fs.readDirectory(tt.path)
		switch {
		case err != nil:
			t.Errorf("fs.readDirectory(%s): unexpected nil error: %v", tt.path, err)
		case len(entries) != len(tt.entries):
			t.Errorf("fs.readDirectory(%s): number of entries do not match, actual %d expected %d", tt.path, len(entries), len(tt.entries))
		default:
			for i, entry := range entries {
				if !compareDirectoryEntries(entry, tt.entries[i], false, false) {
					t.Errorf("fs.readDirectory(%s) %d: entries do not match, actual then expected", tt.path, i)
					t.Logf("%#v\n", entry)
					t.Logf("%#v\n", tt.entries[i])
				}
			}
		}
	}
}

func TestRockRidgeReadDirectory(t *testing.T) {
	// will use the file.iso fixture to test an actual directory
	// \ (root directory) should be in one block
	// \FOO should be in multiple blocks
	file, err := os.Open(RockRidgeFile)
	defer file.Close()
	if err != nil {
		t.Fatalf("Could not open file %s to read: %v", RockRidgeFile, err)
	}
	// FileSystem implements the FileSystem interface
	pathTable, _, _, err := getRockRidgePathTable()
	if err != nil {
		t.Fatalf("Could not get path table: %v", err)
	}
	fs := &FileSystem{
		workspace:      "", // we only ever call readDirectory with no workspace
		size:           ISO9660Size,
		start:          0,
		file:           file,
		blocksize:      2048,
		pathTable:      pathTable,
		suspEnabled:    true,
		suspExtensions: []suspExtension{getRockRidgeExtension("RRIP_1991A")},
	}
	validDe, _, _, _, err := getRockRidgeDirectoryEntries(fs, false)
	if err != nil {
		t.Fatalf("Unable to read valid directory entries: %v", err)
	}
	fs.rootDir = validDe[0] // validDe contains root directory entries, first one is the root itself

	tests := []struct {
		path    string
		entries []*directoryEntry
	}{
		{`\`, validDe},
		{"/", validDe},
	}
	for _, tt := range tests {
		entries, err := fs.readDirectory(tt.path)
		switch {
		case err != nil:
			t.Errorf("fs.readDirectory(%s): unexpected nil error: %v", tt.path, err)
		case len(entries) != len(tt.entries):
			t.Errorf("fs.readDirectory(%s): number of entries do not match, actual %d expected %d", tt.path, len(entries), len(tt.entries))
		default:
			for i, entry := range entries {
				if !compareDirectoryEntries(entry, tt.entries[i], false, false) {
					t.Errorf("fs.readDirectory(%s) %d %s: entries do not match, actual then expected", tt.path, i, entry.filename)
					t.Logf("%#v\n", entry)
					t.Logf("%#v\n", tt.entries[i])
				}
			}
		}
	}
}

func TestLabel(t *testing.T) {
	t.Run("no primary volume descriptor", func(t *testing.T) {
		expected := ""
		fs := FileSystem{}
		label := fs.Label()
		if label != expected {
			t.Errorf("mismatched labels, actual '%s' expected '%s'", label, expected)
		}
	})
	t.Run("primary volume descriptor no label", func(t *testing.T) {
		expected := ""
		fs := FileSystem{
			volumes: volumeDescriptors{
				primary: &primaryVolumeDescriptor{},
			},
		}
		label := fs.Label()
		if label != expected {
			t.Errorf("mismatched labels, actual '%s' expected '%s'", label, expected)
		}
	})
	t.Run("primary volume descriptor with label", func(t *testing.T) {
		expected := "myisolabel"
		fs := FileSystem{
			volumes: volumeDescriptors{
				primary: &primaryVolumeDescriptor{
					volumeIdentifier: expected,
				},
			},
		}
		label := fs.Label()
		if label != expected {
			t.Errorf("mismatched labels, actual '%s' expected '%s'", label, expected)
		}
	})
}
