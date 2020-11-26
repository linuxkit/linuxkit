package iso9660

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"testing"
	"time"
)

func TestRockRidgeID(t *testing.T) {
	id := "abc"
	rr := &rockRidgeExtension{id: id}
	if rr.ID() != id {
		t.Errorf("Mismatched signature, actual '%s' expected '%s'", rr.ID(), id)
	}
}

func TestRockRidgeGetFilename(t *testing.T) {
	tests := []struct {
		dirEntry *directoryEntry
		filename string
		err      error
	}{
		{&directoryEntry{filename: "ABC"}, "", fmt.Errorf("Could not find Rock Ridge filename property")},
		{&directoryEntry{filename: "ABC", extensions: []directoryEntrySystemUseExtension{rockRidgeName{name: "abc"}}}, "abc", nil},
	}
	rr := &rockRidgeExtension{}
	for _, tt := range tests {
		name, err := rr.GetFilename(tt.dirEntry)
		if (err != nil && tt.err == nil) || (err == nil && tt.err != nil) {
			t.Errorf("Mismatched errors, actual then expected")
			t.Log(err)
			t.Log(tt.err)
		} else if name != tt.filename {
			t.Errorf("Mismatched filename actual %s expected %s", name, tt.filename)
		}
	}
}

func TestRockRidgeRelocated(t *testing.T) {
	tests := []struct {
		dirEntry  *directoryEntry
		relocated bool
	}{
		{&directoryEntry{filename: "ABC"}, false},
		{&directoryEntry{filename: "ABC", extensions: []directoryEntrySystemUseExtension{rockRidgeRelocatedDirectory{}}}, true},
	}
	rr := &rockRidgeExtension{}
	for _, tt := range tests {
		reloc := rr.Relocated(tt.dirEntry)
		if reloc != tt.relocated {
			t.Errorf("Mismatched relocated actual %v expected %v", reloc, tt.relocated)
		}
	}
}

func TestRockRidgeUsePathtable(t *testing.T) {
	rr := &rockRidgeExtension{}
	if rr.UsePathtable() {
		t.Errorf("Rock Ridge extension erroneously said to use pathtable")
	}
}

func TestRockRidgeSymlinkMerge(t *testing.T) {
	tests := []struct {
		first        rockRidgeSymlink
		continuation []directoryEntrySystemUseExtension
		result       rockRidgeSymlink
	}{
		{rockRidgeSymlink{name: "/a/b", continued: true}, []directoryEntrySystemUseExtension{rockRidgeSymlink{name: "/c/d", continued: true}, rockRidgeSymlink{name: "/e/f", continued: false}}, rockRidgeSymlink{name: "/a/b/c/d/e/f", continued: false}},
		{rockRidgeSymlink{name: "/a/b", continued: true}, []directoryEntrySystemUseExtension{rockRidgeSymlink{name: "/c/d", continued: false}}, rockRidgeSymlink{name: "/a/b/c/d", continued: false}},
		{rockRidgeSymlink{name: "/a/b", continued: false}, nil, rockRidgeSymlink{name: "/a/b", continued: false}},
	}
	for _, tt := range tests {
		symlink := tt.first.Merge(tt.continuation)
		if symlink != tt.result {
			t.Errorf("Mismatched merge result actual %v expected %v", symlink, tt.result)
		}
	}
}

func TestRockRidgeNameMerge(t *testing.T) {
	tests := []struct {
		first        rockRidgeName
		continuation []directoryEntrySystemUseExtension
		result       rockRidgeName
	}{
		{rockRidgeName{name: "/a/b", continued: true}, []directoryEntrySystemUseExtension{rockRidgeName{name: "/c/d", continued: true}, rockRidgeName{name: "/e/f", continued: false}}, rockRidgeName{name: "/a/b/c/d/e/f", continued: false}},
		{rockRidgeName{name: "/a/b", continued: true}, []directoryEntrySystemUseExtension{rockRidgeName{name: "/c/d", continued: false}}, rockRidgeName{name: "/a/b/c/d", continued: false}},
		{rockRidgeName{name: "/a/b", continued: false}, nil, rockRidgeName{name: "/a/b", continued: false}},
	}
	for _, tt := range tests {
		name := tt.first.Merge(tt.continuation)
		if name != tt.result {
			t.Errorf("Mismatched merge result actual %v expected %v", name, tt.result)
		}
	}
}

func TestRockRidgeSortTimestamp(t *testing.T) {
	// these are ust sorted randomly
	tests := []rockRidgeTimestamp{
		{timestampType: rockRidgeTimestampExpiration},
		{timestampType: rockRidgeTimestampModify},
		{timestampType: rockRidgeTimestampEffective},
		{timestampType: rockRidgeTimestampAttribute},
		{timestampType: rockRidgeTimestampCreation},
		{timestampType: rockRidgeTimestampAccess},
		{timestampType: rockRidgeTimestampBackup},
	}
	expected := []uint8{rockRidgeTimestampCreation, rockRidgeTimestampModify, rockRidgeTimestampAccess,
		rockRidgeTimestampAttribute, rockRidgeTimestampBackup, rockRidgeTimestampExpiration, rockRidgeTimestampEffective}
	sort.Sort(rockRidgeTimestampByBitOrder(tests))
	for i, e := range tests {
		if e.timestampType != expected[i] {
			t.Errorf("At position %d, got %v instead of %v", i, e.timestampType, expected[i])
		}
	}
}

func TestGetExtensions(t *testing.T) {
	// create an extension object and test files
	rr := getRockRidgeExtension(rockRidge112)
	pxLength := rr.pxLength
	dir, err := ioutil.TempDir("", "rockridge")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir) // clean up

	self, err := user.Current()
	if err != nil {
		t.Fatalf("Unable to get current uid/gid: %v", err)
	}
	uidI, err := strconv.Atoi(self.Uid)
	if err != nil {
		t.Fatalf("Unable to convert uid to int: %v", err)
	}
	gidI, err := strconv.Atoi(self.Gid)
	if err != nil {
		t.Fatalf("Unable to convert gid to int: %v", err)
	}
	uid := uint32(uidI)
	gid := uint32(gidI)
	now := time.Now()

	// symlinks have fixed perms based on OS, we will get it and then set it
	if err = os.Symlink("testa", "testb"); err != nil {
		t.Fatalf("unable to create test symlink %s: %v", "testb", err)
	}
	defer os.Remove("testb")
	fi, err := os.Lstat("testb")
	if err != nil {
		t.Fatalf("unable to ready file info for test symlink: %v", err)
	}
	symMode := fi.Mode() & 0777

	tests := []struct {
		name       string
		self       bool
		parent     bool
		extensions []directoryEntrySystemUseExtension
		createFile func(string)
	}{
		// regular file
		{"regular01", false, false, []directoryEntrySystemUseExtension{
			rockRidgePosixAttributes{mode: 0764, linkCount: 1, uid: uid, gid: gid, length: pxLength},
			rockRidgeTimestamps{stamps: []rockRidgeTimestamp{
				rockRidgeTimestamp{timestampType: rockRidgeTimestampModify, time: now},
				rockRidgeTimestamp{timestampType: rockRidgeTimestampAccess, time: now},
				rockRidgeTimestamp{timestampType: rockRidgeTimestampAttribute, time: now},
			},
			},
			rockRidgeName{name: "regular01"},
		}, func(path string) {
			if err := ioutil.WriteFile(path, []byte("some data"), 0764); err != nil {
				t.Fatalf("unable to create regular file %s: %v", path, err)
			}
			// because of umask, must set explicitly
			if err := os.Chmod(path, 0764); err != nil {
				t.Fatalf("unable to chmod %s: %v", path, err)
			}
		},
		},
		// directory
		{"directory02", false, false, []directoryEntrySystemUseExtension{
			rockRidgePosixAttributes{mode: 0754 | os.ModeDir, linkCount: 2, uid: uid, gid: gid, length: pxLength},
			rockRidgeTimestamps{stamps: []rockRidgeTimestamp{
				rockRidgeTimestamp{timestampType: rockRidgeTimestampModify, time: now},
				rockRidgeTimestamp{timestampType: rockRidgeTimestampAccess, time: now},
				rockRidgeTimestamp{timestampType: rockRidgeTimestampAttribute, time: now},
			},
			},
			rockRidgeName{name: "directory02"},
		}, func(path string) {
			if err := os.Mkdir(path, 0754); err != nil {
				t.Fatalf("unable to create directory %s: %v", path, err)
			}
			// because of umask, must set explicitly
			if err := os.Chmod(path, 0754); err != nil {
				t.Fatalf("unable to chmod %s: %v", path, err)
			}
		},
		},
		// symlink
		{"symlink03", false, false, []directoryEntrySystemUseExtension{
			rockRidgePosixAttributes{mode: symMode | os.ModeSymlink, linkCount: 1, uid: uid, gid: gid, length: pxLength},
			rockRidgeTimestamps{stamps: []rockRidgeTimestamp{
				rockRidgeTimestamp{timestampType: rockRidgeTimestampModify, time: now},
				rockRidgeTimestamp{timestampType: rockRidgeTimestampAccess, time: now},
				rockRidgeTimestamp{timestampType: rockRidgeTimestampAttribute, time: now},
			},
			},
			rockRidgeName{name: "symlink03"},
			rockRidgeSymlink{continued: false, name: "/a/b/c/d/efgh"},
		}, func(path string) {
			if err := os.Symlink("/a/b/c/d/efgh", path); err != nil {
				t.Fatalf("unable to create symlink %s: %v", path, err)
			}
		},
		},
		// parent
		{"directoryparent", false, true, []directoryEntrySystemUseExtension{
			rockRidgePosixAttributes{mode: 0754 | os.ModeDir, linkCount: 2, uid: uid, gid: gid, length: pxLength},
			rockRidgeTimestamps{stamps: []rockRidgeTimestamp{
				rockRidgeTimestamp{timestampType: rockRidgeTimestampModify, time: now},
				rockRidgeTimestamp{timestampType: rockRidgeTimestampAccess, time: now},
				rockRidgeTimestamp{timestampType: rockRidgeTimestampAttribute, time: now},
			},
			},
		}, func(path string) {
			if err := os.Mkdir(path, 0754); err != nil {
				t.Fatalf("unable to create parent directory %s: %v", path, err)
			}
		},
		},
		// self
		{"directoryself", true, false, []directoryEntrySystemUseExtension{
			rockRidgePosixAttributes{mode: 0754 | os.ModeDir, linkCount: 2, uid: uid, gid: gid, length: pxLength},
			rockRidgeTimestamps{stamps: []rockRidgeTimestamp{
				rockRidgeTimestamp{timestampType: rockRidgeTimestampModify, time: now},
				rockRidgeTimestamp{timestampType: rockRidgeTimestampAccess, time: now},
				rockRidgeTimestamp{timestampType: rockRidgeTimestampAttribute, time: now},
			},
			},
		}, func(path string) {
			if err := os.Mkdir(path, 0754); err != nil {
				t.Fatalf("unable to create self directory %s: %v", path, err)
			}
		},
		},
	}
	for _, tt := range tests {
		// random filename
		fp := filepath.Join(dir, tt.name)
		// create the file
		tt.createFile(fp)

		// get the extensions
		ext, err := rr.GetFileExtensions(fp, tt.self, tt.parent)
		if err != nil {
			t.Fatalf("%s: Unexpected error getting extensions for %s: %v", tt.name, fp, err)
		}
		if len(ext) != len(tt.extensions) {
			t.Fatalf("%s: rock ridge extensions gave %d extensions instead of expected %d", tt.name, len(ext), len(tt.extensions))
		}
		// loop through each attribute
		for i, e := range ext {
			if stamp, ok := e.(rockRidgeTimestamps); ok {
				if !stamp.Close(tt.extensions[i]) {
					t.Errorf("%s: Mismatched extension number %d for %s, actual then expected", tt.name, i, fp)
					t.Logf("%#v\n", e)
					t.Logf("%#v\n", tt.extensions[i])
				}
			} else if !e.Equal(tt.extensions[i]) {
				t.Errorf("%s: Mismatched extension number %d for %s, actual then expected", tt.name, i, fp)
				t.Logf("%#v\n", e)
				t.Logf("%#v\n", tt.extensions[i])
			}
		}
	}
}
