package diskfs_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
)

const oneMB = 10 * 1024 * 1024

func tmpDisk(source string) (*os.File, error) {
	filename := "disk_test"
	f, err := ioutil.TempFile("", filename)
	if err != nil {
		return nil, fmt.Errorf("Failed to create tempfile %s :%v", filename, err)
	}

	// either copy the contents of the source file over, or make a file of appropriate size
	if source == "" {
		// make it a 10MB file
		f.Truncate(10 * 1024 * 1024)
	} else {
		b, err := ioutil.ReadFile(source)
		if err != nil {
			return nil, fmt.Errorf("Failed to read contents of %s: %v", source, err)
		}
		written, err := f.Write(b)
		if err != nil {
			return nil, fmt.Errorf("Failed to write contents of %s to %s: %v", source, filename, err)
		}
		if written != len(b) {
			return nil, fmt.Errorf("Wrote only %d bytes of %s to %s instead of %d", written, source, filename, len(b))
		}
	}

	return f, nil
}

func TestOpen(t *testing.T) {
	f, err := tmpDisk("./partition/mbr/testdata/mbr.img")
	if err != nil {
		t.Fatalf("Error creating new temporary disk: %v", err)
	}
	defer f.Close()
	path := f.Name()
	defer os.Remove(path)
	fileInfo, err := f.Stat()
	if err != nil {
		t.Fatalf("Unable to stat temporary file %s: %v", path, err)
	}
	size := fileInfo.Size()

	tests := []struct {
		path string
		disk *disk.Disk
		err  error
	}{
		{"", nil, fmt.Errorf("must pass device name")},
		{"/tmp/foo/bar/232323/23/2322/disk.img", nil, fmt.Errorf("")},
		{path, &disk.Disk{Type: disk.File, LogicalBlocksize: 512, PhysicalBlocksize: 512, Size: size}, nil},
	}

	for _, tt := range tests {
		d, err := diskfs.Open(tt.path)
		msg := fmt.Sprintf("Open(%s)", tt.path)
		switch {
		case (err == nil && tt.err != nil) || (err != nil && tt.err == nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())):
			t.Errorf("%s: mismatched errors, actual %v expected %v", msg, err, tt.err)
		case (d == nil && tt.disk != nil) || (d != nil && tt.disk == nil):
			t.Errorf("%s: mismatched disk, actual %v expected %v", msg, d, tt.disk)
		case d != nil && (d.LogicalBlocksize != tt.disk.LogicalBlocksize || d.PhysicalBlocksize != tt.disk.PhysicalBlocksize || d.Size != tt.disk.Size || d.Type != tt.disk.Type):
			t.Errorf("%s: mismatched disk, actual then expected", msg)
			t.Logf("%v", d)
			t.Logf("%v", tt.disk)
		}
	}

	for i, tt := range tests {
		d, err := diskfs.OpenWithMode(tt.path, diskfs.ReadOnly)
		msg := fmt.Sprintf("%d: Open(%s)", i, tt.path)
		switch {
		case (err == nil && tt.err != nil) || (err != nil && tt.err == nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())):
			t.Errorf("%s: mismatched errors, actual %v expected %v", msg, err, tt.err)
		case (d == nil && tt.disk != nil) || (d != nil && tt.disk == nil):
			t.Errorf("%s: mismatched disk, actual %v expected %v", msg, d, tt.disk)
		case d != nil && (d.LogicalBlocksize != tt.disk.LogicalBlocksize || d.PhysicalBlocksize != tt.disk.PhysicalBlocksize || d.Size != tt.disk.Size || d.Type != tt.disk.Type):
			t.Errorf("%s: mismatched disk, actual then expected", msg)
			t.Logf("%v", d)
			t.Logf("%v", tt.disk)
		}
	}
}

func TestCreate(t *testing.T) {
	tests := []struct {
		path   string
		size   int64
		format diskfs.Format
		disk   *disk.Disk
		err    error
	}{
		{"", 10 * oneMB, diskfs.Raw, nil, fmt.Errorf("must pass device name")},
		{"/tmp/disk.img", 0, diskfs.Raw, nil, fmt.Errorf("must pass valid device size to create")},
		{"/tmp/disk.img", -1, diskfs.Raw, nil, fmt.Errorf("must pass valid device size to create")},
		{"/tmp/foo/bar/232323/23/2322/disk.img", 10 * oneMB, diskfs.Raw, nil, fmt.Errorf("Could not create device")},
		{"/tmp/disk.img", 10 * oneMB, diskfs.Raw, &disk.Disk{LogicalBlocksize: 512, PhysicalBlocksize: 512, Size: 10 * oneMB, Type: disk.File}, nil},
	}

	for i, tt := range tests {
		disk, err := diskfs.Create(tt.path, tt.size, tt.format)
		msg := fmt.Sprintf("%d: Create(%s, %d, %v)", i, tt.path, tt.size, tt.format)
		switch {
		case (err == nil && tt.err != nil) || (err != nil && tt.err == nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())):
			t.Errorf("%s: mismatched errors, actual %v expected %v", msg, err, tt.err)
		case (disk == nil && tt.disk != nil) || (disk != nil && tt.disk == nil):
			t.Errorf("%s: mismatched disk, actual %v expected %v", msg, disk, tt.disk)
		case disk != nil && (disk.LogicalBlocksize != tt.disk.LogicalBlocksize || disk.PhysicalBlocksize != tt.disk.PhysicalBlocksize || disk.Size != tt.disk.Size || disk.Type != tt.disk.Type):
			t.Errorf("%s: mismatched disk, actual then expected", msg)
			t.Logf("%#v", disk)
			t.Logf("%#v", tt.disk)
		}
		if disk != nil {
			os.Remove(tt.path)
		}
	}
}
