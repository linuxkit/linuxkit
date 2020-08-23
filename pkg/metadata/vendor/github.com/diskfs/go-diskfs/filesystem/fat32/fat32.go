package fat32

import (
	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/util"
)

// MsdosMediaType is the (mostly unused) media type. However, we provide and export the known constants for it.
type MsdosMediaType uint8

const (
	// Media8InchDrDos for single-sided 250KB DR-DOS disks
	Media8InchDrDos MsdosMediaType = 0xe5
	// Media525InchTandy for 5.25 inch floppy disks for Tandy
	Media525InchTandy MsdosMediaType = 0xed
	// MediaCustomPartitionsDrDos for non-standard custom DR-DOS partitions utilizing non-standard BPB formats
	MediaCustomPartitionsDrDos MsdosMediaType = 0xee
	// MediaCustomSuperFloppyDrDos for non-standard custom superfloppy disks for DR-DOS
	MediaCustomSuperFloppyDrDos MsdosMediaType = 0xef
	// Media35Inch for standard 1.44MB and 2.88MB 3.5 inch floppy disks
	Media35Inch MsdosMediaType = 0xf0
	// MediaDoubleDensityAltos for double-density floppy disks for Altos only
	MediaDoubleDensityAltos MsdosMediaType = 0xf4
	// MediaFixedDiskAltos for fixed disk 1.95MB for Altos only
	MediaFixedDiskAltos MsdosMediaType = 0xf5
	// MediaFixedDisk for standard fixed disks - can be used for any partitioned fixed or removable media where the geometry is defined in the BPB
	MediaFixedDisk MsdosMediaType = 0xf8
)

// SectorSize indicates what the sector size in bytes is
type SectorSize uint16

const (
	// SectorSize512 is a sector size of 512 bytes, used as the logical size for all FAT filesystems
	SectorSize512        SectorSize = 512
	maxClusterSize       int        = 128
	minClusterSize       int        = 65529
	bytesPerSlot         int        = 32
	maxCharsLongFilename int        = 13
)

// FileSystem implememnts the FileSystem interface
type FileSystem struct {
	bootSector      msDosBootSector
	fsis            FSInformationSector
	table           table
	dataStart       uint32
	bytesPerCluster int
	size            int64
	start           int64
	file            util.File
}

// Equal compare if two filesystems are equal
func (fs *FileSystem) Equal(a *FileSystem) bool {
	localMatch := fs.file == a.file && fs.dataStart == a.dataStart && fs.bytesPerCluster == a.bytesPerCluster
	tableMatch := fs.table.equal(&a.table)
	bsMatch := fs.bootSector.equal(&a.bootSector)
	fsisMatch := fs.fsis == a.fsis
	return localMatch && tableMatch && bsMatch && fsisMatch
}

// Create creates a FAT32 filesystem in a given file or device
//
// requires the util.File where to create the filesystem, size is the size of the filesystem in bytes,
// start is how far in bytes from the beginning of the util.File to create the filesystem,
// and blocksize is is the logical blocksize to use for creating the filesystem
//
// note that you are *not* required to create the filesystem on the entire disk. You could have a disk of size
// 20GB, and create a small filesystem of size 50MB that begins 2GB into the disk.
// This is extremely useful for creating filesystems on disk partitions.
//
// Note, however, that it is much easier to do this using the higher-level APIs at github.com/diskfs/go-diskfs
// which allow you to work directly with partitions, rather than having to calculate (and hopefully not make any errors)
// where a partition starts and ends.
//
// If the provided blocksize is 0, it will use the default of 512 bytes. If it is any number other than 0
// or 512, it will return an error.
func Create(f util.File, size int64, start int64, blocksize int64, volumeLabel string) (*FileSystem, error) {
	if volumeLabel == "" {
		volumeLabel = "NO NAME"
	}
	// ensure the volumeLabel is proper sized
	volumeLabel = fmt.Sprintf("%-11.11s", volumeLabel)
	// blocksize must be <=0 or exactly SectorSize512 or error
	if blocksize != int64(SectorSize512) && blocksize > 0 {
		return nil, fmt.Errorf("blocksize for FAT32 must be either 512 bytes or 0, not %d", blocksize)
	}
	if size > Fat32MaxSize {
		return nil, fmt.Errorf("requested size is larger than maximum allowed FAT32, requested %d, maximum %d", size, Fat32MaxSize)
	}
	if size < blocksize*4 {
		return nil, fmt.Errorf("requested size is smaller than minimum allowed FAT32, requested %d minimum %d", size, blocksize*4)
	}
	// FAT filesystems use time-of-day of creation as a volume ID
	now := time.Now()
	// because we like the fudges other people did for uniqueness
	volid := uint32(now.Unix()<<20 | (now.UnixNano() / 1000000))

	fsisPrimarySector := uint16(1)
	fsisBackupSector := uint16(6)

	/*
		size calculations
		we have the total size of the disk from `size uint64`
		we have the blocksize fixed at SectorSize512
		    so we can calculate diskSectors = size/512
		we know the number of reserved sectors is 32
		so the number of non-reserved sectors: data + FAT = diskSectos - 32
		now we need to figure out cluster size. The allowed number of:
		    sectors per cluster: 1, 2, 4, 8, 16, 32, 64, 128
		    bytes per cluster: 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536
		    since FAT32 uses the least significant 28 bits of a 4-byte entry (uint32) as pointers to a cluster,
		       the maximum cluster pointer address of a FAT32 entry is 268,435,456. However, several
		       entries are reserved, notably 0x0FFFFFF7-0x0FFFFFFF flag bad cluster to end of file,
		       0x0000000 flags an empty cluster, and 0x0000001 is not used, so we only have
		       a potential 268,435,444 pointer entries
		    the maximum size of a disk for FAT32 is 16 sectors per cluster = 8KB/cluster * 268435444 = ~2TB

		Follow Microsoft's `format` commad as per http://www.win.tue.nl/~aeb/linux/fs/fat/fatgen103.pdf p. 20.
		Thanks to github.com/dosfstools/dosfstools for the link
		Filesystem size / cluster size
		   <= 260M      /   1 sector =   512 bytes
			 <=   8G      /   8 sector =  4096 bytes
			 <=  16G      /  32 sector = 16384 bytes
			 <=  32G      /  64 sector = 32768 bytes
			  >  32G      / 128 sector = 65536 bytes
	*/

	var sectorsPerCluster uint8
	switch {
	case size <= 260*MB:
		sectorsPerCluster = 1
	case size <= 8*GB:
		sectorsPerCluster = 8
	case size <= 16*GB:
		sectorsPerCluster = 32
	case size <= 32*GB:
		sectorsPerCluster = 64
	case size <= Fat32MaxSize:
		sectorsPerCluster = 128
	}

	// stick with uint32 and round down
	totalSectors := uint32(size / int64(SectorSize512))
	reservedSectors := uint16(32)
	dataSectors := totalSectors - uint32(reservedSectors)
	totalClusters := dataSectors / uint32(sectorsPerCluster)
	// FAT uses 4 bytes per cluster pointer
	//   so a 512 byte sector can store 512/4 = 128 pointer entries
	//   therefore sectors per FAT = totalClusters / 128
	sectorsPerFat := uint16(totalClusters / 128)

	// what is our FAT ID / Media Type?
	mediaType := uint8(MediaFixedDisk)

	fatIDbase := uint32(0x0f << 24)
	fatID := fatIDbase + 0xffff00 + uint32(mediaType)

	// we need an Extended BIOS Parameter Block
	dos20bpb := dos20BPB{
		sectorsPerCluster:    sectorsPerCluster,
		reservedSectors:      reservedSectors,
		fatCount:             2,
		totalSectors:         0,
		mediaType:            mediaType,
		bytesPerSector:       SectorSize512,
		rootDirectoryEntries: 0,
		sectorsPerFat:        0,
	}

	// some fake logic for heads, since everything is LBA access anyways
	dos331bpb := dos331BPB{
		dos20BPB:        &dos20bpb,
		totalSectors:    totalSectors,
		heads:           1,
		sectorsPerTrack: 1,
		hiddenSectors:   0,
	}

	ebpb := dos71EBPB{
		dos331BPB:             &dos331bpb,
		version:               fatVersion0,
		rootDirectoryCluster:  2,
		fsInformationSector:   fsisPrimarySector,
		backupFSInfoSector:    fsisBackupSector,
		bootFileName:          [12]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		extendedBootSignature: longDos71EBPB,
		volumeSerialNumber:    volid,
		volumeLabel:           fmt.Sprintf("%-11.11s", volumeLabel), // "NO NAME    "
		fileSystemType:        fileSystemTypeFAT32,
		mirrorFlags:           0,
		reservedFlags:         0,
		driveNumber:           128,
		sectorsPerFat:         uint32(sectorsPerFat),
	}
	// we need a new boot sector
	bs := msDosBootSector{
		oemName:            "godiskfs",
		jumpInstruction:    [3]byte{0xeb, 0x58, 0x90},
		bootCode:           []byte{},
		biosParameterBlock: &ebpb,
	}
	/*
		err := bs.write(f)
		if err != nil {
			return nil, fmt.Errorf("Error writing MS-DOS Boot Sector: %v", err)
		}
	*/
	b, err := bs.toBytes()
	if err != nil {
		return nil, fmt.Errorf("Error converting MS-DOS Boot Sector to bytes: %v", err)
	}
	// write to the file
	count, err := f.WriteAt(b, 0+int64(start))
	if err != nil {
		return nil, fmt.Errorf("Error writing MS-DOS Boot Sector to disk: %v", err)
	}
	if count != int(SectorSize512) {
		return nil, fmt.Errorf("Wrote %d bytes of MS-DOS Boot Sector to disk instead of expected %d", count, SectorSize512)
	}

	// boot sector is in place

	// create and allocate FAT32 FSInformationSector
	fsis := FSInformationSector{
		lastAllocatedCluster:  0xffffffff,
		freeDataClustersCount: 0xffffffff,
	}

	fsisBytes, err := fsis.toBytes()
	if err != nil {
		return nil, fmt.Errorf("Could not create a valid byte stream for a FAT32 Filesystem Information Sector: %v", err)
	}
	fsisPrimary := int64(fsisPrimarySector * uint16(SectorSize512))
	fsisBackup := int64(fsisBackupSector * uint16(SectorSize512))

	f.WriteAt(fsisBytes, fsisPrimary+int64(start))
	f.WriteAt(fsisBytes, fsisBackup+int64(start))

	// write FAT tables
	eocMarker := uint32(0x0fffffff)
	fatPrimaryStart := reservedSectors * uint16(SectorSize512)
	fatSize := uint32(sectorsPerFat) * uint32(SectorSize512)
	fatSecondaryStart := uint64(fatPrimaryStart) + uint64(fatSize)
	maxCluster := fatSize / 4
	rootDirCluster := uint32(2)
	fat := table{
		fatID:          fatID,
		eocMarker:      eocMarker,
		size:           fatSize,
		rootDirCluster: rootDirCluster,
		clusters: map[uint32]uint32{
			// when we start, there is just one directory with a single cluster
			rootDirCluster: eocMarker,
		},
		maxCluster: maxCluster,
	}

	fatBytes, err := fat.bytes()
	if err != nil {
		return nil, fmt.Errorf("Error converting FAT32 table into bytes: %v", err)
	}
	_, err = f.WriteAt(fatBytes, int64(fatPrimaryStart)+int64(start))
	if err != nil {
		return nil, fmt.Errorf("Unable to write primary FAT table: %v", err)
	}
	_, err = f.WriteAt(fatBytes, int64(fatSecondaryStart)+int64(start))
	if err != nil {
		return nil, fmt.Errorf("Unable to write backup FAT table: %v", err)
	}

	// where does our data start?
	dataStart := uint32(fatSecondaryStart) + fatSize

	// create root directory
	// there is nothing in there
	fs := &FileSystem{
		bootSector:      bs,
		fsis:            fsis,
		table:           fat,
		dataStart:       dataStart,
		bytesPerCluster: int(sectorsPerCluster) * int(SectorSize512),
		start:           start,
		size:            size,
		file:            f,
	}

	// be sure to zero out the root cluster, so we do not pick up phantom
	// entries.
	clusterStart := fs.start + int64(fs.dataStart)
	// length of cluster in bytes
	tmpb := make([]byte, fs.bytesPerCluster)
	// zero out the root directory cluster
	written, err := f.WriteAt(tmpb, clusterStart)
	if err != nil {
		return nil, fmt.Errorf("failed to zero out root directory: %v", err)
	}
	if written != len(tmpb) || written != fs.bytesPerCluster {
		return nil, fmt.Errorf("incomplete zero out of root directory, wrote %d bytes instead of expected %d for cluster size %d", written, len(b), fs.bytesPerCluster)
	}

	// create a volumelabel entry in the root directory
	rootDir := &Directory{
		directoryEntry: directoryEntry{
			clusterLocation: uint32(fs.table.rootDirCluster),
			isSubdirectory:  true,
			filesystem:      fs,
		},
	}
	_, err = fs.mkLabel(rootDir, volumeLabel)
	if err != nil {
		return nil, fmt.Errorf("failed to create volume label root directory entry '%s': %v", volumeLabel, err)
	}
	// write the root directory entries to disk
	err = fs.writeDirectoryEntries(rootDir)
	if err != nil {
		return nil, fmt.Errorf("Error writing root directory to disk: %v", err)
	}

	return fs, nil
}

// Read reads a filesystem from a given disk.
//
// requires the util.File where to read the filesystem, size is the size of the filesystem in bytes,
// start is how far in bytes from the beginning of the util.File the filesystem is expected to begin,
// and blocksize is is the logical blocksize to use for creating the filesystem
//
// note that you are *not* required to read a filesystem on the entire disk. You could have a disk of size
// 20GB, and a small filesystem of size 50MB that begins 2GB into the disk.
// This is extremely useful for working with filesystems on disk partitions.
//
// Note, however, that it is much easier to do this using the higher-level APIs at github.com/diskfs/go-diskfs
// which allow you to work directly with partitions, rather than having to calculate (and hopefully not make any errors)
// where a partition starts and ends.
//
// If the provided blocksize is 0, it will use the default of 512 bytes. If it is any number other than 0
// or 512, it will return an error.
func Read(file util.File, size int64, start int64, blocksize int64) (*FileSystem, error) {
	// blocksize must be <=0 or exactly SectorSize512 or error
	if blocksize != int64(SectorSize512) && blocksize > 0 {
		return nil, fmt.Errorf("blocksize for FAT32 must be either 512 bytes or 0, not %d", blocksize)
	}
	if size > Fat32MaxSize {
		return nil, fmt.Errorf("requested size is larger than maximum allowed FAT32 size %d", Fat32MaxSize)
	}
	if size < blocksize*4 {
		return nil, fmt.Errorf("requested size is smaller than minimum allowed FAT32 size %d", blocksize*4)
	}

	// load the information from the disk
	// read first 512 bytes from the file
	bsb := make([]byte, SectorSize512, SectorSize512)
	n, err := file.ReadAt(bsb, start)
	if err != nil {
		return nil, fmt.Errorf("Could not read bytes from file: %v", err)
	}
	if uint16(n) < uint16(SectorSize512) {
		return nil, fmt.Errorf("Only could read %d bytes from file", n)
	}
	bs, err := msDosBootSectorFromBytes(bsb)

	if err != nil {
		return nil, fmt.Errorf("Error reading MS-DOS Boot Sector: %v", err)
	}

	sectorsPerFat := bs.biosParameterBlock.sectorsPerFat
	fatSize := uint32(sectorsPerFat) * uint32(SectorSize512)
	reservedSectors := bs.biosParameterBlock.dos331BPB.dos20BPB.reservedSectors
	sectorsPerCluster := bs.biosParameterBlock.dos331BPB.dos20BPB.sectorsPerCluster
	fatPrimaryStart := uint64(reservedSectors) * uint64(SectorSize512)
	fatSecondaryStart := uint64(fatPrimaryStart) + uint64(fatSize)

	fsisBytes := make([]byte, 512, 512)
	read, err := file.ReadAt(fsisBytes, int64(bs.biosParameterBlock.fsInformationSector)*int64(blocksize)+int64(start))
	if err != nil {
		return nil, fmt.Errorf("Unable to read bytes for FSInformationSector: %v", err)
	}
	if read != 512 {
		return nil, fmt.Errorf("Read %d bytes instead of expected %d for FS Information Sector", read, 512)
	}
	fsis, err := fsInformationSectorFromBytes(fsisBytes)
	if err != nil {
		return nil, fmt.Errorf("Error reading FileSystem Information Sector: %v", err)
	}

	b := make([]byte, fatSize, fatSize)
	file.ReadAt(b, int64(fatPrimaryStart)+int64(start))
	fat, err := tableFromBytes(b)

	if err != nil {
		return nil, fmt.Errorf("Error reading primary FAT32 Table: %v", err)
	}
	file.ReadAt(b, int64(fatSecondaryStart)+int64(start))
	_, err = tableFromBytes(b)
	if err != nil {
		return nil, fmt.Errorf("Error reading backup FAT32 Table: %v", err)
	}
	dataStart := uint32(fatSecondaryStart) + fat.size

	return &FileSystem{
		bootSector:      *bs,
		fsis:            *fsis,
		table:           *fat,
		dataStart:       dataStart,
		bytesPerCluster: int(sectorsPerCluster) * int(SectorSize512),
		start:           start,
		size:            size,
		file:            file,
	}, nil
}

// Type returns the type code for the filesystem. Always returns filesystem.TypeFat32
func (fs *FileSystem) Type() filesystem.Type {
	return filesystem.TypeFat32
}

// Mkdir make a directory at the given path. It is equivalent to `mkdir -p`, i.e. idempotent, in that:
//
// * It will make the entire tree path if it does not exist
// * It will not return an error if the path already exists
func (fs *FileSystem) Mkdir(p string) error {
	_, _, err := fs.readDirWithMkdir(p, true)
	// we are not interesting in returning the entries
	return err
}

// ReadDir return the contents of a given directory in a given filesystem.
//
// Returns a slice of os.FileInfo with all of the entries in the directory.
//
// Will return an error if the directory does not exist or is a regular file and not a directory
func (fs *FileSystem) ReadDir(p string) ([]os.FileInfo, error) {
	_, entries, err := fs.readDirWithMkdir(p, false)
	if err != nil {
		return nil, fmt.Errorf("Error reading directory %s: %v", p, err)
	}
	// once we have made it here, looping is done. We have found the final entry
	// we need to return all of the file info
	count := len(entries)
	ret := make([]os.FileInfo, count, count)
	for i, e := range entries {
		shortName := e.filenameShort
		if e.lowercaseShortname {
			shortName = strings.ToLower(shortName)
		}
		fileExtension := e.fileExtension
		if e.lowercaseExtension {
			shortName = strings.ToLower(fileExtension)
		}
		if fileExtension != "" {
			shortName = fmt.Sprintf("%s.%s", shortName, fileExtension)
		}
		ret[i] = FileInfo{
			modTime:   e.modifyTime,
			name:      e.filenameLong,
			shortName: shortName,
			size:      int64(e.fileSize),
			isDir:     e.isSubdirectory,
		}
	}
	return ret, nil
}

// OpenFile returns an io.ReadWriter from which you can read the contents of a file
// or write contents to the file
//
// accepts normal os.OpenFile flags
//
// returns an error if the file does not exist
func (fs *FileSystem) OpenFile(p string, flag int) (filesystem.File, error) {
	// get the path
	dir := path.Dir(p)
	filename := path.Base(p)
	// if the dir == filename, then it is just /
	if dir == filename {
		return nil, fmt.Errorf("Cannot open directory %s as file", p)
	}
	// get the directory entries
	parentDir, entries, err := fs.readDirWithMkdir(dir, false)
	if err != nil {
		return nil, fmt.Errorf("Could not read directory entries for %s", dir)
	}
	// we now know that the directory exists, see if the file exists
	var targetEntry *directoryEntry
	for _, e := range entries {
		shortName := e.filenameShort
		if e.fileExtension != "" {
			shortName += "." + e.fileExtension
		}
		if e.filenameLong != filename && shortName != filename {
			continue
		}
		// cannot do anything with directories
		if e.isSubdirectory {
			return nil, fmt.Errorf("Cannot open directory %s as file", p)
		}
		// if we got this far, we have found the file
		targetEntry = e
	}

	// see if the file exists
	// if the file does not exist, and is not opened for os.O_CREATE, return an error
	if targetEntry == nil {
		if flag&os.O_CREATE == 0 {
			return nil, fmt.Errorf("Target file %s does not exist and was not asked to create", p)
		}
		// else create it
		targetEntry, err = fs.mkFile(parentDir, filename)
		if err != nil {
			return nil, fmt.Errorf("failed to create file %s: %v", p, err)
		}
		// write the directory entries to disk
		err = fs.writeDirectoryEntries(parentDir)
		if err != nil {
			return nil, fmt.Errorf("Error writing directory file %s to disk: %v", p, err)
		}

	}
	offset := int64(0)
	if flag&os.O_APPEND == os.O_APPEND {
		offset = int64(targetEntry.fileSize)
	}
	return &File{
		directoryEntry: targetEntry,
		isReadWrite:    flag&os.O_RDWR != 0,
		isAppend:       flag&os.O_APPEND != 0,
		offset:         offset,
		filesystem:     fs,
		parent:         parentDir,
	}, nil
}

// Label get the label of the filesystem
func (fs *FileSystem) Label() string {
	// be sane about everything existing
	bpb := fs.bootSector.biosParameterBlock
	if bpb == nil {
		return ""
	}
	return bpb.volumeLabel
}

// read directory entries for a given cluster
func (fs *FileSystem) getClusterList(firstCluster uint32) ([]uint32, error) {
	// first, get the chain of clusters
	complete := false
	cluster := firstCluster
	clusters := fs.table.clusters

	// do we even have a valid cluster?
	if _, ok := clusters[cluster]; !ok {
		return nil, fmt.Errorf("Invalid start cluster: %d", cluster)
	}

	clusterList := make([]uint32, 0, 5)
	for !complete {
		// save the current cluster
		clusterList = append(clusterList, cluster)
		// get the next cluster
		newCluster := clusters[cluster]
		// if it is EOC, we are done
		switch {
		case fs.table.isEoc(newCluster):
			complete = true
		case cluster <= 2:
			return nil, fmt.Errorf("Invalid cluster chain at %d", cluster)
		}
		cluster = newCluster
	}
	return clusterList, nil
}

// read directory entries for a given cluster
func (fs *FileSystem) readDirectory(dir *Directory) ([]*directoryEntry, error) {
	clusterList, err := fs.getClusterList(dir.clusterLocation)
	if err != nil {
		return nil, fmt.Errorf("Could not read cluster list: %v", err)
	}
	// read the data from all of the cluster entries in the list
	byteCount := len(clusterList) * fs.bytesPerCluster
	b := make([]byte, 0, byteCount)
	for _, cluster := range clusterList {
		// bytes where the cluster starts
		clusterStart := fs.start + int64(fs.dataStart) + int64(cluster-2)*int64(fs.bytesPerCluster)
		// length of cluster in bytes
		tmpb := make([]byte, fs.bytesPerCluster, fs.bytesPerCluster)
		// read the entire cluster
		fs.file.ReadAt(tmpb, clusterStart)
		b = append(b, tmpb...)
	}
	// get the directory
	err = dir.entriesFromBytes(b, fs)
	if err != nil {
		return nil, err
	}
	return dir.entries, nil
}

// make a subdirectory
func (fs *FileSystem) mkSubdir(parent *Directory, name string) (*directoryEntry, error) {
	// get a cluster chain for the file
	clusters, err := fs.allocateSpace(1, 0)
	if err != nil {
		return nil, fmt.Errorf("Could not allocate disk space for file %s: %v", name, err)
	}
	// create a directory entry for the file
	return parent.createEntry(name, clusters[0], true)
}

func (fs *FileSystem) writeDirectoryEntries(dir *Directory) error {
	// we need to save the entries of theparent
	b, err := dir.entriesToBytes(fs.bytesPerCluster)
	if err != nil {
		return fmt.Errorf("Could not create a valid byte stream for a FAT32 Entries: %v", err)
	}
	// now have to expand with zeros to the a multiple of cluster lengths
	// how many clusters do we need, how many do we have?
	clusterList, err := fs.getClusterList(dir.clusterLocation)
	if err != nil {
		return fmt.Errorf("Unable to get clusters for directory: %v", err)
	}
	extraClusters := len(b)/(int(fs.bootSector.biosParameterBlock.dos331BPB.dos20BPB.sectorsPerCluster)*fs.bytesPerCluster) - len(clusterList)
	if extraClusters > 0 {
		clusters, err := fs.allocateSpace(uint64(extraClusters), clusterList[len(clusterList)-1])
		if err != nil {
			return fmt.Errorf("Unable to allocate space for directory entries: %v", err)
		}
		clusterList = clusters
	}
	// now write everything out to the cluster list
	// read the data from all of the cluster entries in the list
	for i, cluster := range clusterList {
		// bytes where the cluster starts
		clusterStart := fs.start + int64(fs.dataStart) + int64(cluster-2)*int64(fs.bytesPerCluster)
		bStart := i * fs.bytesPerCluster
		written, err := fs.file.WriteAt(b[bStart:bStart+fs.bytesPerCluster], clusterStart)
		if err != nil {
			return fmt.Errorf("Error writing directory entries: %v", err)
		}
		if written != fs.bytesPerCluster {
			return fmt.Errorf("Wrote %d bytes to cluster %d instead of expected %d", written, cluster, fs.bytesPerCluster)
		}
	}
	return nil
}

// mkFile make a file in a directory
func (fs *FileSystem) mkFile(parent *Directory, name string) (*directoryEntry, error) {
	// get a cluster chain for the file
	clusters, err := fs.allocateSpace(1, 0)
	if err != nil {
		return nil, fmt.Errorf("Could not allocate disk space for directory %s: %v", name, err)
	}
	// create a directory entry for the file
	return parent.createEntry(name, clusters[0], false)
}

// mkLabel make a volume label in a directory
func (fs *FileSystem) mkLabel(parent *Directory, name string) (*directoryEntry, error) {
	// create a directory entry for the file
	return parent.createVolumeLabel(name)
}

// readDirWithMkdir - walks down a directory tree to the last entry
// if it does not exist, it may or may not make it
func (fs *FileSystem) readDirWithMkdir(p string, doMake bool) (*Directory, []*directoryEntry, error) {
	paths, err := splitPath(p)

	if err != nil {
		return nil, nil, err
	}
	// walk down the directory tree until all paths have been walked or we cannot find something
	// start with the root directory
	var entries []*directoryEntry
	currentDir := &Directory{
		directoryEntry: directoryEntry{
			clusterLocation: uint32(fs.table.rootDirCluster),
			isSubdirectory:  true,
			filesystem:      fs,
		},
	}
	entries, err = fs.readDirectory(currentDir)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to read directory %s", "/")
	}
	for i, subp := range paths {
		// do we have an entry whose name is the same as this name?
		found := false
		for _, e := range entries {
			if e.filenameLong != subp && e.filenameShort != subp && (!e.lowercaseShortname || (e.lowercaseShortname && strings.ToLower(e.filenameShort) != subp)) {
				continue
			}
			if !e.isSubdirectory {
				return nil, nil, fmt.Errorf("Cannot create directory at %s since it is a file", "/"+strings.Join(paths[0:i+1], "/"))
			}
			// the filename matches, and it is a subdirectory, so we can break after saving the cluster
			found = true
			currentDir = &Directory{
				directoryEntry: *e,
			}
			break

		}

		// if not, either make it, retrieve its cluster and entries, and loop;
		//  or error out
		if !found {
			if doMake {
				var subdirEntry *directoryEntry
				subdirEntry, err = fs.mkSubdir(currentDir, subp)
				if err != nil {
					return nil, nil, fmt.Errorf("Failed to create subdirectory %s", "/"+strings.Join(paths[0:i+1], "/"))
				}
				// make a basic entry for the new subdir
				dir := &Directory{
					directoryEntry: directoryEntry{clusterLocation: subdirEntry.clusterLocation},
					entries: []*directoryEntry{
						{filenameShort: ".", isSubdirectory: true, clusterLocation: subdirEntry.clusterLocation},
						{filenameShort: "..", isSubdirectory: true, clusterLocation: currentDir.clusterLocation},
					},
				}
				// write the new directory entries to disk
				err = fs.writeDirectoryEntries(dir)
				if err != nil {
					return nil, nil, fmt.Errorf("Error writing new directory entries to disk: %v", err)
				}
				// write the parent directory entries to disk
				err = fs.writeDirectoryEntries(currentDir)
				if err != nil {
					return nil, nil, fmt.Errorf("Error writing directory entries to disk: %v", err)
				}
				// save where we are to search next
				currentDir = &Directory{
					directoryEntry: *subdirEntry,
				}
			} else {
				return nil, nil, fmt.Errorf("Path %s not found", "/"+strings.Join(paths[0:i+1], "/"))
			}
		}
		// get all of the entries in this directory
		entries, err = fs.readDirectory(currentDir)
		if err != nil {
			return nil, nil, fmt.Errorf("Failed to read directory %s", "/"+strings.Join(paths[0:i+1], "/"))
		}
	}
	// once we have made it here, looping is done; we have found the final entry
	return currentDir, entries, nil
}

// allocateSpace ensure that a cluster chain exists to handle a file of a given size.
// arguments are file size in bytes and starting cluster of the chain
// if starting is 0, then we are not (re)sizing an existing chain but creating a new one
// returns the indexes of clusters to be used in order
func (fs *FileSystem) allocateSpace(size uint64, previous uint32) ([]uint32, error) {
	var (
		clusters []uint32
		err      error
	)
	// 1- calculate how many clusters needed
	// 2- see how many clusters already are allocated
	// 3- if needed, allocate new clusters and extend the chain in the FAT table
	keys := make([]uint32, 0, 20)
	allocated := make([]uint32, 0, 20)

	// what is the total count of clusters needed?
	count := int(size / uint64(fs.bytesPerCluster))
	if size%uint64(fs.bytesPerCluster) > 0 {
		count++
	}
	extraClusterCount := count

	clusters = make([]uint32, 0, 20)

	// are we extending an existing chain, or creating a new one?
	if previous >= 2 {
		clusters, err = fs.getClusterList(previous)
		if err != nil {
			return nil, fmt.Errorf("Unable to get cluster list: %v", err)
		}
		originalClusterCount := len(clusters)
		extraClusterCount = count - originalClusterCount
	}

	// what id we do not need to allocate any?
	if extraClusterCount < 1 {
		return clusters, nil
	}

	// get a list of allocated clusters, so we can know which ones are unallocated and therefore allocatable
	allClusters := fs.table.clusters
	maxCluster := fs.table.maxCluster
	for k := range allClusters {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for i := uint32(2); i < maxCluster && len(allocated) < extraClusterCount; i++ {
		if _, ok := allClusters[i]; !ok {
			// these become the same at this point
			allocated = append(allocated, i)
		}
	}

	// did we allocate them all?
	if len(allocated) < extraClusterCount {
		return nil, errors.New("No space left on device")
	}
	// mark last allocated one as EOC
	lastAlloc := len(allocated) - 1

	// extend the chain and fill them in
	if previous > 0 {
		allClusters[previous] = allocated[0]
	}
	for i := 0; i < lastAlloc; i++ {
		allClusters[allocated[i]] = allocated[i+1]
	}
	allClusters[allocated[lastAlloc]] = fs.table.eocMarker

	// update the FSIS
	fs.fsis.lastAllocatedCluster = allocated[len(allocated)-1]
	// write them all
	b, err := fs.table.bytes()
	if err != nil {
		return nil, fmt.Errorf("Error converting FAT table to bytes: %v", err)
	}
	fs.file.WriteAt(b, int64(fs.bootSector.biosParameterBlock.dos331BPB.dos20BPB.reservedSectors)*int64(SectorSize512)+fs.start)

	fsisBytes, err := fs.fsis.toBytes()
	if err != nil {
		return nil, fmt.Errorf("Could not create a valid byte stream for a FAT32 Filesystem Information Sector: %v", err)
	}
	fsisPrimary := fs.bootSector.biosParameterBlock.fsInformationSector
	fsisBackup := fs.bootSector.biosParameterBlock.backupFSInfoSector

	fs.file.WriteAt(fsisBytes, int64(fsisPrimary)*int64(SectorSize512)+fs.start)
	fs.file.WriteAt(fsisBytes, int64(fsisBackup)*int64(SectorSize512)+fs.start)

	// return all of the clusters
	return append(clusters, allocated...), nil
}
