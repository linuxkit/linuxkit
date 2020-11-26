package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	blocksize = 2048
	pvdBlock  = 16
)

type Enumerable interface {
	Each(handler func(Printable))
}
type Printable interface {
}

type dirEntry struct {
	recordSize               uint8
	extAttrSize              uint8
	location                 uint32
	size                     uint32
	creation                 time.Time
	isHidden                 bool
	isSubdirectory           bool
	isAssociated             bool
	hasExtendedAttrs         bool
	hasOwnerGroupPermissions bool
	hasMoreEntries           bool
	volumeSequence           uint16
	filename                 string
	extensions               []directoryEntrySystemUseExtension
}

type directoryEntrySystemUseExtension struct {
	entryType string
	length    uint8
	version   uint8
	data      []byte
}
type dirEntryList []*dirEntry

func (d dirEntryList) Each(handler func(Printable)) {
	for _, e := range d {
		handler(e)
	}
}

type pathEntry struct {
	nameSize      uint8
	size          uint16
	extAttrLength uint8
	location      uint32
	parentIndex   uint16
	dirname       string
}
type pathEntryList []*pathEntry

func (p pathEntryList) Each(handler func(Printable)) {
	for _, e := range p {
		handler(e)
	}
}

func main() {
	// get the args
	args := os.Args
	if len(args) < 2 {
		log.Fatalf("Must provide command. Usage:\n%s <command> <args> ... <args>\nCommands: directory pathtable", args[0])
	}

	cmd := args[1]
	opts := args[2:]
	switch cmd {
	case "directory":
		readdirCmd(opts)
	case "pathtable":
		readpathCmd(opts)
	default:
		log.Fatalf("Unknown command: %s", cmd)
	}

}

func readpathCmd(opts []string) {
	if len(opts) != 1 {
		log.Fatalf("Command 'pathtable' must have exactly one arguments. Options: <filename>")
	}
	filename := opts[0]
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open file %s: %v", filename, err)
	}

	// get the path table L location and size from the primary volume descriptor
	b := make([]byte, blocksize, blocksize)
	// get the primary volume descriptor
	read, err := f.ReadAt(b, pvdBlock*int64(blocksize))
	if err != nil {
		log.Fatalf("Error reading path table location: %v", err)
	}
	if read != len(b) {
		log.Fatalf("Read %d bytes instead of expected %d", read, len(b))
	}
	// get the location and size
	size := binary.LittleEndian.Uint32(b[132 : 132+4])
	location := binary.LittleEndian.Uint32(b[140 : 140+4])

	// read in the path table
	ptBytes := make([]byte, size, size)
	read, err = f.ReadAt(ptBytes, int64(location*blocksize))
	if err != nil {
		log.Fatalf("Error reading path table of size from location %d: %v", size, location, err)
	}
	if read != len(ptBytes) {
		log.Fatalf("Read %d bytes instead of expected %d", read, len(b))
	}

	// now parse the path table
	// cycle through
	entries := make([]*pathEntry, 0, 10)
	// basic bytes are 9
	for i := 0; i < len(ptBytes); {
		// get the size of the next record
		nameSize := ptBytes[i+0]
		recordSize := uint16(nameSize) + 8
		if nameSize%2 != 0 {
			recordSize++
		}

		e := &pathEntry{
			nameSize:      nameSize,
			size:          recordSize,
			extAttrLength: ptBytes[i+1],
			location:      binary.LittleEndian.Uint32(ptBytes[i+2 : i+6]),
			parentIndex:   binary.LittleEndian.Uint16(ptBytes[i+6 : i+8]),
			dirname:       string(ptBytes[i+8 : i+8+int(nameSize)]),
		}
		entries = append(entries, e)
		i += int(recordSize)
	}

	dump(pathEntryList(entries))
}
func readdirCmd(opts []string) {
	if len(opts) != 2 {
		log.Fatalf("Command 'directory' must have exactly two arguments. Options: <filename> <path>")
	}
	filename := opts[0]
	p := opts[1]
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open file %s: %v", filename, err)
	}

	// simplistically get the root file system
	b := make([]byte, blocksize, blocksize)
	// get the primary volume descriptor
	read, err := f.ReadAt(b, pvdBlock*int64(blocksize))
	if err != nil {
		log.Fatalf("Error reading primary volume descriptor: %v", err)
	}
	if read != len(b) {
		log.Fatalf("Read %d bytes instead of expected %d", read, len(b))
	}
	// get the root directory block and size
	rootDirEntryBytes := b[156 : 156+34]
	// get the location and size
	location := binary.LittleEndian.Uint32(rootDirEntryBytes[2 : 2+4])
	size := binary.LittleEndian.Uint32(rootDirEntryBytes[10 : 10+4])

	// now parse the requested path and find out which one we want
	parts, err := splitPath(p)
	if err != nil {
		log.Fatalf("Could not parse path %s: %v", p, err)
	}
	err = readAndProcessDirs(parts, location, size, f)
	if err != nil {
		log.Fatalf("Failed to process path %s: %v", p, err)
	}
}

func readAndProcessDirs(parts []string, location, size uint32, f *os.File) error {
	dirs := readDirectory(location, size, f)
	if len(parts) < 1 {
		dump(dirEntryList(dirs))
	} else {
		current, parts := parts[0], parts[1:]
		child := findChild(current, dirs)
		if child == nil {
			return fmt.Errorf("Could not find directory %s", current)
		}
		readAndProcessDirs(parts, child.location, child.size, f)
	}
	return nil
}

func findChild(name string, entries []*dirEntry) *dirEntry {
	for _, e := range entries {
		if name == e.filename {
			return e
		}
	}
	return nil
}

func dump(entries Enumerable) {
	entries.Each(func(e Printable) {
		val := fmt.Sprintf("%#v", e)
		// strip the type header and add a , at the end
		//re := regexp.MustCompile(`^&main\.[^{]*`)
		//val = re.ReplaceAllString(val, ``)

		re := regexp.MustCompile(`main\.`)
		val = re.ReplaceAllString(val, ``)
		fmt.Printf("%s,\n", val)
	})
}

func readDirectory(location, size uint32, f *os.File) []*dirEntry {
	// read the correct number of bytes, then process entries one by one
	b := make([]byte, size, size)
	read, err := f.ReadAt(b, int64(location)*int64(blocksize))
	if err != nil {
		log.Fatalf("Failed to read directory at location %d", location)
	}
	if read != len(b) {
		log.Fatalf("Read %d bytes instead of expected %d at location %d", read, len(b), location)
	}
	// cycle through
	entries := make([]*dirEntry, 0, 10)
	for i := 0; i < len(b); {
		// get the size of the next record
		recordSize := b[i+0]
		// size == 0 means we have no more in this sector
		if recordSize == 0 {
			i += (blocksize - i%blocksize)
			continue
		}
		recordBytes := b[i+0 : i+int(recordSize)]
		i += int(recordSize)

		extAttrSize := recordBytes[1]
		location := binary.LittleEndian.Uint32(recordBytes[2:6])
		size := binary.LittleEndian.Uint32(recordBytes[10:14])

		// get the flags
		isSubdirectory := recordBytes[26]&0x02 == 0x02

		// size includes the ";1" at the end as two bytes if a file and not a directory
		namelen := recordBytes[32]

		// get the filename itself
		filename := string(recordBytes[33 : 33+namelen])

		// get the extensions
		// and now for extensions in the system use area
		suspFields := make([]directoryEntrySystemUseExtension, 0)
		suspBytes := make([]byte, 0)

		if int(recordSize) > 33+int(namelen) {
			suspBytes = recordBytes[33+int(namelen):]
		}
		// minimum size of 4 bytes for any SUSP entry
		for i := 0; i+4 < len(suspBytes); {
			// get the indicator
			signature := string(suspBytes[i : i+2])
			size := suspBytes[i+2]
			version := suspBytes[i+3]
			data := make([]byte, 0)
			if size > 4 {
				data = suspBytes[i+4 : i+int(size)]
			}

			suspEntry := directoryEntrySystemUseExtension{
				entryType: signature,
				length:    size,
				version:   version,
				data:      data,
			}
			suspFields = append(suspFields, suspEntry)
			i += int(size)
		}

		e := &dirEntry{
			recordSize:     recordSize,
			extAttrSize:    extAttrSize,
			location:       location,
			size:           size,
			isSubdirectory: isSubdirectory,
			filename:       filename,
			extensions:     suspFields,
		}
		entries = append(entries, e)
	}
	return entries
}

func universalizePath(p string) (string, error) {
	// globalize the separator
	ps := strings.Replace(p, "\\", "/", 0)
	if ps[0] != '/' {
		return "", errors.New("Must use absolute paths")
	}
	return ps, nil
}
func splitPath(p string) ([]string, error) {
	ps, err := universalizePath(p)
	if err != nil {
		return nil, err
	}
	// we need to split such that each one ends in "/", except possibly the last one
	parts := strings.Split(ps, "/")
	// eliminate empty parts
	ret := make([]string, 0)
	for _, sub := range parts {
		if sub != "" {
			ret = append(ret, sub)
		}
	}
	return ret, nil
}
