package fat32

import (
	"os"
	"testing"
	"time"
)

var (
	now = time.Now()
	f   = &FileInfo{
		modTime:   now,
		mode:      os.ModePerm,
		name:      "foobarlomngname.abcdef",
		shortName: "FOOBAR~1.ABC",
		size:      1567,
		isDir:     false,
	}
)

func TestFileInfoIsDir(t *testing.T) {
	isDir := f.IsDir()
	if isDir != f.isDir {
		t.Errorf("IsDir() returned %t instead of expected %t", isDir, f.isDir)
	}
}

func TestFileInfoModTime(t *testing.T) {
	modtime := f.ModTime()
	if modtime != f.modTime {
		t.Errorf("ModTime() returned %v instead of expected %v", modtime, f.modTime)
	}
}

func TestFileInfoMode(t *testing.T) {
	mode := f.Mode()
	if mode != f.mode {
		t.Errorf("Mode() returned %v instead of expected %v", mode, f.mode)
	}
}

func TestFileInfoName(t *testing.T) {
	name := f.Name()
	if name != f.name {
		t.Errorf("Name() returned %s instead of expected %s", name, f.name)
	}
}

func TestFileInfoShortName(t *testing.T) {
	name := f.ShortName()
	if name != f.shortName {
		t.Errorf("ShortName() returned %s instead of expected %s", name, f.shortName)
	}
}

func TestFileInfoSize(t *testing.T) {
	s := f.Size()
	if s != f.size {
		t.Errorf("Size() returned %d instead of expected %d", s, f.size)
	}
}

func TestFileInfoSys(t *testing.T) {
	s := f.Sys()
	if s != nil {
		t.Errorf("Sys() returned non-nil: %v", s)
	}

}
