package vhdcore

// VhdDefaultBlockSize is the default block size of the VHD.
//
const VhdDefaultBlockSize int64 = 512 * 1024

// VhdNoDataLong is the value in the BAT indicating a block is empty.
//
const VhdNoDataLong int64 = ^int64(0)

// VhdNoDataInt is the value in the BAT indicating a block is empty.
//
const VhdNoDataInt uint32 = 0xFFFFFFFF

// VhdPageSize is the size of the VHD page size.
//
const VhdPageSize int64 = 512

// VhdFooterSize is the size of the VHD footer in bytes.
//
const VhdFooterSize int64 = 512

// VhdSectorLength is the sector length which is always 512 bytes, as per VHD specification.
//
const VhdSectorLength int64 = 512

// VhdFooterChecksumOffset is the bye offset of checksum field in the VHD footer.
//
const VhdFooterChecksumOffset int = 64
