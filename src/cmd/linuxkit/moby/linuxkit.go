package moby

import (
	"crypto/sha256"
	// fix: #3742
	// golint requires comments on non-main(test)
	// package for blank import
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	log "github.com/sirupsen/logrus"
)

//go:embed mkimage.yaml
var linuxkitYamlStr string
var linuxkitYaml = map[string]string{"mkimage": linuxkitYamlStr}

func imageFilename(name string) string {
	yaml := linuxkitYaml[name]
	hash := sha256.Sum256([]byte(yaml))
	return filepath.Join(MobyDir, "linuxkit", name+"-"+fmt.Sprintf("%x", hash))
}

func ensureLinuxkitImage(name, cache string) error {
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
	// This is just a local utility used for conversion, so it does not matter what architecture we use.
	// Might as well just use our local one.
	arch := runtime.GOARCH
	// TODO pass through --pull to here
	tf, err := os.CreateTemp("", "")
	if err != nil {
		return err
	}
	defer os.Remove(tf.Name())
	if err := Build(m, tf, BuildOpts{Pull: false, BuilderType: "", DecompressKernel: false, CacheDir: cache, DockerCache: true, Arch: arch}); err != nil {
		return err
	}
	if err := tf.Close(); err != nil {
		return err
	}

	image, err := os.Open(tf.Name())
	if err != nil {
		return err
	}
	defer image.Close()
	kernel, initrd, cmdline, _, err := tarToInitrd(image)
	if err != nil {
		return fmt.Errorf("Error converting to initrd: %v", err)
	}
	return writeKernelInitrd(filename, kernel, initrd, cmdline)
}

func writeKernelInitrd(filename string, kernel []byte, initrd []byte, cmdline string) error {
	err := os.WriteFile(filename+"-kernel", kernel, 0600)
	if err != nil {
		return err
	}
	err = os.WriteFile(filename+"-initrd.img", initrd, 0600)
	if err != nil {
		return err
	}
	return os.WriteFile(filename+"-cmdline", []byte(cmdline), 0600)
}

func outputLinuxKit(format string, filename string, kernel []byte, initrd []byte, cmdline string, size int) error {
	log.Debugf("output linuxkit generated img: %s %s size %d", format, filename, size)

	tmp, err := os.MkdirTemp(filepath.Join(MobyDir, "tmp"), "moby")
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
	commandLine := []string{
		"-q", "run", "qemu",
		"--disk", fmt.Sprintf("%s,size=%s,format=%s", filename, sizeString, format),
		"--disk", fmt.Sprintf("%s,format=raw", tardisk),
		"--kernel", imageFilename("mkimage"),
	}
	log.Debugf("run %s: %v", linuxkit, commandLine)
	cmd := exec.Command(linuxkit, commandLine...)
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
