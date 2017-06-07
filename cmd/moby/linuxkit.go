package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
)

var linuxkitYaml = map[string]string{"mkimage": `
kernel:
  image: "linuxkit/kernel:4.9.x"
  cmdline: "console=ttyS0"
init:
  - linuxkit/init:1b8a7e394d2ec2f1fdb4d67645829d1b5bdca037
  - linuxkit/runc:3a4e6cbf15470f62501b019b55e1caac5ee7689f
  - linuxkit/containerd:b1766e4c4c09f63ac4925a6e4612852a93f7e73b
onboot:
  - name: mkimage
    image: "linuxkit/mkimage:5ad60299be03008f29c5caec3c5ea4ac0387aae6"
  - name: poweroff
    image: "linuxkit/poweroff:a8f1e4ad8d459f1fdaad9e4b007512cb3b504ae8"
trust:
  org:
    - linuxkit
`}

func imageFilename(name string) string {
	yaml := linuxkitYaml[name]
	hash := sha256.Sum256([]byte(yaml))
	return filepath.Join(MobyDir, "linuxkit", name+"-"+fmt.Sprintf("%x", hash))
}

func ensureLinuxkitImage(name string) error {
	filename := imageFilename(name)
	_, err1 := os.Stat(filename + "-kernel")
	_, err2 := os.Stat(filename + "-initrd.img")
	_, err3 := os.Stat(filename + "-cmdline")
	if err1 == nil && err2 == nil && err3 == nil {
		return nil
	}
	err := os.MkdirAll(filepath.Join(MobyDir, "linuxkit"), 0755)
	if err != nil {
		return err
	}
	// TODO clean up old files
	log.Infof("Building LinuxKit image %s to generate output formats", name)

	yaml := linuxkitYaml[name]

	m, err := NewConfig([]byte(yaml))
	if err != nil {
		return err
	}
	// TODO pass through --pull to here
	buf := new(bytes.Buffer)
	buildInternal(m, buf, false, nil)
	image := buf.Bytes()
	kernel, initrd, cmdline, err := tarToInitrd(image)
	if err != nil {
		return fmt.Errorf("Error converting to initrd: %v", err)
	}
	err = writeKernelInitrd(filename, kernel, initrd, cmdline)
	if err != nil {
		return err
	}

	return nil
}

func writeKernelInitrd(filename string, kernel []byte, initrd []byte, cmdline string) error {
	err := ioutil.WriteFile(filename+"-kernel", kernel, 0600)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename+"-initrd.img", initrd, 0600)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename+"-cmdline", []byte(cmdline), 0600)
	if err != nil {
		return err
	}
	return nil
}

func outputLinuxKit(format string, filename string, kernel []byte, initrd []byte, cmdline string, size int, hyperkit bool) error {
	log.Debugf("output linuxkit generated img: %s %s size %d", format, filename, size)

	tmp, err := ioutil.TempDir(filepath.Join(MobyDir, "tmp"), "moby")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	buf, err := tarInitrdKernel(kernel, initrd, cmdline)
	if err != nil {
		return err
	}

	tardisk := filepath.Join(tmp, "tardisk")
	f, err := os.Create(tardisk)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, buf)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}

	sizeString := fmt.Sprintf("%dM", size)
	_ = os.Remove(filename)
	_, err = os.Stat(filename)
	if err == nil || !os.IsNotExist(err) {
		return fmt.Errorf("Cannot remove existing file [%s]", filename)
	}
	linuxkit, err := exec.LookPath("linuxkit")
	if err != nil {
		return fmt.Errorf("Cannot find linuxkit executable, needed to build %s output type: %v", format, err)
	}
	commandLine := []string{"-q", "run", "qemu", "-disk", fmt.Sprintf("%s,size=%s,format=%s", filename, sizeString, format), "-disk", fmt.Sprintf("%s,format=raw", tardisk), "-kernel", imageFilename("mkimage")}
	// if hyperkit && format == "raw" {
	// TODO support hyperkit
	// }
	log.Debugf("run %s: %v", linuxkit, commandLine)
	cmd := exec.Command(linuxkit, commandLine...)
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
