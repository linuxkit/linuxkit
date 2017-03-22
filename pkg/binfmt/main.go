package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"syscall"
)

var (
	dir   string
	mount string
)

func init() {
	flag.StringVar(&dir, "dir", "/etc/binfmt.d", "directory with config files")
	flag.StringVar(&mount, "mount", "/proc/sys/fs/binfmt_misc", "binfmt_misc mount point")
}

func binfmt(line []byte) error {
	register := filepath.Join(mount, "register")
	file, err := os.OpenFile(register, os.O_WRONLY, 0)
	if err != nil {
		e, ok := err.(*os.PathError)
		if ok && e.Err == syscall.ENOENT {
			return fmt.Errorf("ENOENT opening %s is it mounted?", register)
		}
		if ok && e.Err == syscall.EPERM {
			return fmt.Errorf("EPERM opening %s check permissions?", register)
		}
		return fmt.Errorf("Cannot open %s: %s", register, err)
	}
	defer file.Close()
	// short writes should not occur on sysfs, cannot usefully recover
	_, err = file.Write(line)
	if err != nil {
		e, ok := err.(*os.PathError)
		if ok && e.Err == syscall.EEXIST {
			// clear existing entry
			split := bytes.SplitN(line[1:], []byte(":"), 2)
			if len(split) == 0 {
				return fmt.Errorf("Cannot determine arch from: %s", line)
			}
			arch := filepath.Join(mount, string(split[0]))
			clear, err := os.OpenFile(arch, os.O_WRONLY, 0)
			if err != nil {
				return fmt.Errorf("Cannot open %s: %s", arch, err)
			}
			defer clear.Close()
			_, err = clear.Write([]byte("-1"))
			if err != nil {
				return fmt.Errorf("Cannot write to %s: %s", arch, err)
			}
			_, err = file.Write(line)
			if err != nil {
				return fmt.Errorf("Cannot write to %s: %s", register, err)
			}
			return nil
		}
		return fmt.Errorf("Cannot write to %s: %s", register, err)
	}
	return nil
}

func main() {
	flag.Parse()

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatalf("Cannot read directory %s: %s", dir, err)
	}

	for _, file := range files {
		contents, err := ioutil.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			log.Fatalf("Cannot read file %s: %s", file.Name(), err)
		}
		lines := bytes.Split(contents, []byte("\n"))
		for _, line := range lines {
			if len(line) == 0 {
				continue
			}
			err = binfmt(line)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
