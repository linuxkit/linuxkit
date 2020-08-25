package main

import (
	"os"
	"syscall"
)

// MemoryMappedFile A file and its mapped memory. Linux exclusive
type MemoryMappedFile struct {
	file *os.File
	data *[]byte
}

// OpenMemoryMappedFile opens a file and mmap it
func OpenMemoryMappedFile(path string) (mmf MemoryMappedFile, err error) {
	var (
		stat os.FileInfo
		data []byte
	)
	// can't really use := because name shadowing
	if mmf.file, err = os.Open(path); err != nil {
		return
	}
	if stat, err = mmf.file.Stat(); err != nil {
		return
	}
	// since we work on linux exclusive platform this is fine
	if data, err = syscall.Mmap(int(mmf.file.Fd()), 0, int(stat.Size()), syscall.PROT_READ, syscall.MAP_SHARED); err != nil {
		return
	}
	mmf.data = &data
	return
}

// Close munmap the mmap'd buffer if the file has opened before and then close the file
func (m *MemoryMappedFile) Close() (err error) {
	if m.file != nil {
		if m.data != nil {
			err = syscall.Munmap(*m.data)
			m.data = nil
		}
		if err == nil {
			err = m.file.Close()
			m.file = nil
		}
	}
	return
}
