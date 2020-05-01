package mbr

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/diskfs/go-diskfs/util"
)

// Partition represents the structure of a single partition on the disk
// note that start and end cylinder, head, sector (CHS) are ignored, for the most part.
// godiskfs works with disks that support [Logical Block Addressing (LBA)](https://en.wikipedia.org/wiki/Logical_block_addressing)
type Partition struct {
	Bootable      bool
	Type          Type   //
	Start         uint32 // Start first absolute LBA sector for partition
	Size          uint32 // Size number of sectors in partition
	StartCylinder byte
	StartHead     byte
	StartSector   byte
	EndCylinder   byte
	EndHead       byte
	EndSector     byte
	// we need this for calculations
	logicalSectorSize  int
	physicalSectorSize int
}

// PartitionEqualBytes compares if the bytes for 2 partitions are equal, ignoring CHS start and end
func PartitionEqualBytes(b1, b2 []byte) bool {
	if (b1 == nil && b2 != nil) || (b2 == nil && b1 != nil) {
		return false
	}
	if b1 == nil && b2 == nil {
		return true
	}
	if len(b1) != len(b2) {
		return false
	}
	return b1[0] == b2[0] &&
		b1[4] == b2[4] &&
		bytes.Compare(b1[8:12], b2[8:12]) == 0 &&
		bytes.Compare(b1[12:16], b2[12:16]) == 0
}

// Equal compares if another partition is equal to this one, ignoring CHS start and end
func (p *Partition) Equal(p2 *Partition) bool {
	if p2 == nil {
		return false
	}
	return p.Bootable == p2.Bootable &&
		p.Type == p2.Type &&
		p.Start == p2.Start &&
		p.Size == p2.Size
}

func (p *Partition) GetSize() int64 {
	_, lss := p.sectorSizes()
	return int64(p.Size) * int64(lss)
}
func (p *Partition) GetStart() int64 {
	_, lss := p.sectorSizes()
	return int64(p.Start) * int64(lss)
}

// toBytes return the 16 bytes for this partition
func (p *Partition) toBytes() ([]byte, error) {
	b := make([]byte, partitionEntrySize, partitionEntrySize)
	if p.Bootable {
		b[0] = 0x80
	} else {
		b[0] = 0x00
	}
	b[1] = p.StartHead
	b[2] = p.StartSector
	b[3] = p.StartCylinder
	b[4] = byte(p.Type)
	b[5] = p.EndHead
	b[6] = p.EndSector
	b[7] = p.EndCylinder
	binary.LittleEndian.PutUint32(b[8:12], p.Start)
	binary.LittleEndian.PutUint32(b[12:16], p.Size)
	return b, nil
}

// partitionFromBytes create a partition entry from 16 bytes
func partitionFromBytes(b []byte, logicalSectorSize, physicalSectorSize int) (*Partition, error) {
	if len(b) != partitionEntrySize {
		return nil, fmt.Errorf("Data for partition was %d bytes instead of expected %d", len(b), partitionEntrySize)
	}
	var bootable bool
	switch b[0] {
	case 0x00:
		bootable = false
	case 0x80:
		bootable = true
	default:
		return nil, fmt.Errorf("Invalid partition")
	}

	return &Partition{
		Bootable:           bootable,
		StartHead:          b[1],
		StartSector:        b[2],
		StartCylinder:      b[3],
		Type:               Type(b[4]),
		EndHead:            b[5],
		EndSector:          b[6],
		EndCylinder:        b[7],
		Start:              binary.LittleEndian.Uint32(b[8:12]),
		Size:               binary.LittleEndian.Uint32(b[12:16]),
		logicalSectorSize:  logicalSectorSize,
		physicalSectorSize: physicalSectorSize,
	}, nil
}

// WriteContents fills the partition with the contents provided
// reads from beginning of reader to exactly size of partition in bytes
func (p *Partition) WriteContents(f util.File, contents io.Reader) (uint64, error) {
	pss, lss := p.sectorSizes()
	total := uint64(0)

	// chunks of physical sector size for efficient writing
	b := make([]byte, pss, pss)
	// we start at the correct byte location
	start := p.Start * uint32(lss)
	size := p.Size * uint32(lss)

	// loop in physical sector sizes
	for {
		read, err := contents.Read(b)
		if err != nil && err != io.EOF {
			return total, fmt.Errorf("Could not read contents to pass to partition: %v", err)
		}
		tmpTotal := uint64(read) + total
		if uint32(tmpTotal) > size {
			return total, fmt.Errorf("Requested to write at least %d bytes to partition but maximum size is %d", tmpTotal, size)
		}
		var written int
		if read > 0 {
			written, err = f.WriteAt(b[:read], int64(start)+int64(total))
			if err != nil {
				return total, fmt.Errorf("Error writing to file: %v", err)
			}
			// increment our total
			total = total + uint64(written)
		}
		// is this the end of the data?
		if err == io.EOF {
			break
		}
	}
	// did the total written equal the size of the partition?
	if total != uint64(size) {
		return total, fmt.Errorf("Write %d bytes to partition but actual size is %d", total, size)
	}
	return total, nil
}

// readContents reads the contents of the partition into a writer
// streams the entire partition to the writer
func (p *Partition) ReadContents(f util.File, out io.Writer) (int64, error) {
	pss, lss := p.sectorSizes()
	total := int64(0)
	// chunks of physical sector size for efficient writing
	b := make([]byte, pss, pss)
	// we start at the correct byte location
	start := p.Start * uint32(lss)
	size := p.Size * uint32(lss)

	// loop in physical sector sizes
	for {
		read, err := f.ReadAt(b, int64(start)+total)
		if err != nil && err != io.EOF {
			return total, fmt.Errorf("Error reading from file: %v", err)
		}
		if read > 0 {
			out.Write(b[:read])
		}
		// increment our total
		total += int64(read)
		// is this the end of the data?
		if err == io.EOF || total >= int64(size) {
			break
		}
	}
	return total, nil
}

// sectorSizes get the sector sizes for this partition, falling back to the defaults if 0
func (p *Partition) sectorSizes() (physical, logical int) {
	physical = p.physicalSectorSize
	if physical == 0 {
		physical = physicalSectorSize
	}
	logical = p.logicalSectorSize
	if logical == 0 {
		logical = logicalSectorSize
	}
	return physical, logical
}
