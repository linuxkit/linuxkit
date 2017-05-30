package header

import (
	"fmt"
	"strings"
	"time"

	"github.com/radu-matei/azure-vhd-utils/vhdcore"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/common"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/header/parentlocator"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/reader"
)

// Factory type is used to create VhdHeader instance by reading vhd header section.
//
type Factory struct {
	vhdReader    *reader.VhdReader
	headerOffset int64
}

// NewFactory creates a new instance of Factory, which can be used to create
// a VhdHeader instance by reading the header section using vhdReader.
//
func NewFactory(vhdReader *reader.VhdReader, headerOffset int64) *Factory {
	return &Factory{vhdReader: vhdReader, headerOffset: headerOffset}
}

// Create creates a Header instance by reading the header section of a expandable disk.
// This function return error if any error occurs while reading or parsing the header fields.
//
func (f *Factory) Create() (*Header, error) {
	header := &Header{}
	var err error
	errDone := func() (*Header, error) {
		return nil, err
	}

	header.Cookie, err = f.readHeaderCookie()
	if err != nil {
		return errDone()
	}

	header.DataOffset, err = f.readDataOffset()
	if err != nil {
		return errDone()
	}

	header.TableOffset, err = f.readBATOffset()
	if err != nil {
		return errDone()
	}

	header.HeaderVersion, err = f.readHeaderVersion()
	if err != nil {
		return errDone()
	}

	header.MaxTableEntries, err = f.readMaxBATEntries()
	if err != nil {
		return errDone()
	}

	header.BlockSize, err = f.readBlockSize()
	if err != nil {
		return errDone()
	}

	header.CheckSum, err = f.readCheckSum()
	if err != nil {
		return errDone()
	}

	header.ParentUniqueID, err = f.readParentUniqueID()
	if err != nil {
		return errDone()
	}

	header.ParentTimeStamp, err = f.readParentTimeStamp()
	if err != nil {
		return errDone()
	}

	header.Reserved, err = f.readReserved()
	if err != nil {
		return errDone()
	}

	header.ParentPath, err = f.readParentPath()
	if err != nil {
		return errDone()
	}

	header.ParentLocators, err = f.readParentLocators()
	if err != nil {
		return errDone()
	}

	header.RawData, err = f.readWholeHeader()
	if err != nil {
		return errDone()
	}

	return header, nil
}

// readHeaderCookie reads the vhd cookie string and returns it as an instance of VhdCookie.
// This function return error if the cookie is invalid, if no or fewer bytes could be read.
// Cookie is stored as eight-character ASCII string starting at offset 0 relative to the beginning
// of header.
//
func (f *Factory) readHeaderCookie() (*vhdcore.Cookie, error) {
	cookieData := make([]byte, 8)
	if _, err := f.vhdReader.ReadBytes(f.headerOffset+0, cookieData); err != nil {
		return nil, NewParseError("Cookie", err)
	}

	cookie := vhdcore.CreateNewVhdCookie(true, cookieData)
	if !cookie.IsValid() {
		return nil, NewParseError("Cookie", fmt.Errorf("Invalid header cookie data %v", cookieData))
	}
	return cookie, nil
}

// readDataOffset reads and return the absolute offset to the next structure in the disk. This field
// is currently unused and holds the value 0xFFFFFFFF. This function return error if no or fewer
// bytes could be read.
// This value is stored as 8 bytes value starting at offset 8 relative to the beginning of header.
// This value is stored in big-endian format.
//
func (f *Factory) readDataOffset() (int64, error) {
	value, err := f.vhdReader.ReadInt64(f.headerOffset + 8)
	if err != nil {
		return -1, NewParseError("DataOffset", err)
	}
	return value, nil
}

// readBATOffset reads and return the absolute offset to the the Block Allocation Table (BAT) in the
// disk. This function return error if no or fewer bytes could be read.
// BATOffset is stored as 8 bytes value starting at offset 16 relative to the beginning of header.
// This value is stored in big-endian format.
//
func (f *Factory) readBATOffset() (int64, error) {
	value, err := f.vhdReader.ReadInt64(f.headerOffset + 16)
	if err != nil {
		return -1, NewParseError("BATOffset", err)
	}
	return value, nil
}

// readHeaderVersion reads the value of the field the holds the major/minor version of the disk header.
// This function return error if no or fewer bytes could be read. HeaderVersion is stored as 4 bytes
// value starting at offset 24 relative to the beginning of header.
//
func (f *Factory) readHeaderVersion() (VhdHeaderVersion, error) {
	value, err := f.vhdReader.ReadUInt32(f.headerOffset + 24)
	if err != nil {
		return VhdHeaderVersionNone, NewParseError("HeaderVersion", err)
	}
	v := VhdHeaderVersion(value)
	if !v.IsSupported() {
		return VhdHeaderVersionNone,
			NewParseError("HeaderVersion", fmt.Errorf("Invalid header version %v, unsupported format", v))
	}
	return v, nil
}

// readMaxTableEntries reads and return maximum entries present in the BAT. This function return
// error if no or fewer bytes could be read.
// MaxTableEntries is stored as 4 bytes value starting at offset 28 relative to the beginning of
// header. This value is stored in big-endian format.
//
func (f *Factory) readMaxBATEntries() (uint32, error) {
	value, err := f.vhdReader.ReadUInt32(f.headerOffset + 28)
	if err != nil {
		return 0, NewParseError("MaxBATEntries", err)
	}
	return value, nil
}

// readBlockSize reads size of the 'data section' of a block, this does not include size of 'block
// bitmap section'. This function return error if no or fewer bytes could be read.
// BlockSize is stored as 4 bytes value starting at offset 32 relative to the beginning of header.
// This value is stored in big-endian format.
//
func (f *Factory) readBlockSize() (uint32, error) {
	value, err := f.vhdReader.ReadUInt32(f.headerOffset + 32)
	if err != nil {
		return 0, NewParseError("BlockSize", err)
	}
	return value, nil
}

// readCheckSum reads the field that stores basic checksum of the hard disk header.
// This function return error if no or fewer bytes could be read.
// The value is stored as 4 byte value starting at offset 36 relative to the beginning of header.
// This value is stored in big-endian format.
//
func (f *Factory) readCheckSum() (uint32, error) {
	value, err := f.vhdReader.ReadUInt32(f.headerOffset + 36)
	if err != nil {
		return 0, NewParseError("CheckSum", err)
	}
	return value, nil
}

// readParentUniqueId reads the field that stores unique ID used to identify the parent disk. This
// field is used only for differencing disk.  This is a 128-bit universally unique identifier (UUID).
// This function return error if no or fewer bytes could be read.
// The value is stored as 16 byte value starting at offset 40 relative to the beginning of header.
//
func (f *Factory) readParentUniqueID() (*common.UUID, error) {
	value, err := f.vhdReader.ReadUUID(f.headerOffset + 40)
	if err != nil {
		return nil, NewParseError("ParentUniqueId", err)
	}
	return value, nil
}

// readTimeStamp reads the field storing modification time stamp of the parent hard disk which is
// stored as the number of seconds since January 1, 2000 12:00:00 AM in UTC/GMT and return it as
// instance of time.Time. This function return error if no or fewer bytes could be read.
// TimeStamp is stored as 4 bytes value starting at offset 56 relative to the beginning of header.
// This value is stored in big-endian format.
//
func (f *Factory) readParentTimeStamp() (*time.Time, error) {
	value, err := f.vhdReader.ReadDateTime(f.headerOffset + 56)
	if err != nil {
		return nil, NewParseError("ParentTimeStamp", err)
	}
	return value, nil
}

// readReserved reads the reserved field which is not used and all set to zero. This function return
// error if no or fewer bytes could be read. Reserved is stored as 4 bytes value starting at offset
// 60 relative to the beginning of header. This value is stored in big-endian format.
//
func (f *Factory) readReserved() (uint32, error) {
	value, err := f.vhdReader.ReadUInt32(f.headerOffset + 60)
	if err != nil {
		return 0, NewParseError("Reserved", err)
	}
	return value, nil
}

// readParentPath reads the field storing parent hard disk file name. This function return error if
// no or fewer bytes could be read. ParentPath is stored in UTF-16 as big-endian format, its length is
// 512 bytes, starting at offset 64 relative to the beginning of header.
//
func (f *Factory) readParentPath() (string, error) {
	parentPath := make([]byte, 512)
	_, err := f.vhdReader.ReadBytes(f.headerOffset+64, parentPath)
	if err != nil {
		return "", NewParseError("ParentPath", err)
	}
	return strings.TrimSuffix(common.Utf16BytesToStringBE(parentPath), "\x00"), nil
}

// readParentLocators reads the collection of parent locator entries. This function return error if
// no or fewer bytes could be read. There are 8 entries, each 24 bytes, starting at offset 576 relative
// to the beginning of header.
//
func (f *Factory) readParentLocators() (parentlocator.ParentLocators, error) {
	var err error
	count := 8
	parentLocators := make(parentlocator.ParentLocators, count)
	offset := f.headerOffset + 576
	for i := 0; i < count; i++ {
		parentLocFac := parentlocator.NewFactory(f.vhdReader, offset)
		parentLocators[i], err = parentLocFac.Create()
		if err != nil {
			return nil, NewParseError("ParentLocator", err)
		}
		offset += 24
	}

	return parentLocators, nil
}

// readWholeHeader reads the entire header as a raw bytes. This function return error if the byte
// could be read.
//
func (f *Factory) readWholeHeader() ([]byte, error) {
	rawData := make([]byte, 1024)
	_, err := f.vhdReader.ReadBytes(f.headerOffset+0, rawData)
	if err != nil {
		return nil, err
	}
	return rawData, nil
}
