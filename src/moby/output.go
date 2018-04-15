package moby

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	"github.com/moby/tool/src/initrd"
	log "github.com/sirupsen/logrus"
)

var (
	outputImages = map[string]string{
		"iso-bios":    "linuxkit/mkimage-iso-bios:9a51dc64a461f1cc50ba05f30a38f73f5227ac03",
		"iso-efi":     "linuxkit/mkimage-iso-efi:343cf1a8ac0aba7d8a1f13b7f45fa0b57ab897dc",
		"raw-bios":    "linuxkit/mkimage-raw-bios:d90713b2dd610cf9a0f5f9d9095f8bf86f40d5c6",
		"raw-efi":     "linuxkit/mkimage-raw-efi:8938ffb6014543e557b624a40cce1714f30ce4b6",
		"squashfs":    "linuxkit/mkimage-squashfs:b44d00b0a336fd32c122ff32bd2b39c36a965135",
		"gcp":         "linuxkit/mkimage-gcp:e6cdcf859ab06134c0c37a64ed5f886ec8dae1a1",
		"qcow2-efi":   "linuxkit/mkimage-qcow2-efi:787b54906e14a56b9f1da35dcc8e46bd58435285",
		"vhd":         "linuxkit/mkimage-vhd:3820219e5c350fe8ab2ec6a217272ae82f4b9242",
		"dynamic-vhd": "linuxkit/mkimage-dynamic-vhd:743ac9959fe6d3912ebd78b4fd490b117c53f1a6",
		"vmdk":        "linuxkit/mkimage-vmdk:cee81a3ed9c44ae446ef7ebff8c42c1e77b3e1b5",
		"rpi3":        "linuxkit/mkimage-rpi3:0f23c4f37cdca99281ca33ac6188e1942fa7a2b8",
	}
)

// UpdateOutputImages overwrite the docker images used to build the outputs
// 'update' is a map where the key is the output format and the value is a LinuxKit 'mkimage' image.
func UpdateOutputImages(update map[string]string) error {
	for k, img := range update {
		if _, ok := outputImages[k]; !ok {
			return fmt.Errorf("Image format %s is not known", k)
		}
		outputImages[k] = img
	}
	return nil
}

var outFuns = map[string]func(string, io.Reader, int) error{
	"kernel+initrd": func(base string, image io.Reader, size int) error {
		kernel, initrd, cmdline, ucode, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputKernelInitrd(base, kernel, initrd, cmdline, ucode)
		if err != nil {
			return fmt.Errorf("Error writing kernel+initrd output: %v", err)
		}
		return nil
	},
	"tar-kernel-initrd": func(base string, image io.Reader, size int) error {
		kernel, initrd, cmdline, ucode, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		if err := outputKernelInitrdTarball(base, kernel, initrd, cmdline, ucode); err != nil {
			return fmt.Errorf("Error writing kernel+initrd tarball output: %v", err)
		}
		return nil
	},
	"iso-bios": func(base string, image io.Reader, size int) error {
		err := outputIso(outputImages["iso-bios"], base+".iso", image)
		if err != nil {
			return fmt.Errorf("Error writing iso-bios output: %v", err)
		}
		return nil
	},
	"iso-efi": func(base string, image io.Reader, size int) error {
		err := outputIso(outputImages["iso-efi"], base+"-efi.iso", image)
		if err != nil {
			return fmt.Errorf("Error writing iso-efi output: %v", err)
		}
		return nil
	},
	"raw-bios": func(base string, image io.Reader, size int) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		// TODO: Handle ucode
		err = outputImg(outputImages["raw-bios"], base+"-bios.img", kernel, initrd, cmdline)
		if err != nil {
			return fmt.Errorf("Error writing raw-bios output: %v", err)
		}
		return nil
	},
	"raw-efi": func(base string, image io.Reader, size int) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(outputImages["raw-efi"], base+"-efi.img", kernel, initrd, cmdline)
		if err != nil {
			return fmt.Errorf("Error writing raw-efi output: %v", err)
		}
		return nil
	},
	"kernel+squashfs": func(base string, image io.Reader, size int) error {
		err := outputKernelSquashFS(outputImages["squashfs"], base, image)
		if err != nil {
			return fmt.Errorf("Error writing kernel+squashfs output: %v", err)
		}
		return nil
	},
	"aws": func(base string, image io.Reader, size int) error {
		filename := base + ".raw"
		log.Infof("  %s", filename)
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputLinuxKit("raw", filename, kernel, initrd, cmdline, size)
		if err != nil {
			return fmt.Errorf("Error writing raw output: %v", err)
		}
		return nil
	},
	"gcp": func(base string, image io.Reader, size int) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(outputImages["gcp"], base+".img.tar.gz", kernel, initrd, cmdline)
		if err != nil {
			return fmt.Errorf("Error writing gcp output: %v", err)
		}
		return nil
	},
	"qcow2-efi": func(base string, image io.Reader, size int) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(outputImages["qcow2-efi"], base+"-efi.qcow2", kernel, initrd, cmdline)
		if err != nil {
			return fmt.Errorf("Error writing qcow2 EFI output: %v", err)
		}
		return nil
	},
	"qcow2-bios": func(base string, image io.Reader, size int) error {
		filename := base + ".qcow2"
		log.Infof("  %s", filename)
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		// TODO: Handle ucode
		err = outputLinuxKit("qcow2", filename, kernel, initrd, cmdline, size)
		if err != nil {
			return fmt.Errorf("Error writing qcow2 output: %v", err)
		}
		return nil
	},
	"vhd": func(base string, image io.Reader, size int) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(outputImages["vhd"], base+".vhd", kernel, initrd, cmdline)
		if err != nil {
			return fmt.Errorf("Error writing vhd output: %v", err)
		}
		return nil
	},
	"dynamic-vhd": func(base string, image io.Reader, size int) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(outputImages["dynamic-vhd"], base+".vhd", kernel, initrd, cmdline)
		if err != nil {
			return fmt.Errorf("Error writing vhd output: %v", err)
		}
		return nil
	},
	"vmdk": func(base string, image io.Reader, size int) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(outputImages["vmdk"], base+".vmdk", kernel, initrd, cmdline)
		if err != nil {
			return fmt.Errorf("Error writing vmdk output: %v", err)
		}
		return nil
	},
	"rpi3": func(base string, image io.Reader, size int) error {
		if runtime.GOARCH != "arm64" {
			return fmt.Errorf("Raspberry Pi output currently only supported on arm64")
		}
		err := outputRPi3(outputImages["rpi3"], base+".tar", image)
		if err != nil {
			return fmt.Errorf("Error writing rpi3 output: %v", err)
		}
		return nil
	},
}

var prereq = map[string]string{
	"aws":        "mkimage",
	"qcow2-bios": "mkimage",
}

func ensurePrereq(out string) error {
	var err error
	p := prereq[out]
	if p != "" {
		err = ensureLinuxkitImage(p)
	}
	return err
}

// ValidateFormats checks if the format type is known
func ValidateFormats(formats []string) error {
	log.Debugf("validating output: %v", formats)

	for _, o := range formats {
		f := outFuns[o]
		if f == nil {
			return fmt.Errorf("Unknown format type %s", o)
		}
		err := ensurePrereq(o)
		if err != nil {
			return fmt.Errorf("Failed to set up format type %s: %v", o, err)
		}
	}

	return nil
}

// Formats generates all the specified output formats
func Formats(base string, image string, formats []string, size int) error {
	log.Debugf("format: %v %s", formats, base)

	err := ValidateFormats(formats)
	if err != nil {
		return err
	}
	for _, o := range formats {
		ir, err := os.Open(image)
		if err != nil {
			return err
		}
		defer ir.Close()
		f := outFuns[o]
		if err := f(base, ir, size); err != nil {
			return err
		}
	}

	return nil
}

func tarToInitrd(r io.Reader) ([]byte, []byte, string, []byte, error) {
	w := new(bytes.Buffer)
	iw := initrd.NewWriter(w)
	tr := tar.NewReader(r)
	kernel, cmdline, ucode, err := initrd.CopySplitTar(iw, tr)
	if err != nil {
		return []byte{}, []byte{}, "", []byte{}, err
	}
	iw.Close()
	return kernel, w.Bytes(), cmdline, ucode, nil
}

func tarInitrdKernel(kernel, initrd []byte, cmdline string) (*bytes.Buffer, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	hdr := &tar.Header{
		Name: "kernel",
		Mode: 0600,
		Size: int64(len(kernel)),
	}
	err := tw.WriteHeader(hdr)
	if err != nil {
		return buf, err
	}
	_, err = tw.Write(kernel)
	if err != nil {
		return buf, err
	}
	hdr = &tar.Header{
		Name: "initrd.img",
		Mode: 0600,
		Size: int64(len(initrd)),
	}
	err = tw.WriteHeader(hdr)
	if err != nil {
		return buf, err
	}
	_, err = tw.Write(initrd)
	if err != nil {
		return buf, err
	}
	hdr = &tar.Header{
		Name: "cmdline",
		Mode: 0600,
		Size: int64(len(cmdline)),
	}
	err = tw.WriteHeader(hdr)
	if err != nil {
		return buf, err
	}
	_, err = tw.Write([]byte(cmdline))
	if err != nil {
		return buf, err
	}
	return buf, tw.Close()
}

func outputImg(image, filename string, kernel []byte, initrd []byte, cmdline string) error {
	log.Debugf("output img: %s %s", image, filename)
	log.Infof("  %s", filename)
	buf, err := tarInitrdKernel(kernel, initrd, cmdline)
	if err != nil {
		return err
	}
	output, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer output.Close()
	return dockerRun(buf, output, true, image, cmdline)
}

func outputIso(image, filename string, filesystem io.Reader) error {
	log.Debugf("output ISO: %s %s", image, filename)
	log.Infof("  %s", filename)
	output, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer output.Close()
	return dockerRun(filesystem, output, true, image)
}

func outputRPi3(image, filename string, filesystem io.Reader) error {
	log.Debugf("output RPi3: %s %s", image, filename)
	log.Infof("  %s", filename)
	output, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer output.Close()
	return dockerRun(filesystem, output, true, image)
}

func outputKernelInitrd(base string, kernel []byte, initrd []byte, cmdline string, ucode []byte) error {
	log.Debugf("output kernel/initrd: %s %s", base, cmdline)

	if len(ucode) != 0 {
		log.Infof("  %s ucode+%s %s", base+"-kernel", base+"-initrd.img", base+"-cmdline")
		if err := ioutil.WriteFile(base+"-initrd.img", ucode, os.FileMode(0644)); err != nil {
			return err
		}
		f, err := os.OpenFile(base+"-initrd.img", os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		if _, err = f.Write(initrd); err != nil {
			return err
		}
	} else {
		log.Infof("  %s %s %s", base+"-kernel", base+"-initrd.img", base+"-cmdline")
		if err := ioutil.WriteFile(base+"-initrd.img", initrd, os.FileMode(0644)); err != nil {
			return err
		}
	}
	if err := ioutil.WriteFile(base+"-kernel", kernel, os.FileMode(0644)); err != nil {
		return err
	}
	return ioutil.WriteFile(base+"-cmdline", []byte(cmdline), os.FileMode(0644))
}

func outputKernelInitrdTarball(base string, kernel []byte, initrd []byte, cmdline string, ucode []byte) error {
	log.Debugf("output kernel/initrd tarball: %s %s", base, cmdline)
	log.Infof("  %s", base+"-initrd.tar")
	f, err := os.Create(base + "-initrd.tar")
	if err != nil {
		return err
	}
	defer f.Close()
	tw := tar.NewWriter(f)
	hdr := &tar.Header{
		Name: "kernel",
		Mode: 0644,
		Size: int64(len(kernel)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write(kernel); err != nil {
		return err
	}
	hdr = &tar.Header{
		Name: "initrd.img",
		Mode: 0644,
		Size: int64(len(initrd)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write(initrd); err != nil {
		return err
	}
	hdr = &tar.Header{
		Name: "cmdline",
		Mode: 0644,
		Size: int64(len(cmdline)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write([]byte(cmdline)); err != nil {
		return err
	}
	if len(ucode) != 0 {
		hdr := &tar.Header{
			Name: "ucode.cpio",
			Mode: 0644,
			Size: int64(len(ucode)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(ucode); err != nil {
			return err
		}
	}
	return tw.Close()
}

func outputKernelSquashFS(image, base string, filesystem io.Reader) error {
	log.Debugf("output kernel/squashfs: %s %s", image, base)
	log.Infof("  %s-squashfs.img", base)

	tr := tar.NewReader(filesystem)
	buf := new(bytes.Buffer)
	rootfs := tar.NewWriter(buf)

	for {
		var thdr *tar.Header
		thdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		switch {
		case thdr.Name == "boot/kernel":
			kernel, err := ioutil.ReadAll(tr)
			if err != nil {
				return err
			}
			if err := ioutil.WriteFile(base+"-kernel", kernel, os.FileMode(0644)); err != nil {
				return err
			}
		case thdr.Name == "boot/cmdline":
			cmdline, err := ioutil.ReadAll(tr)
			if err != nil {
				return err
			}
			if err := ioutil.WriteFile(base+"-cmdline", cmdline, os.FileMode(0644)); err != nil {
				return err
			}
		case strings.HasPrefix(thdr.Name, "boot/"):
			// skip the rest of boot/
		default:
			rootfs.WriteHeader(thdr)
			if _, err := io.Copy(rootfs, tr); err != nil {
				return err
			}
		}
	}
	rootfs.Close()

	output, err := os.Create(base + "-squashfs.img")
	if err != nil {
		return err
	}
	defer output.Close()

	return dockerRun(buf, output, true, image)
}
