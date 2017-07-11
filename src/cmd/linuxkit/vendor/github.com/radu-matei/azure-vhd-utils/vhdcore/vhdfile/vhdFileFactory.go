package vhdFile

import (
	"os"
	"path/filepath"

	"github.com/radu-matei/azure-vhd-utils/vhdcore/bat"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/footer"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/header"
	"github.com/radu-matei/azure-vhd-utils/vhdcore/reader"
)

// FileFactory is a type to create VhdFile representing VHD in the local machine
//
type FileFactory struct {
	vhdDir               string       // Path to the directory holding VHD file
	fd                   *os.File     // File descriptor of the VHD file
	parentVhdFileFactory *FileFactory // Reference to the parent VhdFileFactory if this VHD file is parent of a dynamic VHD
	childVhdFileFactory  *FileFactory // Reference to the child VhdFileFactory if this VHD file has dynamic VHD child
}

// Create creates a new VhdFile representing a VHD in the local machine located at vhdPath
//
func (f *FileFactory) Create(vhdPath string) (*VhdFile, error) {
	var err error
	if f.fd, err = os.Open(vhdPath); err != nil {
		f.Dispose(err)
		return nil, err
	}

	f.vhdDir = filepath.Dir(vhdPath)
	fStat, _ := f.fd.Stat()
	file, err := f.CreateFromReaderAtReader(f.fd, fStat.Size())
	if err != nil {
		f.Dispose(err)
		return nil, err
	}

	return file, nil
}

// CreateFromReaderAtReader creates a new VhdFile from a reader.ReadAtReader, which is a reader associated
// with a VHD in the local machine. The parameter size is the size of the VHD in bytes
//
func (f *FileFactory) CreateFromReaderAtReader(r reader.ReadAtReader, size int64) (*VhdFile, error) {
	vhdReader := reader.NewVhdReader(r, size)
	vhdFooter, err := (footer.NewFactory(vhdReader)).Create()
	if err != nil {
		return nil, err
	}

	vhdFile := VhdFile{
		Footer:    vhdFooter,
		VhdReader: vhdReader,
	}

	if vhdFooter.DiskType == footer.DiskTypeFixed {
		return &vhdFile, nil
	}

	// Disk is an expanding type (Dynamic or differencing)
	vhdHeader, err := (header.NewFactory(vhdReader, vhdFooter.HeaderOffset)).Create()
	if err != nil {
		return nil, err
	}
	vhdFile.Header = vhdHeader

	vhdBlockAllocationTable, err := (bat.NewBlockAllocationFactory(vhdReader, vhdHeader)).Create()
	if err != nil {
		return nil, err
	}
	vhdFile.BlockAllocationTable = vhdBlockAllocationTable

	if vhdFooter.DiskType == footer.DiskTypeDynamic {
		return &vhdFile, nil
	}

	var parentPath string
	if f.vhdDir == "." || f.vhdDir == string(os.PathSeparator) {
		parentPath = vhdHeader.ParentPath
	} else {
		parentPath = filepath.Join(parentPath, vhdHeader.ParentLocators.GetRelativeParentPath())
	}

	// Insert a node in the doubly linked list of VhdFileFactory chain.
	f.parentVhdFileFactory = &FileFactory{childVhdFileFactory: f}
	// Set differencing disk parent VhdFile
	vhdFile.Parent, err = f.parentVhdFileFactory.Create(parentPath)
	if err != nil {
		return nil, err
	}

	return &vhdFile, nil
}

// Dispose disposes this instance of VhdFileFactory and VhdFileFactory instances of parent and child
// VHDs
//
func (f *FileFactory) Dispose(err error) {
	if f.fd != nil {
		f.fd.Close()
		f.fd = nil
	}

	if f.parentVhdFileFactory != nil {
		f.parentVhdFileFactory.disposeUp(err)
	}

	if f.childVhdFileFactory != nil {
		f.childVhdFileFactory.disposeDown(err)
	}
}

// Dispose disposes this instance of VhdFileFactory and VhdFileFactory instances of all ancestor VHDs
//
func (f *FileFactory) disposeUp(err error) {
	if f.fd != nil {
		f.fd.Close()
		f.fd = nil
	}

	if f.parentVhdFileFactory != nil {
		f.parentVhdFileFactory.disposeUp(err)
	}
}

// Dispose disposes this instance of VhdFileFactory and VhdFileFactory instances of all descendant VHDs
//
func (f *FileFactory) disposeDown(err error) {
	if f.fd != nil {
		f.fd.Close()
		f.fd = nil
	}

	if f.childVhdFileFactory != nil {
		f.childVhdFileFactory.disposeDown(err)
	}
}
