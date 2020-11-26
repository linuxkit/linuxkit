package iso9660

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestCopyFileData(t *testing.T) {
	// create an empty file as source
	from, err := ioutil.TempFile("", "iso9660_finalize_test_from")
	if err != nil {
		t.Fatal("error creating 'from' tmpfile", err)
	}

	defer os.Remove(from.Name()) // clean up

	// create some random data
	// 100KB is fine
	blen := 1024 * 100
	b := make([]byte, blen)
	_, err = rand.Read(b)
	if err != nil {
		t.Fatal("error getting random bytes:", err)
	}

	if _, err = from.Write(b); err != nil {
		t.Fatal("Error writing random bytes to 'from' tmpfile", err)
	}

	// create a target file
	to, err := ioutil.TempFile("", "iso9660_finalize_test_to")
	if err != nil {
		t.Fatal("error creating 'to' tmpfile", err)
	}
	defer os.Remove(from.Name()) // clean up

	copied, err := copyFileData(from, to, 0, 0)
	if err != nil {
		t.Fatal("error copying data from/to", err)
	}
	expected := blen
	if copied != expected {
		t.Fatalf("copied %d bytes instead of expected %d", copied, blen)
	}

	_, err = to.Seek(0, os.SEEK_SET)
	if err != nil {
		t.Fatal("Error resetting 'to' file", err)
	}
	c := make([]byte, blen)
	if _, err = to.Read(c); err != nil {
		t.Fatal("Error reading 'to' tmpfile", err)
	}

	if bytes.Compare(b, c) != 0 {
		t.Fatalf("Mismatched content between 'from' and 'to' files, 'from' then 'to'\n%#x\n%#x", b, c)
	}

	if err := from.Close(); err != nil {
		t.Fatal("error closing 'from' tmpfile", err)
	}
	if err := to.Close(); err != nil {
		t.Fatal("error closing 'to' tmpfile", err)
	}
}

func TestSortFinalizeFileInfoPathTable(t *testing.T) {
	tests := []struct {
		left  *finalizeFileInfo
		right *finalizeFileInfo
		less  bool
	}{
		{&finalizeFileInfo{parent: nil, depth: 3, name: "ABC", shortname: "ABC"}, &finalizeFileInfo{parent: nil, depth: 3, name: "XYZ", shortname: "DEF"}, true},                                                                                                                      // same parent, should sort by name
		{&finalizeFileInfo{parent: nil, depth: 3, name: "XYZ", shortname: "XYZ"}, &finalizeFileInfo{parent: nil, depth: 3, name: "ABC", shortname: "ABC"}, false},                                                                                                                     // same parent, should sort by name
		{&finalizeFileInfo{parent: &finalizeFileInfo{}, depth: 3, name: "ABC", shortname: "ABC"}, &finalizeFileInfo{parent: &finalizeFileInfo{}, depth: 4, name: "ABC", shortname: "ABC"}, true},                                                                                      // different parents, should sort by depth
		{&finalizeFileInfo{parent: &finalizeFileInfo{}, depth: 4, name: "ABC", shortname: "ABC"}, &finalizeFileInfo{parent: &finalizeFileInfo{}, depth: 3, name: "ABC", shortname: "ABC"}, false},                                                                                     // different parents, should sort by depth
		{&finalizeFileInfo{parent: &finalizeFileInfo{parent: nil, name: "AAA", shortname: "AAA"}, depth: 3, name: "ABC", shortname: "ABC"}, &finalizeFileInfo{parent: &finalizeFileInfo{parent: nil, name: "ZZZ", shortname: "ZZZ"}, depth: 3, name: "ABC", shortname: "ABC"}, true},  // different parents, same depth, should sort by parent
		{&finalizeFileInfo{parent: &finalizeFileInfo{parent: nil, name: "ZZZ", shortname: "ZZZ"}, depth: 3, name: "ABC", shortname: "ABC"}, &finalizeFileInfo{parent: &finalizeFileInfo{parent: nil, name: "AAA", shortname: "AAA"}, depth: 3, name: "ABC", shortname: "ABC"}, false}, // different parents, same depth, should sort by parent
	}
	for i, tt := range tests {
		result := sortFinalizeFileInfoPathTable(tt.left, tt.right)
		if result != tt.less {
			t.Errorf("%d: got %v expected %v", i, result, tt.less)
		}
	}
}

func TestCreatePathTable(t *testing.T) {
	// uses name, parent, location
	root := &finalizeFileInfo{name: "", location: 16, isDir: true}
	root.parent = root
	fives := &finalizeFileInfo{name: "FIVES", shortname: "FIVES", location: 22, parent: root, isDir: true}
	tens := &finalizeFileInfo{name: "TENLETTERS", shortname: "TENLETTERS", location: 17, parent: root, isDir: true}
	subFives := &finalizeFileInfo{name: "SUBOFFIVES12", shortname: "SUBOFFIVES12", location: 45, parent: fives, isDir: true}
	subTen := &finalizeFileInfo{name: "SHORT", shortname: "SHORT", location: 32, parent: tens, isDir: true}
	input := []*finalizeFileInfo{subTen, fives, root, tens, subFives}
	expected := &pathTable{
		records: []*pathTableEntry{
			{nameSize: 0, size: 8, extAttrLength: 0, location: 16, parentIndex: 1, dirname: ""},
			{nameSize: 5, size: 14, extAttrLength: 0, location: 22, parentIndex: 1, dirname: "FIVES"},
			{nameSize: 10, size: 18, extAttrLength: 0, location: 17, parentIndex: 1, dirname: "TENLETTERS"},
			{nameSize: 12, size: 20, extAttrLength: 0, location: 45, parentIndex: 2, dirname: "SUBOFFIVES12"},
			{nameSize: 5, size: 14, extAttrLength: 0, location: 32, parentIndex: 3, dirname: "SHORT"},
		},
	}
	pt := createPathTable(input)
	// createPathTable(fi []*finalizeFileInfo) *pathTable
	if !pt.equal(expected) {
		t.Errorf("pathTable not as expected, actual then expected\n%#v\n%#v", pt.names(), expected.names())
	}
}

func TestCollapseAndSortChildren(t *testing.T) {
	// we need to build a file tree, and then see that the results are correct and in order
	// the algorithm uses the following properties of finalizeFileInfo:
	//   isDir, children, name, shortname
	// the algorithm is supposed to sort by name in each node, and depth first
	root := &finalizeFileInfo{name: ".", depth: 1, isDir: true}
	children := []*finalizeFileInfo{
		{name: "ABC", shortname: "ABC", isDir: false},
		{name: "DEF", shortname: "DEF", isDir: true},
		{name: "TWODEEP", shortname: "TWODEEP", isDir: true, children: []*finalizeFileInfo{
			{name: "TWODEEP1", shortname: "TWODEEP1", isDir: false},
			{name: "TWODEEP3", shortname: "TWODEEP3", isDir: true, children: []*finalizeFileInfo{
				{name: "TWODEEP33", shortname: "TWODEEP33", isDir: false},
				{name: "TWODEEP31", shortname: "TWODEEP31", isDir: false},
				{name: "TWODEEP32", shortname: "TWODEEP32", isDir: true},
			}},
			{name: "TWODEEP2", shortname: "TWODEEP2", isDir: false},
		}},
		{name: "README.MD", shortname: "README.MD", isDir: false},
		{name: "ONEDEEP", shortname: "ONEDEEP", isDir: true, children: []*finalizeFileInfo{
			{name: "ONEDEEP1", shortname: "ONEDEEP1", isDir: false},
			{name: "ONEDEEP3", shortname: "ONEDEEP3", isDir: false},
			{name: "ONEDEEP2", shortname: "ONEDEEP2", isDir: false},
		}},
	}
	expectedDirs := []*finalizeFileInfo{
		children[1], children[4], children[2], children[2].children[1], children[2].children[1].children[2],
	}
	expectedFiles := []*finalizeFileInfo{
		children[0], children[3], children[4].children[0], children[4].children[2], children[4].children[1],
		children[2].children[0], children[2].children[2], children[2].children[1].children[1], children[2].children[1].children[0],
	}
	root.children = children
	root.addProperties(1)
	dirs, files := root.collapseAndSortChildren()
	dirsMatch := true
	if len(dirs) != len(expectedDirs) {
		dirsMatch = false
	}
	filesMatch := true
	if len(files) != len(expectedFiles) {
		filesMatch = false
	}
	if dirsMatch {
		for i, d := range dirs {
			if d != expectedDirs[i] {
				dirsMatch = false
				break
			}
		}
	}
	if filesMatch {
		for i, f := range files {
			if f != expectedFiles[i] {
				filesMatch = false
				break
			}
		}
	}
	if !dirsMatch {
		t.Error("mismatched dirs, actual then expected")
		output := ""
		for _, e := range dirs {
			output = fmt.Sprintf("%s{%s,%v},", output, e.name, e.isDir)
		}
		t.Log(output)
		output = ""
		for _, e := range expectedDirs {
			output = fmt.Sprintf("%s{%s,%v},", output, e.name, e.isDir)
		}
		t.Log(output)
	}
	if !filesMatch {
		t.Error("mismatched files, actual then expected")
		output := ""
		for _, e := range files {
			output = fmt.Sprintf("%s{%s,%v},", output, e.name, e.isDir)
		}
		t.Log(output)
		output = ""
		for _, e := range expectedFiles {
			output = fmt.Sprintf("%s{%s,%v},", output, e.name, e.isDir)
		}
		t.Log(output)
	}
}
