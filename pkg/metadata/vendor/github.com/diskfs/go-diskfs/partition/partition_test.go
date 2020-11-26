package partition_test

/*
 These tests the exported functions
 We want to do full-in tests with files
*/

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/diskfs/go-diskfs/partition"
)

func TestRead(t *testing.T) {
	tests := []struct {
		path      string
		tableType string
		err       error
	}{
		{"./mbr/testdata/mbr.img", "mbr", nil},
		{"./gpt/testdata/gpt.img", "gpt", nil},
		{"", "", fmt.Errorf("Unknown disk partition type")},
	}
	for _, tt := range tests {
		// create a blank file if we did not provide a path to a test file
		var f *os.File
		var err error
		if tt.path == "" {
			filename := "partition_test"
			f, err = ioutil.TempFile("", filename)
			// make it a 10MB file
			f.Truncate(10 * 1024 * 1024)
			defer f.Close()
			defer os.Remove(f.Name())
			if err != nil {
				t.Errorf("Failed to create tempfile %s :%v", filename, err)
				continue
			}
		} else {
			f, err = os.Open(tt.path)
			if err != nil {
				t.Errorf("Failed to open file %s :%v", tt.path, err)
				continue
			}
		}

		// be sure to close the file
		defer f.Close()

		table, err := partition.Read(f, 512, 512)

		switch {
		case (err == nil && tt.err != nil) || (err != nil && tt.err == nil) || (err != nil && tt.err != nil && !strings.HasPrefix(err.Error(), tt.err.Error())):
			t.Errorf("Read(%s): mismatched errors, actual %v expected %v", f.Name(), err, tt.err)
		case (table == nil && tt.tableType != "") || (table != nil && tt.tableType == "") || (table != nil && table.Type() != tt.tableType):
			t.Errorf("Create(%s): mismatched table, actual then expected", f.Name())
			t.Logf("%v", table.Type())
			t.Logf("%v", tt.tableType)
		}
	}

}
