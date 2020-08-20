package mbr

import (
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

const (
	mbrFile = "./testdata/mbr.img"
)

func GetValidTable() *Table {
	table := &Table{
		LogicalSectorSize:  512,
		PhysicalSectorSize: 512,
	}
	parts := []*Partition{
		{
			Bootable:      false,
			StartHead:     0x20,
			StartSector:   0x21,
			StartCylinder: 0x00,
			Type:          Linux,
			EndHead:       0x31,
			EndSector:     0x18,
			EndCylinder:   0x00,
			Start:         partitionStart,
			Size:          partitionSize,
		},
	}
	// add 127 Unused partitions to the table
	for i := 1; i < 4; i++ {
		parts = append(parts, &Partition{Type: Empty})
	}
	table.Partitions = parts
	return table
}

func TestTableFromBytes(t *testing.T) {
	t.Run("Short byte slice", func(t *testing.T) {
		b := make([]byte, 512-1, 512-1)
		rand.Read(b)
		table, err := tableFromBytes(b, 512, 512)
		if table != nil {
			t.Error("should return nil table")
		}
		if err == nil {
			t.Error("should not return nil error")
		}
		expected := fmt.Sprintf("Data for partition was %d bytes", len(b))
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}
	})
	t.Run("Invalid MBR Signature", func(t *testing.T) {
		b, err := ioutil.ReadFile(mbrFile)
		if err != nil {
			t.Fatalf("Unable to read test fixture file %s: %v", mbrFile, err)
		}
		b[511] = 0x00
		table, err := tableFromBytes(b[:512], 512, 512)
		if table != nil {
			t.Error("should return nil table")
		}
		if err == nil {
			t.Error("should not return nil error")
		}
		expected := fmt.Sprintf("Invalid MBR Signature")
		if !strings.HasPrefix(err.Error(), expected) {
			t.Errorf("Error type %s instead of expected %s", err.Error(), expected)
		}
	})
	t.Run("Valid table", func(t *testing.T) {
		b, err := ioutil.ReadFile(mbrFile)
		if err != nil {
			t.Fatalf("Unable to read test fixture file %s: %v", mbrFile, err)
		}
		table, err := tableFromBytes(b[:512], 512, 512)
		if table == nil {
			t.Error("should not return nil table")
		}
		if err != nil {
			t.Errorf("returned non-nil error: %v", err)
		}
		expected := GetValidTable()
		if table == nil && expected != nil || !table.Equal(expected) {
			t.Errorf("actual table was %v instead of expected %v", table, expected)
		}
	})
}
