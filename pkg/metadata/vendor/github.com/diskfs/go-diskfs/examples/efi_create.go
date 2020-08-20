// The following example will create a fully bootable EFI disk image. It assumes you have a bootable EFI file (any modern Linux kernel compiled with `CONFIG_EFI_STUB=y` will work) available.

package examples

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	diskfs "github.com/diskfs/go-diskfs"
	diskpkg "github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/partition/gpt"
)

func CreateEfi(diskImg string) {

	var (
		espSize          int64 = 100 * 1024 * 1024     // 100 MB
		diskSize         int64 = espSize + 4*1024*1024 // 104 MB
		blkSize          int64 = 512
		partitionStart   int64 = 2048
		partitionSectors int64 = espSize / blkSize
		partitionEnd     int64 = partitionSectors - partitionStart + 1
	)

	// create a disk image
	disk, err := diskfs.Create(diskImg, diskSize, diskfs.Raw)
	if err != nil {
		log.Panic(err)
	}
	// create a partition table
	table := &gpt.Table{
		Partitions: []*gpt.Partition{
			&gpt.Partition{Start: uint64(partitionStart), End: uint64(partitionEnd), Type: gpt.EFISystemPartition, Name: "EFI System"},
		},
	}
	// apply the partition table
	err = disk.Partition(table)

	/*
	 * create an ESP partition with some contents
	 */
	kernel, err := ioutil.ReadFile("/some/kernel/file")

	spec := diskpkg.FilesystemSpec{Partition: 0, FSType: filesystem.TypeFat32}
	fs, err := disk.CreateFilesystem(spec)

	// make our directories
	err = fs.Mkdir("/EFI/BOOT")
	rw, err := fs.OpenFile("/EFI/BOOT/BOOTX64.EFI", os.O_CREATE|os.O_RDWR)

	n, err := rw.Write(kernel)
	if err != nil {
		log.Panic(err)
	}
	fmt.Printf("Wrote %d bytes\n", n)
}
