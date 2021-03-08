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

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/initrd"
	log "github.com/sirupsen/logrus"
)

var (
	outputImages = map[string]string{
		"iso":         "linuxkit/mkimage-iso:4f1a2476ac515983ade72814cf08624c8968f65f",
		"iso-bios":    "linuxkit/mkimage-iso-bios:ea9a22b705b8201a201609905f7636fba8d061b9",
		"iso-efi":     "linuxkit/mkimage-iso-efi:c62420c8588a1d1440249c2c58f325700d72280f",
		"raw-bios":    "linuxkit/mkimage-raw-bios:4f3041edd9de02ef8f15bd92cc2d1afecb90084b",
		"raw-efi":     "linuxkit/mkimage-raw-efi:9ed69b7ac9e75aef6eebaed787223d9504dd967b",
		"squashfs":    "linuxkit/mkimage-squashfs:a1e99651662cb5781f8485a588ce4c85a75d7c9c",
		"gcp":         "linuxkit/mkimage-gcp:a7416d21d4ef642bb2ba560c8f7651250823546d",
		"qcow2-efi":   "linuxkit/mkimage-qcow2-efi:9a623f72befcaadb560290c29b9fb28f3843545b",
		"vhd":         "linuxkit/mkimage-vhd:4cc60c4f46b07e11c64ba618e46b81fa0096c91f",
		"dynamic-vhd": "linuxkit/mkimage-dynamic-vhd:99b9009ed54a793020d3ce8322a42e0cc06da71a",
		"vmdk":        "linuxkit/mkimage-vmdk:b55ea46297a16d8a4448ce7f5a2df987a9602b27",
		"rpi3":        "linuxkit/mkimage-rpi3:19c5354d6f8f68781adbc9bb62095ebb424222dc",
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

var outFuns = map[string]func(string, io.Reader, int, bool) error{
	"kernel+initrd": func(base string, image io.Reader, size int, trust bool) error {
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
	"tar-kernel-initrd": func(base string, image io.Reader, size int, trust bool) error {
		kernel, initrd, cmdline, ucode, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		if err := outputKernelInitrdTarball(base, kernel, initrd, cmdline, ucode); err != nil {
			return fmt.Errorf("Error writing kernel+initrd tarball output: %v", err)
		}
		return nil
	},
	"iso-bios": func(base string, image io.Reader, size int, trust bool) error {
		err := outputIso(outputImages["iso-bios"], base+".iso", image, trust)
		if err != nil {
			return fmt.Errorf("Error writing iso-bios output: %v", err)
		}
		return nil
	},
	"iso-efi": func(base string, image io.Reader, size int, trust bool) error {
		err := outputIso(outputImages["iso-efi"], base+"-efi.iso", image, trust)
		if err != nil {
			return fmt.Errorf("Error writing iso-efi output: %v", err)
		}
		return nil
	},
	"raw-bios": func(base string, image io.Reader, size int, trust bool) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		// TODO: Handle ucode
		err = outputImg(outputImages["raw-bios"], base+"-bios.img", kernel, initrd, cmdline, trust)
		if err != nil {
			return fmt.Errorf("Error writing raw-bios output: %v", err)
		}
		return nil
	},
	"raw-efi": func(base string, image io.Reader, size int, trust bool) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(outputImages["raw-efi"], base+"-efi.img", kernel, initrd, cmdline, trust)
		if err != nil {
			return fmt.Errorf("Error writing raw-efi output: %v", err)
		}
		return nil
	},
	"kernel+squashfs": func(base string, image io.Reader, size int, trust bool) error {
		err := outputKernelSquashFS(outputImages["squashfs"], base, image, trust)
		if err != nil {
			return fmt.Errorf("Error writing kernel+squashfs output: %v", err)
		}
		return nil
	},
	"kernel+iso": func(base string, image io.Reader, size int, trust bool) error {
		err := outputKernelISO(outputImages["iso"], base, image, trust)
		if err != nil {
			return fmt.Errorf("Error writing kernel+iso output: %v", err)
		}
		return nil
	},
	"aws": func(base string, image io.Reader, size int, trust bool) error {
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
	"gcp": func(base string, image io.Reader, size int, trust bool) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(outputImages["gcp"], base+".img.tar.gz", kernel, initrd, cmdline, trust)
		if err != nil {
			return fmt.Errorf("Error writing gcp output: %v", err)
		}
		return nil
	},
	"qcow2-efi": func(base string, image io.Reader, size int, trust bool) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(outputImages["qcow2-efi"], base+"-efi.qcow2", kernel, initrd, cmdline, trust)
		if err != nil {
			return fmt.Errorf("Error writing qcow2 EFI output: %v", err)
		}
		return nil
	},
	"qcow2-bios": func(base string, image io.Reader, size int, trust bool) error {
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
	"vhd": func(base string, image io.Reader, size int, trust bool) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(outputImages["vhd"], base+".vhd", kernel, initrd, cmdline, trust)
		if err != nil {
			return fmt.Errorf("Error writing vhd output: %v", err)
		}
		return nil
	},
	"dynamic-vhd": func(base string, image io.Reader, size int, trust bool) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(outputImages["dynamic-vhd"], base+".vhd", kernel, initrd, cmdline, trust)
		if err != nil {
			return fmt.Errorf("Error writing vhd output: %v", err)
		}
		return nil
	},
	"vmdk": func(base string, image io.Reader, size int, trust bool) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(outputImages["vmdk"], base+".vmdk", kernel, initrd, cmdline, trust)
		if err != nil {
			return fmt.Errorf("Error writing vmdk output: %v", err)
		}
		return nil
	},
	"rpi3": func(base string, image io.Reader, size int, trust bool) error {
		if runtime.GOARCH != "arm64" {
			return fmt.Errorf("Raspberry Pi output currently only supported on arm64")
		}
		err := outputRPi3(outputImages["rpi3"], base+".tar", image, trust)
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

func ensurePrereq(out, cache string) error {
	var err error
	p := prereq[out]
	if p != "" {
		err = ensureLinuxkitImage(p, cache)
	}
	return err
}

// ValidateFormats checks if the format type is known
func ValidateFormats(formats []string, cache string) error {
	log.Debugf("validating output: %v", formats)

	for _, o := range formats {
		f := outFuns[o]
		if f == nil {
			return fmt.Errorf("Unknown format type %s", o)
		}
		err := ensurePrereq(o, cache)
		if err != nil {
			return fmt.Errorf("Failed to set up format type %s: %v", o, err)
		}
	}

	return nil
}

// Formats generates all the specified output formats
func Formats(base string, image string, formats []string, size int, trust bool, cache string) error {
	log.Debugf("format: %v %s", formats, base)

	err := ValidateFormats(formats, cache)
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
		if err := f(base, ir, size, trust); err != nil {
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
		Name:    "kernel",
		Mode:    0600,
		Size:    int64(len(kernel)),
		ModTime: defaultModTime,
		Format:  tar.FormatPAX,
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
		Name:    "initrd.img",
		Mode:    0600,
		Size:    int64(len(initrd)),
		ModTime: defaultModTime,
		Format:  tar.FormatPAX,
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
		Name:    "cmdline",
		Mode:    0600,
		Size:    int64(len(cmdline)),
		ModTime: defaultModTime,
		Format:  tar.FormatPAX,
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

func outputImg(image, filename string, kernel []byte, initrd []byte, cmdline string, trust bool) error {
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
	return dockerRun(buf, output, trust, image, cmdline)
}

func outputIso(image, filename string, filesystem io.Reader, trust bool) error {
	log.Debugf("output ISO: %s %s", image, filename)
	log.Infof("  %s", filename)
	output, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer output.Close()
	return dockerRun(filesystem, output, trust, image)
}

func outputRPi3(image, filename string, filesystem io.Reader, trust bool) error {
	log.Debugf("output RPi3: %s %s", image, filename)
	log.Infof("  %s", filename)
	output, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer output.Close()
	return dockerRun(filesystem, output, trust, image)
}

func outputKernelInitrd(base string, kernel []byte, initrd []byte, cmdline string, ucode []byte) error {
	log.Debugf("output kernel/initrd: %s %s", base, cmdline)

	if len(ucode) != 0 {
		log.Infof("  %s ucode+%s %s", base+"-kernel", base+"-initrd.img", base+"-cmdline")
		if err := ioutil.WriteFile(base+"-initrd.img", ucode, os.FileMode(0644)); err != nil {
			return err
		}
		if len(initrd) != 0 {
			f, err := os.OpenFile(base+"-initrd.img", os.O_APPEND|os.O_WRONLY, 0644)
			if err != nil {
				return err
			}
			defer f.Close()
			if _, err = f.Write(initrd); err != nil {
				return err
			}
		}
	} else {
		if len(initrd) != 0 {
			log.Infof("  %s %s %s", base+"-kernel", base+"-initrd.img", base+"-cmdline")
			if err := ioutil.WriteFile(base+"-initrd.img", initrd, os.FileMode(0644)); err != nil {
				return err
			}
		}
	}
	if len(kernel) != 0 {
		if err := ioutil.WriteFile(base+"-kernel", kernel, os.FileMode(0644)); err != nil {
			return err
		}
	}
	if len(cmdline) != 0 {
		return ioutil.WriteFile(base+"-cmdline", []byte(cmdline), os.FileMode(0644))
	}
	return nil
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
	if len(kernel) != 0 {
		hdr := &tar.Header{
			Name:    "kernel",
			Mode:    0644,
			Size:    int64(len(kernel)),
			ModTime: defaultModTime,
			Format:  tar.FormatPAX,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(kernel); err != nil {
			return err
		}
	}
	if len(initrd) != 0 {
		hdr := &tar.Header{
			Name:    "initrd.img",
			Mode:    0644,
			Size:    int64(len(initrd)),
			ModTime: defaultModTime,
			Format:  tar.FormatPAX,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write(initrd); err != nil {
			return err
		}
	}
	if len(cmdline) != 0 {
		hdr := &tar.Header{
			Name:    "cmdline",
			Mode:    0644,
			Size:    int64(len(cmdline)),
			ModTime: defaultModTime,
			Format:  tar.FormatPAX,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write([]byte(cmdline)); err != nil {
			return err
		}
	}
	if len(ucode) != 0 {
		hdr := &tar.Header{
			Name:    "ucode.cpio",
			Mode:    0644,
			Size:    int64(len(ucode)),
			ModTime: defaultModTime,
			Format:  tar.FormatPAX,
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

func outputKernelSquashFS(image, base string, filesystem io.Reader, trust bool) error {
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
		thdr.Format = tar.FormatPAX
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

	return dockerRun(buf, output, trust, image)
}

func outputKernelISO(image, base string, filesystem io.Reader, trust bool) error {
	log.Debugf("output kernel/iso: %s %s", image, base)
	log.Infof("  %s.iso", base)

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
		thdr.Format = tar.FormatPAX
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

	output, err := os.Create(base + ".iso")
	if err != nil {
		return err
	}
	defer output.Close()

	return dockerRun(buf, output, trust, image)
}
