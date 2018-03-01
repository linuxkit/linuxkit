package moby

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/moby/tool/src/initrd"
	log "github.com/sirupsen/logrus"
)

const (
	isoBios    = "linuxkit/mkimage-iso-bios:3315508388e62f7a599fa5c2d5318e78017ef553"
	isoEfi     = "linuxkit/mkimage-iso-efi:6afada67184c7f68add9562375c662a4559eaa18"
	rawBios    = "linuxkit/mkimage-raw-bios:31e7ef4ed982bad6ab9ff1f1185514492c325571"
	rawEfi     = "linuxkit/mkimage-raw-efi:82db3af46d299be160590fb1633bbfebc891a927"
	gcp        = "linuxkit/mkimage-gcp:df4f46fbcabcfef84af2ff34ff1ef7e7673bc329"
	vhd        = "linuxkit/mkimage-vhd:796acfc515c22afb8f32d6b5c4bdd456b7f79d8c"
	vmdk       = "linuxkit/mkimage-vmdk:deb9018d06dbb9da29464a4320187ce7e4ae1856"
	dynamicvhd = "linuxkit/mkimage-dynamic-vhd:172fb196713a4aff677b88422026512600b1ca55"
	rpi3       = "linuxkit/mkimage-rpi3:553c6c2d13b7d54f6b73b3b0c1c15f2e47ffb0df"
	qcow2Efi   = "linuxkit/mkimage-qcow2-efi:9bc3de981188da099eaf44cc467f5bbb29c13033"
)

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
		err := outputIso(isoBios, base+".iso", image)
		if err != nil {
			return fmt.Errorf("Error writing iso-bios output: %v", err)
		}
		return nil
	},
	"iso-efi": func(base string, image io.Reader, size int) error {
		err := outputIso(isoEfi, base+"-efi.iso", image)
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
		err = outputImg(rawBios, base+"-bios.img", kernel, initrd, cmdline)
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
		err = outputImg(rawEfi, base+"-efi.img", kernel, initrd, cmdline)
		if err != nil {
			return fmt.Errorf("Error writing raw-efi output: %v", err)
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
		err = outputImg(gcp, base+".img.tar.gz", kernel, initrd, cmdline)
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
		err = outputImg(qcow2Efi, base+"-efi.qcow2", kernel, initrd, cmdline)
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
		err = outputImg(vhd, base+".vhd", kernel, initrd, cmdline)
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
		err = outputImg(dynamicvhd, base+".vhd", kernel, initrd, cmdline)
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
		err = outputImg(vmdk, base+".vmdk", kernel, initrd, cmdline)
		if err != nil {
			return fmt.Errorf("Error writing vmdk output: %v", err)
		}
		return nil
	},
	"rpi3": func(base string, image io.Reader, size int) error {
		if runtime.GOARCH != "arm64" {
			return fmt.Errorf("Raspberry Pi output currently only supported on arm64")
		}
		err := outputRPi3(rpi3, base+".tar", image)
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
