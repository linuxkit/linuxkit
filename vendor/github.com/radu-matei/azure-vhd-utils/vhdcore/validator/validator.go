package validator

import (
	"fmt"

	"github.com/radu-matei/azure-vhd-utils/vhdcore/diskstream"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/vhdfile"
)

// oneTB is one TeraByte
//
const oneTB int64 = 1024 * 1024 * 1024 * 1024

// ValidateVhd returns error if the vhdPath refer to invalid vhd.
//
func ValidateVhd(vhdPath string) error {
	vFactory := &vhdFile.FileFactory{}
	_, err := vFactory.Create(vhdPath)
	if err != nil {
		return fmt.Errorf("%s is not a valid VHD: %v", vhdPath, err)
	}
	return nil
}

// ValidateVhdSize returns error if size of the vhd referenced by vhdPath is more than
// the maximum allowed size (1TB)
//
func ValidateVhdSize(vhdPath string) error {
	stream, _ := diskstream.CreateNewDiskStream(vhdPath)
	if stream.GetSize() > oneTB {
		return fmt.Errorf("VHD size is too large ('%d'), maximum allowed size is '%d'", stream.GetSize(), oneTB)
	}
	return nil
}
