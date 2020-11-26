// The following example will read a filesystem, assuming it to cover the entire image

package examples

import (
	"fmt"
	"log"

	diskfs "github.com/diskfs/go-diskfs"
)

func ReadFilesystem(p string) {
	disk, err := diskfs.Open(p)
	if err != nil {
		log.Panic(err)
	}
	fs, err := disk.GetFilesystem(0) // assuming it is the whole disk, so partition = 0
	if err != nil {
		log.Panic(err)
	}
	files, err := fs.ReadDir("/") // this should list everything
	if err != nil {
		log.Panic(err)
	}
	fmt.Println(files)
}
