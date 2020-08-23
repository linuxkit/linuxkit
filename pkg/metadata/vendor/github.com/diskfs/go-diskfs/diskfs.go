// Package diskfs implements methods for creating and manipulating disks and filesystems
//
// methods for creating and manipulating disks and filesystems, whether block devices
// in /dev or direct disk images. This does **not**
// mount any disks or filesystems, neither directly locally nor via a VM. Instead, it manipulates the
// bytes directly.
//
// This is not intended as a replacement for operating system filesystem and disk drivers. Instead,
// it is intended to make it easy to work with partitions, partition tables and filesystems directly
// without requiring operating system mounts.
//
// Some examples:
//
// 1. Create a disk image of size 10MB with a FAT32 filesystem spanning the entire disk.
//
//     import diskfs "github.com/diskfs/go-diskfs"
//     size := 10*1024*1024 // 10 MB
//
//     diskImg := "/tmp/disk.img"
//     disk := diskfs.Create(diskImg, size, diskfs.Raw)
//
//     fs, err := disk.CreateFilesystem(0, diskfs.TypeFat32)
//
// 2. Create a disk of size 20MB with an MBR partition table, a single partition beginning at block 2048 (1MB),
//    of size 10MB filled with a FAT32 filesystem.
//
//     import diskfs "github.com/diskfs/go-diskfs"
//
//     diskSize := 10*1024*1024 // 10 MB
//
//     diskImg := "/tmp/disk.img"
//     disk := diskfs.Create(diskImg, size, diskfs.Raw)
//
//     table := &mbr.Table{
//       LogicalSectorSize:  512,
//       PhysicalSectorSize: 512,
//       Partitions: []*mbr.Partition{
//         {
//           Bootable:      false,
//           Type:          Linux,
//           Start:         2048,
//           Size:          20480,
//         },
//       },
//     }
//
//     fs, err := disk.CreateFilesystem(1, diskfs.TypeFat32)
//
// 3. Create a disk of size 20MB with a GPT partition table, a single partition beginning at block 2048 (1MB),
//    of size 10MB, and fill with the contents from the 10MB file "/root/contents.dat"
//
//     import diskfs "github.com/diskfs/go-diskfs"
//
//     diskSize := 10*1024*1024 // 10 MB
//
//     diskImg := "/tmp/disk.img"
//     disk := diskfs.Create(diskImg, size, diskfs.Raw)
//
//     table := &gpt.Table{
//       LogicalSectorSize:  512,
//       PhysicalSectorSize: 512,
//       Partitions: []*gpt.Partition{
//         {
//           LogicalSectorSize:  512,
//           PhysicalSectorSize: 512,
//           ProtectiveMBR:      true,
//         },
//       },
//     }
//
//     f, err := os.Open("/root/contents.dat")
//     written, err := disk.WritePartitionContents(1, f)
//
// 4. Create a disk of size 20MB with an MBR partition table, a single partition beginning at block 2048 (1MB),
//    of size 10MB filled with a FAT32 filesystem, and create some directories and files in that filesystem.
//
//     import diskfs "github.com/diskfs/go-diskfs"
//
//     diskSize := 10*1024*1024 // 10 MB
//
//     diskImg := "/tmp/disk.img"
//     disk := diskfs.Create(diskImg, size, diskfs.Raw)
//
//     table := &mbr.Table{
//       LogicalSectorSize:  512,
//       PhysicalSectorSize: 512,
//       Partitions: []*mbr.Partition{
//         {
//           Bootable:      false,
//           Type:          Linux,
//           Start:         2048,
//           Size:          20480,
//         },
//       },
//     }
//
//     fs, err := disk.CreateFilesystem(1, diskfs.TypeFat32)
//     err := fs.Mkdir("/FOO/BAR")
//     rw, err := fs.OpenFile("/FOO/BAR/AFILE.EXE", os.O_CREATE|os.O_RDRWR)
//     b := make([]byte, 1024, 1024)
//     rand.Read(b)
//     err := rw.Write(b)
//
package diskfs

import (
	"errors"
	"fmt"
	"io"
	"os"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"

	"github.com/diskfs/go-diskfs/disk"
)

// when we use a disk image with a GPT, we cannot get the logical sector size from the disk via the kernel
//    so we use the default sector size of 512, per Rod Smith
const (
	defaultBlocksize, firstblock int = 512, 2048
	blksszGet                        = 0x1268
	blkbszGet                        = 0x80081270
)

// Format represents the format of the disk
type Format int

const (
	// Raw disk format for basic raw disk
	Raw Format = iota
)

// OpenModeOption represents file open modes
type OpenModeOption int

const (
	// ReadOnly open file in read only mode
	ReadOnly OpenModeOption = iota
	// ReadWriteExclusive open file in read-write exclusive mode
	ReadWriteExclusive
)

// OpenModeOption.String()
func (m OpenModeOption) String() string {
	switch m {
	case ReadOnly:
		return "read-only"
	case ReadWriteExclusive:
		return "read-write exclusive"
	default:
		return "unknown"
	}
}

var openModeOptions = map[OpenModeOption]int{
	ReadOnly:           os.O_RDONLY,
	ReadWriteExclusive: os.O_RDWR | os.O_EXCL,
}

func writableMode(mode OpenModeOption) bool {
	m, ok := openModeOptions[mode]
	if ok {
		if m&os.O_RDWR != 0 || m&os.O_WRONLY != 0 {
			return true
		}
	}

	return false
}

func initDisk(f *os.File, openMode OpenModeOption) (*disk.Disk, error) {
	var (
		diskType      disk.Type
		size          int64
		lblksize      = int64(defaultBlocksize)
		pblksize      = int64(defaultBlocksize)
		defaultBlocks = true
	)
	log.Debug("initDisk(): start")

	// get device information
	devInfo, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("could not get info for device %s: %x", f.Name(), err)
	}
	mode := devInfo.Mode()
	switch {
	case mode.IsRegular():
		log.Debug("initDisk(): regular file")
		diskType = disk.File
		size = devInfo.Size()
		if size <= 0 {
			return nil, fmt.Errorf("could not get file size for device %s", f.Name())
		}
	case mode&os.ModeDevice != 0:
		log.Debug("initDisk(): block device")
		diskType = disk.Device
		file, err := os.Open(f.Name())
		if err != nil {
			return nil, fmt.Errorf("error opening block device %s: %s\n", f.Name(), err)
		}
		size, err = file.Seek(0, io.SeekEnd)
		if err != nil {
			return nil, fmt.Errorf("error seeking to end of block device %s: %s\n", f.Name(), err)
		}
		lblksize, pblksize, err = getSectorSizes(f)
		log.Debugf("initDisk(): logical block size %d, physical block size %d", lblksize, pblksize)
		defaultBlocks = false
		if err != nil {
			return nil, fmt.Errorf("Unable to get block sizes for device %s: %v", f.Name(), err)
		}
	default:
		return nil, fmt.Errorf("device %s is neither a block device nor a regular file", f.Name())
	}

	// how many good blocks do we have?
	//var goodBlocks, orphanedBlocks int
	//goodBlocks = size / lblksize

	writable := writableMode(openMode)

	return &disk.Disk{
		File:              f,
		Info:              devInfo,
		Type:              diskType,
		Size:              size,
		LogicalBlocksize:  lblksize,
		PhysicalBlocksize: pblksize,
		Writable:          writable,
		DefaultBlocks:     defaultBlocks,
	}, nil
}

func checkDevice(device string) error {
	if device == "" {
		return errors.New("must pass device name")
	}
	if _, err := os.Stat(device); os.IsNotExist(err) {
		return fmt.Errorf("provided device %s does not exist", device)
	}

	return nil
}

// Open a Disk from a path to a device in read-write exclusive mode
// Should pass a path to a block device e.g. /dev/sda or a path to a file /tmp/foo.img
// The provided device must exist at the time you call Open()
func Open(device string) (*disk.Disk, error) {
	err := checkDevice(device)
	if err != nil {
		return nil, err
	}
	f, err := os.OpenFile(device, os.O_RDWR|os.O_EXCL, 0600)
	if err != nil {
		return nil, fmt.Errorf("Could not open device %s exclusively for writing", device)
	}
	// return our disk
	return initDisk(f, ReadWriteExclusive)
}

// OpenWithMode open a Disk from a path to a device with a given open mode
// If the device is open in read-only mode, operations to change disk partitioning will
// return an error
// Should pass a path to a block device e.g. /dev/sda or a path to a file /tmp/foo.img
// The provided device must exist at the time you call OpenWithMode()
func OpenWithMode(device string, mode OpenModeOption) (*disk.Disk, error) {
	err := checkDevice(device)
	if err != nil {
		return nil, err
	}
	m, ok := openModeOptions[mode]
	if !ok {
		return nil, errors.New("unsupported file open mode")
	}
	f, err := os.OpenFile(device, m, 0600)
	if err != nil {
		return nil, fmt.Errorf("Could not open device %s with mode %v: %v", device, mode, err)
	}
	// return our disk
	return initDisk(f, mode)
}

// Create a Disk from a path to a device
// Should pass a path to a block device e.g. /dev/sda or a path to a file /tmp/foo.img
// The provided device must not exist at the time you call Create()
func Create(device string, size int64, format Format) (*disk.Disk, error) {
	if device == "" {
		return nil, errors.New("must pass device name")
	}
	if size <= 0 {
		return nil, errors.New("must pass valid device size to create")
	}
	f, err := os.OpenFile(device, os.O_RDWR|os.O_EXCL|os.O_CREATE, 0666)
	if err != nil {
		return nil, fmt.Errorf("Could not create device %s", device)
	}
	err = os.Truncate(device, size)
	if err != nil {
		return nil, fmt.Errorf("Could not expand device %s to size %d", device, size)
	}
	// return our disk
	return initDisk(f, ReadWriteExclusive)
}

// to get the logical and physical sector sizes
func getSectorSizes(f *os.File) (int64, int64, error) {
	/*
		ioctl(fd, BLKBSZGET, &physicalsectsize);

	*/
	fd := f.Fd()
	logicalSectorSize, err := unix.IoctlGetInt(int(fd), blksszGet)
	if err != nil {
		return 0, 0, fmt.Errorf("Unable to get device logical sector size: %v", err)
	}
	physicalSectorSize, err := unix.IoctlGetInt(int(fd), blkbszGet)
	if err != nil {
		return 0, 0, fmt.Errorf("Unable to get device physical sector size: %v", err)
	}
	return int64(logicalSectorSize), int64(physicalSectorSize), nil
}
