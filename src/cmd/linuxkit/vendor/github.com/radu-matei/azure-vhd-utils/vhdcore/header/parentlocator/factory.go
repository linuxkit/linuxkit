package parentlocator

import (
	"fmt"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/reader"
)

// Factory type is used to create ParentLocator instance by reading one entry
// in vhd header's parent-hard-disk-locator-info collection section.
//
type Factory struct {
	vhdReader     *reader.VhdReader
	locatorOffset int64
}

// NewFactory creates a new instance of Factory, which can be used to create ParentLocator instance
// by reading one entry from the vhd header's parent-hard-disk-locator-info collection,
// locatorOffset is the offset of the entry to read, vhdReader is the reader to be used to read the entry.
//
func NewFactory(vhdReader *reader.VhdReader, locatorOffset int64) *Factory {
	return &Factory{vhdReader: vhdReader, locatorOffset: locatorOffset}
}

// Create creates a ParentLocator instance by reading one entry in vhd header's parent-hard-disk-locator-info
// collection section of the disk. This function return error if any error occurs while reading or parsing
// the parent locators table fields.
//
func (f *Factory) Create() (*ParentLocator, error) {
	locator := &ParentLocator{}
	var err error
	errDone := func() (*ParentLocator, error) {
		return nil, err
	}

	locator.PlatformCode, err = f.readPlatformCode()
	if err != nil {
		return errDone()
	}

	locator.PlatformDataSpace, err = f.readPlatformDataSpace()
	if err != nil {
		return errDone()
	}

	locator.PlatformDataLength, err = f.readPlatformDataLength()
	if err != nil {
		return errDone()
	}

	locator.Reserved, err = f.readReserved()
	if err != nil {
		return errDone()
	}

	locator.PlatformDataOffset, err = f.readPlatformDataOffset()
	if err != nil {
		return errDone()
	}

	fileLocator := make([]byte, locator.PlatformDataLength)
	_, err = f.vhdReader.ReadBytes(locator.PlatformDataOffset, fileLocator)
	if err != nil {
		err = NewParseError("ParentLocator", fmt.Errorf("Unable to resolve file locator: %v", err))
		return errDone()
	}

	locator.SetPlatformSpecificFileLocator(fileLocator)
	return locator, nil
}

// readPlatformCode reads the field that stores the platform-specific format used to encode the
// file locator in parent-hard-disk-locator-info
// This function return error if no or fewer bytes could be read. The value is stored as 4 byte
// value starting at offset 0 relative to the beginning of this parent-hard-disk-locator. This value
// is stored in big-endian format.
//
func (f *Factory) readPlatformCode() (PlatformCode, error) {
	value, err := f.vhdReader.ReadInt32(f.locatorOffset + 0)
	if err != nil {
		return PlatformCodeNone, NewParseError("PlatformCode", err)
	}
	return PlatformCode(value), nil
}

// readPlatformDataSpace reads the field that stores the number of 512-byte sectors needed to store
// the parent hard disk file locator.  This function return error if no or fewer bytes could be read.
// The value is stored as 4 byte value starting at offset 4 relative to the beginning parent-hard-disk-locator-info.
// This value is stored in big-endian format.
//
func (f *Factory) readPlatformDataSpace() (int32, error) {
	value, err := f.vhdReader.ReadInt32(f.locatorOffset + 4)
	if err != nil {
		return -1, NewParseError("PlatformDataSpace", err)
	}
	return value, nil
}

// readPlatformDataLength reads the field that stores the actual length of the parent hard disk
// locator in bytes. This function return error if no or fewer bytes could be read. The value is stored
// as 4 byte value starting at offset 8 relative to the beginning parent-hard-disk-locator-info. This value
// is stored in big-endian format.
//
func (f *Factory) readPlatformDataLength() (int32, error) {
	value, err := f.vhdReader.ReadInt32(f.locatorOffset + 8)
	if err != nil {
		return -1, NewParseError("PlatformDataLength", err)
	}
	return value, nil
}

// readReserved reads the reserved field value which is currently set to 0.
// This function return error if no or fewer bytes could be read. The value is stored as 4 byte
// value starting at offset 12 relative to the beginning parent-hard-disk-locator-info.
// This value is stored in big-endian format.
//
func (f *Factory) readReserved() (int32, error) {
	value, err := f.vhdReader.ReadInt32(f.locatorOffset + 12)
	if err != nil {
		return -1, NewParseError("Reserved", err)
	}
	return value, nil
}

// readPlatformDataOffset reads the field that stores the absolute file offset in bytes where the platform
// specific file locator data is stored. Call to this function is panic if no or fewer bytes could be read.
// The value is stored as 4 byte value starting at offset 16 relative to the beginning parent-hard-disk-locator-info.
// This value is stored in big-endian format.
//
func (f *Factory) readPlatformDataOffset() (int64, error) {
	value, err := f.vhdReader.ReadInt64(f.locatorOffset + 16)
	if err != nil {
		return -1, NewParseError("PlatformDataOffset", err)
	}
	return value, nil
}
