package footer

import (
	"fmt"
	"time"

	"github.com/radu-matei/azure-vhd-utils/vhdcore"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/common"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/reader"
)

// Factory type is used to create Footer instance by reading vhd footer section.
//
type Factory struct {
	vhdReader    *reader.VhdReader
	footerOffset int64
}

// NewFactory creates a new instance of Factory, which can be used to create a Footer
// instance by reading the footer section using VhdReader.
//
func NewFactory(vhdReader *reader.VhdReader) *Factory {
	return &Factory{vhdReader: vhdReader, footerOffset: vhdReader.Size - vhdcore.VhdFooterSize}
}

// Create creates a Footer instance by reading the footer section of the disk.
// This function return error if any error occurs while reading or parsing the footer fields.
//
func (f *Factory) Create() (*Footer, error) {
	footer := &Footer{}
	var err error
	errDone := func() (*Footer, error) {
		return nil, err
	}

	footer.Cookie, err = f.readVhdCookie()
	if err != nil {
		return errDone()
	}

	footer.Features, err = f.readFeatures()
	if err != nil {
		return errDone()
	}

	footer.FileFormatVersion, err = f.readFileFormatVersion()
	if err != nil {
		return errDone()
	}

	footer.HeaderOffset, err = f.readHeaderOffset()
	if err != nil {
		return errDone()
	}

	footer.TimeStamp, err = f.readTimeStamp()
	if err != nil {
		return errDone()
	}
	footer.CreatorApplication, err = f.readCreatorApplication()
	if err != nil {
		return errDone()
	}

	footer.CreatorVersion, err = f.readCreatorVersion()
	if err != nil {
		return errDone()
	}

	footer.CreatorHostOsType, err = f.readCreatorHostOsType()
	if err != nil {
		return errDone()
	}

	footer.PhysicalSize, err = f.readPhysicalSize()
	if err != nil {
		return errDone()
	}

	footer.VirtualSize, err = f.readVirtualSize()
	if err != nil {
		return errDone()
	}

	footer.DiskGeometry, err = f.readDiskGeometry()
	if err != nil {
		return errDone()
	}

	footer.DiskType, err = f.readDiskType()
	if err != nil {
		return errDone()
	}

	footer.CheckSum, err = f.readCheckSum()
	if err != nil {
		return errDone()
	}

	footer.UniqueID, err = f.readUniqueID()
	if err != nil {
		return errDone()
	}

	footer.SavedState, err = f.readSavedState()
	if err != nil {
		return errDone()
	}

	footer.Reserved, err = f.readReserved()
	if err != nil {
		return errDone()
	}

	footer.RawData, err = f.readWholeFooter()
	if err != nil {
		return errDone()
	}

	return footer, nil
}

// readVhdCookie reads the vhd cookie string and returns it as an instance of VhdCookie.
// This function returns error if the cookie is invalid, if no or fewer bytes could be
// read. Cookie is stored as eight-character ASCII string starting at offset 0 relative
// to the beginning of footer.
//
func (f *Factory) readVhdCookie() (*vhdcore.Cookie, error) {
	cookieData := make([]byte, 8)
	if _, err := f.vhdReader.ReadBytes(f.footerOffset+0, cookieData); err != nil {
		return nil, NewParseError("Cookie", err)
	}

	cookie := vhdcore.CreateNewVhdCookie(false, cookieData)
	if !cookie.IsValid() {
		return nil, NewParseError("Cookie", fmt.Errorf("Invalid footer cookie data %v", cookieData))
	}
	return cookie, nil
}

// readFeatures reads and return the feature field. This function return error if no or
// fewer bytes could be read.
// Feature is stored as 4 bytes value starting at offset 8 relative to the beginning of
// footer.
//
func (f *Factory) readFeatures() (VhdFeature, error) {
	value, err := f.vhdReader.ReadUInt32(f.footerOffset + 8)
	if err != nil {
		return VhdFeatureNoFeaturesEnabled, NewParseError("Features", err)
	}
	return VhdFeature(value), nil
}

// readFileFormatVersion reads and return the VhdFileFormatVersion field from the footer.
// This function is return error if no or fewer bytes could be read.
// VhdFileFormatVersion is stored as 4 bytes value starting at offset 12 relative to the
// beginning of footer.
//
func (f *Factory) readFileFormatVersion() (VhdFileFormatVersion, error) {
	value, err := f.vhdReader.ReadUInt32(f.footerOffset + 12)
	if err != nil {
		return VhdFileFormatVersionNone, NewParseError("FileFormatVersion", err)
	}
	return VhdFileFormatVersion(value), nil
}

// readHeaderOffset reads and return the absolute offset to the header structure.
// This function return error if no or fewer bytes could be read.
// Header offset is stored as 8 bytes value starting at offset 16 relative to the beginning
// of footer. This value is stored in big-endian format.
//
func (f *Factory) readHeaderOffset() (int64, error) {
	value, err := f.vhdReader.ReadInt64(f.footerOffset + 16)
	if err != nil {
		return -1, NewParseError("HeaderOffset", err)
	}
	return value, nil
}

// readTimeStamp reads the creation time of the disk which is stored as the number of seconds
// since January 1, 2000 12:00:00 AM in UTC/GMT and return it as instance of time.Time.
// This function return error if no or fewer bytes could be read.
// TimeStamp is stored as 4 bytes value starting at offset 24 relative to the beginning
// of footer. This value is stored in big-endian format.
//
func (f *Factory) readTimeStamp() (*time.Time, error) {
	value, err := f.vhdReader.ReadDateTime(f.footerOffset + 24)
	if err != nil {
		return nil, NewParseError("TimeStamp", err)
	}
	return value, nil
}

// readCreatorApplication reads the value of the field containing identity of the application
// used to create the disk. The field is a left-justified text field. It uses a single-byte
// character set. This function return error if no or fewer bytes could be read.
// Identifier is stored as 4 bytes value starting at offset 28 relative to the beginning
// of footer.
//
func (f *Factory) readCreatorApplication() (string, error) {
	creatorApp := make([]byte, 4)
	_, err := f.vhdReader.ReadBytes(f.footerOffset+28, creatorApp)
	if err != nil {
		return "", NewParseError("CreatorApplication", err)
	}
	return string(creatorApp), nil
}

// readCreatorVersion reads the value of the field the holds the major/minor version of the
// application that created the hard disk image. This function return error if no or fewer
// bytes could be read.
// Version is stored as 4 bytes value starting at offset 32 relative to the beginning
// of footer.
//
func (f *Factory) readCreatorVersion() (VhdCreatorVersion, error) {
	value, err := f.vhdReader.ReadUInt32(f.footerOffset + 32)
	if err != nil {
		return VhdCreatorVersionNone, NewParseError("CreatorVersion", err)
	}
	return VhdCreatorVersion(value), nil
}

// readCreatorHostOsType reads the value of the field that stores the type of host operating
// system this disk image is created on. Call to this function return error if no or fewer
// bytes could be read.
// Version is stored as 4 bytes value starting at offset 36 relative to the beginning
// of footer.
//
func (f *Factory) readCreatorHostOsType() (HostOsType, error) {
	value, err := f.vhdReader.ReadUInt32(f.footerOffset + 36)
	if err != nil {
		return HostOsTypeNone, NewParseError("CreatorHostOsType", err)
	}
	return HostOsType(value), nil
}

// readPhysicalSize reads the size of the hard disk in bytes, from the perspective of the
// virtual machine, at creation time. This field is for informational purposes.
// This function return error if no or fewer bytes could be read.
// PhysicalSize is stored as 8 bytes value starting at offset 40 relative to the
// beginning of footer. This size does not include the size consumed by vhd metadata such as
// header, footer BAT, block's bitmap
// This value is stored in big-endian format.
//
func (f *Factory) readPhysicalSize() (int64, error) {
	value, err := f.vhdReader.ReadInt64(f.footerOffset + 40)
	if err != nil {
		return -1, NewParseError("PhysicalSize", err)
	}
	return value, nil
}

// readVirtualSize reads the size of the he current size of the hard disk, in bytes, from the
// perspective of the virtual machine. This value is same as the PhysicalSize when the hard
// disk is created. This value can change depending on whether the hard disk is expanded
// This function return error if no or fewer bytes could be read.
// VirtualSize is stored as 8 bytes value starting at offset 48 relative to the
// beginning of footer. This size does not include the size consumed by vhd metadata such as
// header, footer BAT, block's bitmap
// This value is stored in big-endian format.
//
func (f *Factory) readVirtualSize() (int64, error) {
	value, err := f.vhdReader.ReadInt64(f.footerOffset + 48)
	if err != nil {
		return -1, NewParseError("VirtualSize", err)
	}
	return value, nil
}

// readDiskGeometry reads the 4 byte value that stores the cylinder, heads, and sectors per
// track value for the hard disk. This function return error if no or fewer bytes could
// be read. The value is stored starting starting at offset 56 relative to the beginning of
// footer. This value is stored in big-endian format.
//
func (f *Factory) readDiskGeometry() (*DiskGeometry, error) {
	diskGeometry := &DiskGeometry{}
	cylinder, err := f.vhdReader.ReadUInt16(f.footerOffset + 56 + 0)
	if err != nil {
		return nil, NewParseError("DiskGeometry::Cylinder", err)
	}
	diskGeometry.Cylinder = cylinder
	heads, err := f.vhdReader.ReadByte(f.footerOffset + 56 + 2)
	if err != nil {
		return nil, NewParseError("DiskGeometry::Heads", err)
	}
	diskGeometry.Heads = heads
	sectors, err := f.vhdReader.ReadByte(f.footerOffset + 56 + 3)
	if err != nil {
		return nil, NewParseError("DiskGeometry::Sectors", err)
	}
	diskGeometry.Sectors = sectors
	return diskGeometry, nil
}

// readDiskType reads the field stores type of the disk (fixed, differencing, dynamic)
// This function return error if no or fewer bytes could be read.
// The value is stored as 4 byte value starting at offset 60 relative to the beginning
// of footer. This value is stored in big-endian format.
//
func (f *Factory) readDiskType() (DiskType, error) {
	value, err := f.vhdReader.ReadUInt32(f.footerOffset + 60)
	if err != nil {
		return DiskTypeNone, NewParseError("DiskType", err)
	}
	return DiskType(value), nil
}

// readCheckSum reads the field that stores basic checksum of the hard disk footer.
// This function return error if no or fewer bytes could be read.
// The value is stored as 4 byte value starting at offset 64 relative to the beginning
// of footer. This value is stored in big-endian format.
//
func (f *Factory) readCheckSum() (uint32, error) {
	value, err := f.vhdReader.ReadUInt32(f.footerOffset + 64)
	if err != nil {
		return 0, NewParseError("CheckSum", err)
	}
	return value, nil
}

// readUniqueId reads the field that stores unique ID used to identify the hard disk.
// This is a 128-bit universally unique identifier (UUID)
// This function return error if no or fewer bytes could be read.
// The value is stored as 16 byte value starting at offset 68 relative to the beginning
// of footer.
//
func (f *Factory) readUniqueID() (*common.UUID, error) {
	value, err := f.vhdReader.ReadUUID(f.footerOffset + 68)
	if err != nil {
		return nil, NewParseError("UniqueId", err)
	}
	return value, nil
}

// readSavedState reads the flag indicating whether the system is in saved state.
// This function return error if the byte could be read.
// The value is stored as 1 byte value starting at offset 84 relative to the beginning
// of footer.
//
func (f *Factory) readSavedState() (bool, error) {
	value, err := f.vhdReader.ReadBoolean(f.footerOffset + 84)
	if err != nil {
		return false, NewParseError("SavedState", err)
	}
	return value, err
}

// readReserved reads the reserved field which currently contains zeroes.
// This function return error if the byte could be read.
// It is 427 bytes in size starting at offset 85 relative to the beginning
// of footer.
//
func (f *Factory) readReserved() ([]byte, error) {
	reserved := make([]byte, 427)
	_, err := f.vhdReader.ReadBytes(f.footerOffset+85, reserved)
	if err != nil {
		return nil, NewParseError("Reserved", err)
	}
	return reserved, nil
}

// readWholeFooter reads the entire footer as a raw bytes. This function return
// error if the byte could be read.
//
func (f *Factory) readWholeFooter() ([]byte, error) {
	rawData := make([]byte, 512)
	_, err := f.vhdReader.ReadBytes(f.footerOffset+0, rawData)
	if err != nil {
		return nil, err
	}
	return rawData, nil
}
