package iso9660wrap

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"time"
)

func Panicf(format string, v ...interface{}) {
	panic(fmt.Errorf(format, v...))
}

const volumeDescriptorSetMagic = "\x43\x44\x30\x30\x31\x01"

const primaryVolumeSectorNum uint32 = 16
const numVolumeSectors uint32 = 2 // primary + terminator
const littleEndianPathTableSectorNum uint32 = primaryVolumeSectorNum + numVolumeSectors
const bigEndianPathTableSectorNum uint32 = littleEndianPathTableSectorNum + 1
const numPathTableSectors = 2 // no secondaries
const rootDirectorySectorNum uint32 = primaryVolumeSectorNum + numVolumeSectors + numPathTableSectors

// WriteFile writes the contents of infh to an iso at outfh with the name provided
func WriteFile(outfh, infh *os.File) error {
	fileSize, filename, err := getInputFileSizeAndName(infh)
	if err != nil {
		return err
	}
	filename = strings.ToUpper(filename)
	if !filenameSatisfiesISOConstraints(filename) {
		return fmt.Errorf("Input file name %s does not satisfy the ISO9660 character set constraints", filename)
	}

	buf := make([]byte, fileSize, fileSize)
	_, err = infh.Read(buf)
	if err != nil {
		return err
	}

	return WriteBuffer(outfh, buf, filename)
}

// WriteBuffer writes the contents of buf to an iso at outfh with the name provided
func WriteBuffer(outfh io.Writer, buf []byte, filename string) error {
	fileSize := uint32(len(buf))
	r := bytes.NewReader(buf)

	// reserved sectors
	reservedAreaLength := int64(16 * SectorSize)
	_, err := outfh.Write(make([]byte,reservedAreaLength))
	if err != nil {
		return fmt.Errorf("could not write to output file: %s", err)
	}

	err = nil
	func() {
		defer func() {
			var ok bool
			e := recover()
			if e != nil {
				err, ok = e.(error)
				if !ok {
					panic(e)
				}
			}
		}()

		bufw := bufio.NewWriter(outfh)

		w := NewISO9660Writer(bufw)

		writePrimaryVolumeDescriptor(w, fileSize, filename)
		writeVolumeDescriptorSetTerminator(w)
		writePathTable(w, binary.LittleEndian)
		writePathTable(w, binary.BigEndian)
		writeData(w, r, fileSize, filename)

		w.Finish()

		err := bufw.Flush()
		if err != nil {
			panic(err)
		}
	}()
	if err != nil {
		return fmt.Errorf("could not write to output file: %s", err)
	}
	return nil
}

func writePrimaryVolumeDescriptor(w *ISO9660Writer, fileSize uint32, filename string) {
	if len(filename) > 32 {
		filename = filename[:32]
	}
	now := time.Now()

	sw := w.NextSector()
	if w.CurrentSector() != primaryVolumeSectorNum {
		Panicf("internal error: unexpected primary volume sector %d", w.CurrentSector())
	}

	sw.WriteByte('\x01')
	sw.WriteString(volumeDescriptorSetMagic)
	sw.WriteByte('\x00')

	sw.WritePaddedString("", 32)
	sw.WritePaddedString(filename, 32)

	sw.WriteZeros(8)
	sw.WriteBothEndianDWord(numTotalSectors(fileSize))
	sw.WriteZeros(32)

	sw.WriteBothEndianWord(1) // volume set size
	sw.WriteBothEndianWord(1) // volume sequence number
	sw.WriteBothEndianWord(uint16(SectorSize))
	sw.WriteBothEndianDWord(SectorSize) // path table length

	sw.WriteLittleEndianDWord(littleEndianPathTableSectorNum)
	sw.WriteLittleEndianDWord(0) // no secondary path tables
	sw.WriteBigEndianDWord(bigEndianPathTableSectorNum)
	sw.WriteBigEndianDWord(0) // no secondary path tables

	WriteDirectoryRecord(sw, "\x00", rootDirectorySectorNum) // root directory

	sw.WritePaddedString("", 128) // volume set identifier
	sw.WritePaddedString("", 128) // publisher identifier
	sw.WritePaddedString("", 128) // data preparer identifier
	sw.WritePaddedString("", 128) // application identifier

	sw.WritePaddedString("", 37) // copyright file identifier
	sw.WritePaddedString("", 37) // abstract file identifier
	sw.WritePaddedString("", 37) // bibliographical file identifier

	sw.WriteDateTime(now)         // volume creation
	sw.WriteDateTime(now)         // most recent modification
	sw.WriteUnspecifiedDateTime() // expires
	sw.WriteUnspecifiedDateTime() // is effective (?)

	sw.WriteByte('\x01') // version
	sw.WriteByte('\x00') // reserved

	sw.PadWithZeros() // 512 (reserved for app) + 653 (zeros)
}

func writeVolumeDescriptorSetTerminator(w *ISO9660Writer) {
	sw := w.NextSector()
	if w.CurrentSector() != primaryVolumeSectorNum+1 {
		Panicf("internal error: unexpected volume descriptor set terminator sector %d", w.CurrentSector())
	}

	sw.WriteByte('\xFF')
	sw.WriteString(volumeDescriptorSetMagic)

	sw.PadWithZeros()
}

func writePathTable(w *ISO9660Writer, bo binary.ByteOrder) {
	sw := w.NextSector()
	sw.WriteByte(1) // name length
	sw.WriteByte(0) // number of sectors in extended attribute record
	sw.WriteDWord(bo, rootDirectorySectorNum)
	sw.WriteWord(bo, 1) // parent directory recno (root directory)
	sw.WriteByte(0)     // identifier (root directory)
	sw.WriteByte(1)     // padding
	sw.PadWithZeros()
}

func writeData(w *ISO9660Writer, infh io.Reader, fileSize uint32, filename string) {
	sw := w.NextSector()
	if w.CurrentSector() != rootDirectorySectorNum {
		Panicf("internal error: unexpected root directory sector %d", w.CurrentSector())
	}

	WriteDirectoryRecord(sw, "\x00", w.CurrentSector())
	WriteDirectoryRecord(sw, "\x01", rootDirectorySectorNum)
	WriteFileRecordHeader(sw, filename, w.CurrentSector()+1, fileSize)

	// Now stream the data.  Note that the first buffer is never of SectorSize,
	// since we've already filled a part of the sector.
	b := make([]byte, SectorSize)
	total := uint32(0)
	for {
		l, err := infh.Read(b)
		if err != nil && err != io.EOF {
			Panicf("could not read from input file: %s", err)
		}
		if l > 0 {
			sw = w.NextSector()
			sw.Write(b[:l])
			total += uint32(l)
		}
		if err == io.EOF {
			break
		}
	}
	if total != fileSize {
		Panicf("input file size changed while the ISO file was being created (expected to read %d, read %d)", fileSize, total)
	} else if w.CurrentSector() != numTotalSectors(fileSize)-1 {
		Panicf("internal error: unexpected last sector number (expected %d, actual %d)",
			numTotalSectors(fileSize)-1, w.CurrentSector())
	}
}

func numTotalSectors(fileSize uint32) uint32 {
	var numDataSectors uint32
	numDataSectors = (fileSize + (SectorSize - 1)) / SectorSize
	return 1 + rootDirectorySectorNum + numDataSectors
}

func getInputFileSizeAndName(fh *os.File) (uint32, string, error) {
	fi, err := fh.Stat()
	if err != nil {
		return 0, "", err
	}
	if fi.Size() >= math.MaxUint32 {
		return 0, "", fmt.Errorf("file size %d is too large", fi.Size())
	}
	return uint32(fi.Size()), fi.Name(), nil
}

func filenameSatisfiesISOConstraints(filename string) bool {
	invalidCharacter := func(r rune) bool {
		// According to ISO9660, only capital letters, digits, and underscores
		// are permitted.  Some sources say a dot is allowed as well.  I'm too
		// lazy to figure it out right now.
		if r >= 'A' && r <= 'Z' {
			return false
		} else if r >= '0' && r <= '9' {
			return false
		} else if r == '_' {
			return false
		} else if r == '.' {
			return false
		}
		return true
	}
	return strings.IndexFunc(filename, invalidCharacter) == -1
}
