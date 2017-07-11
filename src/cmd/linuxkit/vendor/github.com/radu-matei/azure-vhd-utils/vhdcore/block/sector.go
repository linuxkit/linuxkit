package block

// Sector represents a sector in the 'data section' of a block.
//
type Sector struct {
	BlockIndex  uint32
	SectorIndex int64
	Data        []byte
}
