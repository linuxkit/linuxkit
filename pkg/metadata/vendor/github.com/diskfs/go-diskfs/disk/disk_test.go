package disk_test

/*
 These tests the exported functions
 We want to do full-in tests with files
*/

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/partition"
	"github.com/diskfs/go-diskfs/partition/gpt"
	"github.com/diskfs/go-diskfs/partition/mbr"
)

var (
	keepTmpFiles = os.Getenv("KEEPTESTFILES") == ""
)

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

func TestGetPartitionTable(t *testing.T) {
	// this just calls partition.Read, so no need to do much more than a single test and see that it exercises it
	path := "../partition/mbr/testdata/mbr.img"
	tableType := "mbr"
	f, err := os.Open(path)
	if err != nil {
		t.Errorf("Failed to open file %s :%v", path, err)
	}

	// be sure to close the file
	defer f.Close()

	d := &disk.Disk{
		File:              f,
		LogicalBlocksize:  512,
		PhysicalBlocksize: 512,
		Writable:          false,
	}
	table, err := d.GetPartitionTable()

	switch {
	case err != nil:
		t.Errorf("Unexpected error: %v", err)
	case table == nil:
		t.Errorf("Unexpected nil table")
	case table.Type() != tableType:
		t.Errorf("mismatched table, actual then expected")
		t.Logf("%v", table.Type())
		t.Logf("%v", tableType)
	}

}

func TestPartition(t *testing.T) {
	t.Run("gpt", func(t *testing.T) {
		f, err := tmpDisk("")
		if err != nil {
			t.Fatalf("Error creating new temporary disk: %v", err)
		}
		defer f.Close()

		if keepTmpFiles {
			defer os.Remove(f.Name())
		} else {
			fmt.Println(f.Name())
		}

		fileInfo, err := f.Stat()
		if err != nil {
			t.Fatalf("Error reading info on temporary disk: %v", err)
		}

		d := &disk.Disk{
			File:              f,
			LogicalBlocksize:  512,
			PhysicalBlocksize: 512,
			Info:              fileInfo,
			Writable:          true,
			Size:              fileInfo.Size(),
		}
		// this is partition start and end in sectors, not bytes
		sectorSize := 512
		partitionStart := uint64(2048)
		// make it a 5MB partition
		partitionSize := uint64(5 * 1024 * 1024 / sectorSize)
		partitionEnd := partitionSize + partitionStart - 1
		table := &gpt.Table{
			Partitions: []*gpt.Partition{
				{Start: partitionStart, End: partitionEnd, Type: gpt.EFISystemPartition, Name: "EFI System"},
			},
			LogicalSectorSize: sectorSize,
		}
		err = d.Partition(table)
		if err != nil {
			t.Errorf("Unexpected err: %v", err)
		}
	})
	t.Run("mbr", func(t *testing.T) {
		f, err := tmpDisk("")
		if err != nil {
			t.Fatalf("Error creating new temporary disk: %v", err)
		}
		defer f.Close()

		if keepTmpFiles {
			defer os.Remove(f.Name())
		} else {
			fmt.Println(f.Name())
		}

		fileInfo, err := f.Stat()
		if err != nil {
			t.Fatalf("Error reading info on temporary disk: %v", err)
		}

		d := &disk.Disk{
			File:              f,
			LogicalBlocksize:  512,
			PhysicalBlocksize: 512,
			Info:              fileInfo,
			Writable:          true,
		}
		// this is partition start and end in sectors, not bytes
		sectorSize := 512
		partitionStart := uint32(2048)
		// make it a 5MB partition
		partitionSize := uint32(5 * 1024 * 1024 / sectorSize)
		table := &mbr.Table{
			Partitions: []*mbr.Partition{
				{Start: partitionStart, Size: partitionSize, Type: mbr.Linux},
			},
			LogicalSectorSize: sectorSize,
		}
		err = d.Partition(table)
		if err != nil {
			t.Errorf("Unexpected err: %v", err)
		}
	})
	t.Run("readonly", func(t *testing.T) {
		d := &disk.Disk{
			Writable: false,
		}
		expectedErr := fmt.Errorf("disk file or device not open for write")
		err := d.Partition(&mbr.Table{})
		if err.Error() != expectedErr.Error() {
			t.Errorf("Mismatched error, actual '%v', expected '%v'", err, expectedErr)
		}
	})
}

func TestWritePartitionContents(t *testing.T) {
	t.Run("gpt", func(t *testing.T) {
		oneMB := uint64(1024 * 1024)
		partitionStart := uint64(2048)
		partitionSize := uint64(5) * oneMB
		partitionEnd := partitionStart + partitionSize/512 - 1
		table := &gpt.Table{
			Partitions: []*gpt.Partition{
				{Start: 2048, End: partitionEnd, Type: gpt.EFISystemPartition, Name: "EFI System"},
			},
			LogicalSectorSize: 512,
		}
		tests := []struct {
			table     partition.Table
			partition int
			err       error
		}{
			// various invalid table scenarios
			{nil, 1, fmt.Errorf("cannot write contents of a partition on a disk without a partition table")},
			{nil, 0, fmt.Errorf("cannot write contents of a partition on a disk without a partition table")},
			{nil, -1, fmt.Errorf("cannot write contents of a partition on a disk without a partition table")},
			{table, 1, nil},
		}
		for _, tt := range tests {
			f, err := tmpDisk("")
			if err != nil {
				t.Fatalf("Error creating new temporary disk: %v", err)
			}
			defer f.Close()

			if keepTmpFiles {
				defer os.Remove(f.Name())
			} else {
				fmt.Println(f.Name())
			}

			fileInfo, err := f.Stat()
			if err != nil {
				t.Fatalf("Error reading info on temporary disk: %v", err)
			}

			d := &disk.Disk{
				File:              f,
				LogicalBlocksize:  512,
				PhysicalBlocksize: 512,
				Info:              fileInfo,
				Table:             tt.table,
				Writable:          true,
			}
			b := make([]byte, partitionSize, partitionSize)
			rand.Read(b)
			reader := bytes.NewReader(b)
			written, err := d.WritePartitionContents(tt.partition, reader)
			switch {
			case (err == nil && tt.err != nil) || (err != nil && tt.err == nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())):
				t.Errorf("mismatched error, actual then expected")
				t.Logf("%v", err)
				t.Logf("%v", tt.err)
			case tt.err != nil && written > 0:
				t.Errorf("Unexpectedly wrote %d bytes, expected 0", written)
			}
		}
	})
	t.Run("readonly", func(t *testing.T) {
		d := &disk.Disk{
			Writable: false,
		}
		expectedErr := fmt.Errorf("disk file or device not open for write")
		_, err := d.WritePartitionContents(0, nil)
		if err.Error() != expectedErr.Error() {
			t.Errorf("mismatched error, actual '%v' expected '%v'", err, expectedErr)
		}
	})
}

func TestReadPartitionContents(t *testing.T) {
	t.Run("gpt", func(t *testing.T) {
		partitionStart := uint64(2048)
		partitionSize := uint64(1000)
		table := &gpt.Table{
			Partitions: []*gpt.Partition{
				{Start: partitionStart, Size: partitionSize},
			},
			LogicalSectorSize: 512,
		}
		tests := []struct {
			table     partition.Table
			partition int
			err       error
		}{
			// various invalid table scenarios
			{nil, 1, fmt.Errorf("cannot read contents of a partition on a disk without a partition table")},
			{nil, 0, fmt.Errorf("cannot read contents of a partition on a disk without a partition table")},
			{nil, -1, fmt.Errorf("cannot read contents of a partition on a disk without a partition table")},
			// invalid partition number scenarios
			{table, -1, fmt.Errorf("cannot read contents of a partition without specifying a partition")},
			{table, 5, fmt.Errorf("cannot read contents of partition %d which is greater than max partition %d", 5, 1)},
			{table, 1, nil},
		}
		for _, tt := range tests {
			f, err := tmpDisk("../partition/gpt/testdata/gpt.img")
			if err != nil {
				t.Fatalf("Error creating new temporary disk: %v", err)
			}
			defer f.Close()

			if keepTmpFiles {
				defer os.Remove(f.Name())
			} else {
				fmt.Println(f.Name())
			}

			fileInfo, err := f.Stat()
			if err != nil {
				t.Fatalf("Error reading info on temporary disk: %v", err)
			}

			// get the actual content
			b2 := make([]byte, partitionSize*512, partitionSize*512)
			f.ReadAt(b2, int64(partitionStart*512))

			d := &disk.Disk{
				File:              f,
				LogicalBlocksize:  512,
				PhysicalBlocksize: 512,
				Info:              fileInfo,
				Table:             tt.table,
				Writable:          false,
			}
			var writer bytes.Buffer
			read, err := d.ReadPartitionContents(tt.partition, &writer)
			b := writer.Bytes()
			switch {
			case (err == nil && tt.err != nil) || (err != nil && tt.err == nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())):
				t.Errorf("mismatched error, actual then expected")
				t.Logf("%v", err)
				t.Logf("%v", tt.err)
			case tt.err != nil && read > 0:
				t.Errorf("Unexpectedly read %d bytes, expected 0", read)
			case tt.err == nil && bytes.Compare(b, b2) != 0:
				t.Errorf("mismatched bytes, actual then expected")
				t.Logf("len(actual) %d len(expected) %d\n", len(b), len(b2))
				//t.Log(b)
				//t.Log(b2)
			}
		}
	})
	t.Run("mbr", func(t *testing.T) {
		partitionStart := uint32(2048)
		partitionSize := uint32(1000)
		table := &mbr.Table{
			Partitions: []*mbr.Partition{
				{Start: partitionStart, Size: partitionSize},
			},
			LogicalSectorSize: 512,
		}
		tests := []struct {
			table     partition.Table
			partition int
			err       error
		}{
			// various invalid table scenarios
			{nil, 1, fmt.Errorf("cannot read contents of a partition on a disk without a partition table")},
			{nil, 0, fmt.Errorf("cannot read contents of a partition on a disk without a partition table")},
			{nil, -1, fmt.Errorf("cannot read contents of a partition on a disk without a partition table")},
			// invalid partition number scenarios
			{table, -1, fmt.Errorf("cannot read contents of a partition without specifying a partition")},
			{table, 5, fmt.Errorf("cannot read contents of partition %d which is greater than max partition %d", 5, 1)},
			{table, 1, nil},
		}
		for _, tt := range tests {
			f, err := tmpDisk("../partition/mbr/testdata/mbr.img")
			if err != nil {
				t.Fatalf("Error creating new temporary disk: %v", err)
			}
			defer f.Close()

			if keepTmpFiles {
				defer os.Remove(f.Name())
			} else {
				fmt.Println(f.Name())
			}

			fileInfo, err := f.Stat()
			if err != nil {
				t.Fatalf("Error reading info on temporary disk: %v", err)
			}

			// get the actual content
			b2 := make([]byte, partitionSize*512, partitionSize*512)
			f.ReadAt(b2, int64(partitionStart*512))

			d := &disk.Disk{
				File:              f,
				LogicalBlocksize:  512,
				PhysicalBlocksize: 512,
				Info:              fileInfo,
				Table:             tt.table,
				Writable:          false,
			}
			var writer bytes.Buffer
			read, err := d.ReadPartitionContents(tt.partition, &writer)
			b := writer.Bytes()
			switch {
			case (err == nil && tt.err != nil) || (err != nil && tt.err == nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())):
				t.Errorf("mismatched error, actual then expected")
				t.Logf("%v", err)
				t.Logf("%v", tt.err)
			case tt.err != nil && read > 0:
				t.Errorf("Unexpectedly read %d bytes, expected 0", read)
			case tt.err == nil && bytes.Compare(b, b2) != 0:
				t.Errorf("mismatched bytes, actual then expected")
				t.Logf("len(actual) %d len(expected) %d\n", len(b), len(b2))
				t.Log(b)
				t.Log(b2)
			}
		}
	})
}

func TestCreateFilesystem(t *testing.T) {
	t.Run("invalid table", func(t *testing.T) {
		f, err := tmpDisk("")
		if err != nil {
			t.Fatalf("Error creating new temporary disk: %v", err)
		}
		defer f.Close()

		if keepTmpFiles {
			defer os.Remove(f.Name())
		} else {
			fmt.Println(f.Name())
		}

		fileInfo, err := f.Stat()
		if err != nil {
			t.Fatalf("Error reading info on temporary disk: %v", err)
		}

		d := &disk.Disk{
			File:              f,
			LogicalBlocksize:  512,
			PhysicalBlocksize: 512,
			Info:              fileInfo,
			Writable:          true,
		}
		expected := fmt.Errorf("cannot create filesystem on a partition without a partition table")
		fs, err := d.CreateFilesystem(disk.FilesystemSpec{Partition: 1, FSType: filesystem.TypeFat32})
		if err == nil || err.Error() != expected.Error() {
			t.Errorf("Mismatched error: actual %v expected %v", err, expected)
		}
		if fs != nil {
			t.Errorf("Returned filesystem was unexpectedly not nil")
		}
	})
	t.Run("whole disk", func(t *testing.T) {
		f, err := tmpDisk("")
		if err != nil {
			t.Fatalf("Error creating new temporary disk: %v", err)
		}
		defer f.Close()

		if keepTmpFiles {
			defer os.Remove(f.Name())
		} else {
			fmt.Println(f.Name())
		}

		fileInfo, err := f.Stat()
		if err != nil {
			t.Fatalf("Error reading info on temporary disk: %v", err)
		}

		d := &disk.Disk{
			File:              f,
			LogicalBlocksize:  512,
			PhysicalBlocksize: 512,
			Info:              fileInfo,
			Size:              fileInfo.Size(),
			Writable:          true,
		}
		fs, err := d.CreateFilesystem(disk.FilesystemSpec{Partition: 0, FSType: filesystem.TypeFat32})
		if err != nil {
			t.Errorf("Error unexpectedly not nil:  %v", err)
		}
		if fs == nil {
			t.Errorf("Returned filesystem was unexpectedly nil")
		}
	})
	t.Run("partition", func(t *testing.T) {
		f, err := tmpDisk("../partition/mbr/testdata/mbr.img")
		if err != nil {
			t.Fatalf("Error creating new temporary disk: %v", err)
		}
		defer f.Close()

		if keepTmpFiles {
			defer os.Remove(f.Name())
		} else {
			fmt.Println(f.Name())
		}

		fileInfo, err := f.Stat()
		if err != nil {
			t.Fatalf("Error reading info on temporary disk: %v", err)
		}

		partitionStart := uint32(2048)
		partitionSize := uint32(20480)
		table := &mbr.Table{
			Partitions: []*mbr.Partition{
				{Start: partitionStart, Size: partitionSize},
			},
			LogicalSectorSize: 512,
		}
		d := &disk.Disk{
			File:              f,
			LogicalBlocksize:  512,
			PhysicalBlocksize: 512,
			Info:              fileInfo,
			Size:              fileInfo.Size(),
			Table:             table,
			Writable:          true,
		}
		fs, err := d.CreateFilesystem(disk.FilesystemSpec{Partition: 1, FSType: filesystem.TypeFat32})
		if err != nil {
			t.Errorf("Error unexpectedly not nil:  %v", err)
		}
		if fs == nil {
			t.Errorf("Returned filesystem was unexpectedly nil")
		}
	})
	t.Run("readonly", func(t *testing.T) {
		d := &disk.Disk{
			Writable: false,
		}
		expectedErr := fmt.Errorf("disk file or device not open for write")
		_, err := d.CreateFilesystem(disk.FilesystemSpec{})
		if err.Error() != expectedErr.Error() {
			t.Errorf("Mismatched error, actual '%v', expected '%v'", err, expectedErr)
		}
	})
}

func TestGetFilesystem(t *testing.T) {
	t.Run("invalid table", func(t *testing.T) {
		f, err := tmpDisk("")
		if err != nil {
			t.Fatalf("Error creating new temporary disk: %v", err)
		}
		defer f.Close()

		if keepTmpFiles {
			defer os.Remove(f.Name())
		} else {
			fmt.Println(f.Name())
		}

		fileInfo, err := f.Stat()
		if err != nil {
			t.Fatalf("Error reading info on temporary disk: %v", err)
		}

		d := &disk.Disk{
			File:              f,
			LogicalBlocksize:  512,
			PhysicalBlocksize: 512,
			Info:              fileInfo,
			Writable:          false,
		}
		expected := fmt.Errorf("cannot read filesystem on a partition without a partition table")
		fs, err := d.GetFilesystem(1)
		if err == nil || err.Error() != expected.Error() {
			t.Errorf("Mismatched error: actual %v expected %v", err, expected)
		}
		if fs != nil {
			t.Errorf("Returned filesystem was unexpectedly not nil")
		}
	})
	t.Run("whole disk", func(t *testing.T) {
		f, err := tmpDisk("../filesystem/fat32/testdata/fat32.img")
		if err != nil {
			t.Fatalf("Error creating new temporary disk: %v", err)
		}
		defer f.Close()

		if keepTmpFiles {
			defer os.Remove(f.Name())
		} else {
			fmt.Println(f.Name())
		}

		fileInfo, err := f.Stat()
		if err != nil {
			t.Fatalf("Error reading info on temporary disk: %v", err)
		}

		d := &disk.Disk{
			File:              f,
			LogicalBlocksize:  512,
			PhysicalBlocksize: 512,
			Info:              fileInfo,
			Size:              fileInfo.Size(),
			Writable:          false,
		}
		fs, err := d.GetFilesystem(0)
		if err != nil {
			t.Errorf("Error unexpectedly not nil:  %v", err)
		}
		if fs == nil {
			t.Errorf("Returned filesystem was unexpectedly nil")
		}
	})
	t.Run("partition", func(t *testing.T) {
		f, err := tmpDisk("../partition/mbr/testdata/mbr.img")
		if err != nil {
			t.Fatalf("Error creating new temporary disk: %v", err)
		}
		defer f.Close()

		if keepTmpFiles {
			defer os.Remove(f.Name())
		} else {
			fmt.Println(f.Name())
		}

		fileInfo, err := f.Stat()
		if err != nil {
			t.Fatalf("Error reading info on temporary disk: %v", err)
		}

		partitionStart := uint32(2048)
		partitionSize := uint32(20480)
		table := &mbr.Table{
			Partitions: []*mbr.Partition{
				{Start: partitionStart, Size: partitionSize},
			},
			LogicalSectorSize: 512,
		}
		d := &disk.Disk{
			File:              f,
			LogicalBlocksize:  512,
			PhysicalBlocksize: 512,
			Info:              fileInfo,
			Size:              fileInfo.Size(),
			Table:             table,
			Writable:          false,
		}
		fs, err := d.GetFilesystem(1)
		if err != nil {
			t.Errorf("Error unexpectedly not nil:  %v", err)
		}
		if fs == nil {
			t.Errorf("Returned filesystem was unexpectedly nil")
		}
	})
}
