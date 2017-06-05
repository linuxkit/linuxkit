package header

import (
	"time"

	"github.com/radu-matei/azure-vhd-utils/vhdcore"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/common"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/header/parentlocator"
)

// Header represents the header of the vhd, size of the header is 1024 bytes.
// The header structure is present only for expanding disks (i.e. dynamic and
// differencing disks.) In case of dynamic and differential vhds the footer is
// replicated at the beginning of the disk as well, the header structure follows
// this replicated footer, the field 'HeaderOffset' in the footer contains absolute
// offset to the header structure.
//
type Header struct {
	// Offset =  0, Size = 8
	Cookie *vhdcore.Cookie
	// Offset =  8, Size = 8
	DataOffset int64
	// Offset = 16, Size = 8
	TableOffset int64
	// Offset = 24, Size = 4
	HeaderVersion VhdHeaderVersion
	// Offset = 28, Size = 4
	MaxTableEntries uint32
	// Offset = 32, Size = 4
	BlockSize uint32
	// Offset = 36, Size = 4
	CheckSum uint32
	// Offset = 40, Size = 16
	ParentUniqueID *common.UUID
	// Offset = 56, Size = 4
	ParentTimeStamp *time.Time
	// Offset = 60, Size = 4
	Reserved uint32
	// Offset = 64, Size = 512
	// This field contains a Unicode string (UTF-16) of the parent hard disk filename.
	// This will be absolute path to the parent disk of differencing disk.
	// If this field is set, then the ParentLocators collection will also contain
	// an entry with the same path, the PlatformCode of that field will be PlatformCodeW2Ku.
	ParentPath string
	// Offset = 576, Count = 8
	// Collection of entries store an absolute byte offset in the file where the
	// parent locator for a differencing hard disk is stored.
	// This field is used only for differencing disks and will be set to zero for
	// dynamic disks.
	ParentLocators parentlocator.ParentLocators
	// Offset = 0, Size = 1024
	// The entire header as raw bytes
	RawData []byte
}
