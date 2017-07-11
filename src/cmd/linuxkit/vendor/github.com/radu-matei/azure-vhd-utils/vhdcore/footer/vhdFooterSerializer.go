package footer

import (
	"github.com/radu-matei/azure-vhd-utils/vhdcore"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/writer"
)

// SerializeFooter returns the given VhdFooter instance as byte slice of length 512 bytes.
//
func SerializeFooter(footer *Footer) []byte {
	buffer := make([]byte, vhdcore.VhdFooterSize)
	writer := writer.NewVhdWriterFromByteSlice(buffer)

	writer.WriteBytes(0, footer.Cookie.Data)
	writer.WriteUInt32(8, uint32(footer.Features))
	writer.WriteUInt32(12, uint32(footer.FileFormatVersion))
	writer.WriteInt64(16, footer.HeaderOffset)
	writer.WriteTimeStamp(24, footer.TimeStamp)
	creatorApp := make([]byte, 4)
	copy(creatorApp, footer.CreatorApplication)
	writer.WriteBytes(28, creatorApp)
	writer.WriteUInt32(32, uint32(footer.CreatorVersion))
	writer.WriteUInt32(36, uint32(footer.CreatorHostOsType))
	writer.WriteInt64(40, footer.PhysicalSize)
	writer.WriteInt64(48, footer.VirtualSize)
	// + DiskGeometry
	writer.WriteUInt16(56, footer.DiskGeometry.Cylinder)
	writer.WriteByte(58, footer.DiskGeometry.Heads)
	writer.WriteByte(59, footer.DiskGeometry.Sectors)
	// - DiskGeometry
	writer.WriteUInt32(60, uint32(footer.DiskType))
	writer.WriteBytes(68, footer.UniqueID.ToByteSlice())
	writer.WriteBoolean(84, footer.SavedState)
	writer.WriteBytes(85, footer.Reserved)
	// + Checksum
	//
	// Checksum is one’s complement of the sum of all the bytes in the footer without the
	// checksum field.
	checkSum := uint32(0)
	for i := int(0); i < int(vhdcore.VhdFooterSize); i++ {
		if i < vhdcore.VhdFooterChecksumOffset || i >= vhdcore.VhdFooterChecksumOffset+4 {
			checkSum += uint32(buffer[i])
		}
	}

	writer.WriteUInt32(64, ^checkSum)
	// - Checksum

	return buffer
}
