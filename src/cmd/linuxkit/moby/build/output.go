package build

import (
	"archive/tar"
	"bytes"

	// fix: #3742
	// golint requires comments on non-main(test)
	// package for blank import
	_ "embed"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/initrd"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

//go:embed images.yaml
var imagesBytes []byte
var outputImages map[string]string

var outFuns = map[string]func(base string, ir io.Reader, size int, arch string) error{
	"kernel+initrd": func(base string, image io.Reader, size int, arch string) error {
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
	"tar-kernel-initrd": func(base string, image io.Reader, size int, arch string) error {
		kernel, initrd, cmdline, ucode, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		if err := outputKernelInitrdTarball(base, kernel, initrd, cmdline, ucode); err != nil {
			return fmt.Errorf("Error writing kernel+initrd tarball output: %v", err)
		}
		return nil
	},
	"iso-bios": func(base string, image io.Reader, size int, arch string) error {
		err := outputIso(outputImages["iso-bios"], base+".iso", image, arch)
		if err != nil {
			return fmt.Errorf("Error writing iso-bios output: %v", err)
		}
		return nil
	},
	"iso-efi": func(base string, image io.Reader, size int, arch string) error {
		err := outputIso(outputImages["iso-efi"], base+"-efi.iso", image, arch)
		if err != nil {
			return fmt.Errorf("Error writing iso-efi output: %v", err)
		}
		return nil
	},
	"iso-efi-initrd": func(base string, image io.Reader, size int, arch string) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}

		err = outputImg(outputImages["iso-efi-initrd"], base+"-efi-initrd.iso", kernel, initrd, cmdline, arch)
		if err != nil {
			return fmt.Errorf("Error writing iso-efi-initrd output: %v", err)
		}
		return nil
	},
	"raw-bios": func(base string, image io.Reader, size int, arch string) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		// TODO: Handle ucode
		err = outputImg(outputImages["raw-bios"], base+"-bios.img", kernel, initrd, cmdline, arch)
		if err != nil {
			return fmt.Errorf("Error writing raw-bios output: %v", err)
		}
		return nil
	},
	"raw-efi": func(base string, image io.Reader, size int, arch string) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(outputImages["raw-efi"], base+"-efi.img", kernel, initrd, cmdline, arch)
		if err != nil {
			return fmt.Errorf("Error writing raw-efi output: %v", err)
		}
		return nil
	},
	"kernel+squashfs": func(base string, image io.Reader, size int, arch string) error {
		err := outputKernelSquashFS(outputImages["squashfs"], base, image, arch)
		if err != nil {
			return fmt.Errorf("Error writing kernel+squashfs output: %v", err)
		}
		return nil
	},
	"kernel+iso": func(base string, image io.Reader, size int, arch string) error {
		err := outputKernelISO(outputImages["iso"], base, image, arch)
		if err != nil {
			return fmt.Errorf("Error writing kernel+iso output: %v", err)
		}
		return nil
	},
	"aws": func(base string, image io.Reader, size int, arch string) error {
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
	"gcp": func(base string, image io.Reader, size int, arch string) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(outputImages["gcp"], base+".img.tar.gz", kernel, initrd, cmdline, arch)
		if err != nil {
			return fmt.Errorf("Error writing gcp output: %v", err)
		}
		return nil
	},
	"qcow2-efi": func(base string, image io.Reader, size int, arch string) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(outputImages["qcow2-efi"], base+"-efi.qcow2", kernel, initrd, cmdline, arch)
		if err != nil {
			return fmt.Errorf("Error writing qcow2 EFI output: %v", err)
		}
		return nil
	},
	"qcow2-bios": func(base string, image io.Reader, size int, arch string) error {
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
	"vhd": func(base string, image io.Reader, size int, arch string) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(outputImages["vhd"], base+".vhd", kernel, initrd, cmdline, arch)
		if err != nil {
			return fmt.Errorf("Error writing vhd output: %v", err)
		}
		return nil
	},
	"dynamic-vhd": func(base string, image io.Reader, size int, arch string) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(outputImages["dynamic-vhd"], base+".vhd", kernel, initrd, cmdline, arch)
		if err != nil {
			return fmt.Errorf("Error writing vhd output: %v", err)
		}
		return nil
	},
	"vmdk": func(base string, image io.Reader, size int, arch string) error {
		kernel, initrd, cmdline, _, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(outputImages["vmdk"], base+".vmdk", kernel, initrd, cmdline, arch)
		if err != nil {
			return fmt.Errorf("Error writing vmdk output: %v", err)
		}
		return nil
	},
	"rpi3": func(base string, image io.Reader, size int, arch string) error {
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

func ensurePrereq(out, cache string) error {
	var err error
	p := prereq[out]
	if p != "" {
		err = ensureLinuxkitImage(p, cache)
	}
	return err
}

// parseOutputImages parse the raw output image info
func parseOutputImages(b []byte) (map[string]string, error) {
	var (
		m map[string]string
	)
	err := yaml.Unmarshal(b, &m)
	return m, err
}

// ValidateFormats checks if the format type is known
func ValidateFormats(formats []string, cache string) error {
	log.Debugf("validating output: %v", formats)
	if outputImages == nil {
		var err error
		outputImages, err = parseOutputImages(imagesBytes)
		if err != nil {
			return err
		}
	}

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
func Formats(base string, image string, formats []string, size int, arch, cache string) error {
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
		if err := f(base, ir, size, arch); err != nil {
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

func outputImg(image, filename string, kernel []byte, initrd []byte, cmdline, arch string) error {
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
	march, err := util.MArch(arch)
	if err != nil {
		return err
	}
	return dockerRun(buf, output, image, []string{fmt.Sprintf("TARGETARCH=%s", march)}, cmdline)
}

func outputIso(image, filename string, filesystem io.Reader, arch string) error {
	log.Debugf("output ISO: %s %s", image, filename)
	log.Infof("  %s", filename)
	output, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer output.Close()
	march, err := util.MArch(arch)
	if err != nil {
		return err
	}
	return dockerRun(filesystem, output, image, []string{fmt.Sprintf("TARGETARCH=%s", march)})
}

func outputRPi3(image, filename string, filesystem io.Reader) error {
	log.Debugf("output RPi3: %s %s", image, filename)
	log.Infof("  %s", filename)
	output, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer output.Close()
	return dockerRun(filesystem, output, image, nil)
}

func outputKernelInitrd(base string, kernel []byte, initrd []byte, cmdline string, ucode []byte) error {
	log.Debugf("output kernel/initrd: %s %s", base, cmdline)

	if len(ucode) != 0 {
		log.Infof("  %s ucode+%s %s", base+"-kernel", base+"-initrd.img", base+"-cmdline")
		if err := os.WriteFile(base+"-initrd.img", ucode, os.FileMode(0644)); err != nil {
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
			if err := os.WriteFile(base+"-initrd.img", initrd, os.FileMode(0644)); err != nil {
				return err
			}
		}
	}
	if len(kernel) != 0 {
		if err := os.WriteFile(base+"-kernel", kernel, os.FileMode(0644)); err != nil {
			return err
		}
	}
	if len(cmdline) != 0 {
		return os.WriteFile(base+"-cmdline", []byte(cmdline), os.FileMode(0644))
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

func outputKernelSquashFS(image, base string, filesystem io.Reader, arch string) error {
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
			kernel, err := io.ReadAll(tr)
			if err != nil {
				return err
			}
			if err := os.WriteFile(base+"-kernel", kernel, os.FileMode(0644)); err != nil {
				return err
			}
		case thdr.Name == "boot/cmdline":
			cmdline, err := io.ReadAll(tr)
			if err != nil {
				return err
			}
			if err := os.WriteFile(base+"-cmdline", cmdline, os.FileMode(0644)); err != nil {
				return err
			}
		case strings.HasPrefix(thdr.Name, "boot/"):
			// skip the rest of boot/
		default:
			_ = rootfs.WriteHeader(thdr)
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

	march, err := util.MArch(arch)
	if err != nil {
		return err
	}
	return dockerRun(buf, output, image, []string{fmt.Sprintf("TARGETARCH=%s", march)})
}

func outputKernelISO(image, base string, filesystem io.Reader, arch string) error {
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
			kernel, err := io.ReadAll(tr)
			if err != nil {
				return err
			}
			if err := os.WriteFile(base+"-kernel", kernel, os.FileMode(0644)); err != nil {
				return err
			}
		case thdr.Name == "boot/cmdline":
			cmdline, err := io.ReadAll(tr)
			if err != nil {
				return err
			}
			if err := os.WriteFile(base+"-cmdline", cmdline, os.FileMode(0644)); err != nil {
				return err
			}
		case strings.HasPrefix(thdr.Name, "boot/"):
			// skip the rest of boot/
		default:
			_ = rootfs.WriteHeader(thdr)
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

	march, err := util.MArch(arch)
	if err != nil {
		return err
	}
	return dockerRun(buf, output, image, []string{fmt.Sprintf("TARGETARCH=%s", march)})
}
