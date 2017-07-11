package block

// DataReader interface that all block readers specific to disk type (fixed,
// dynamic, differencing) needs to satisfy.
//
type DataReader interface {
	// Read reads the disk block identified by the parameter block
	//
	Read(block *Block) ([]byte, error)
}
