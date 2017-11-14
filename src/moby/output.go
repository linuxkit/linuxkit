package moby

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"

	"github.com/moby/tool/src/initrd"
	log "github.com/sirupsen/logrus"
)

const (
	isoBios    = "linuxkit/mkimage-iso-bios:165b051322578cb0c2a4f16253b20f7d2797a502"
	isoEfi     = "linuxkit/mkimage-iso-efi:43592c9b316b036338c4db9f90aada37b48415dc"
	rawBios    = "linuxkit/mkimage-raw-bios:eac4dcb78f837618e7e009060146977c2adabe19"
	rawEfi     = "linuxkit/mkimage-raw-efi:167441e223c1c131dac349deb67966d400066e04"
	gcp        = "linuxkit/mkimage-gcp:d1883809d212ce048f60beb0308a4d2b14c256af"
	vhd        = "linuxkit/mkimage-vhd:2a31f2bc91c1d247160570bd17868075e6c0009a"
	vmdk       = "linuxkit/mkimage-vmdk:df02a4fabd87a82209fbbacebde58c4440d2daf0"
	dynamicvhd = "linuxkit/mkimage-dynamic-vhd:8553167d10c3e8d8603b2566d01bdc0cf5908fa5"
	rpi3       = "linuxkit/mkimage-rpi3:0735656fff247ca978135e3aeb62864adc612180"
)

var outFuns = map[string]func(string, []byte, int) error{
	"kernel+initrd": func(base string, image []byte, size int) error {
		kernel, initrd, cmdline, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputKernelInitrd(base, kernel, initrd, cmdline)
		if err != nil {
			return fmt.Errorf("Error writing kernel+initrd output: %v", err)
		}
		return nil
	},
	"tar-kernel-initrd": func(base string, image []byte, size int) error {
		kernel, initrd, cmdline, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		if err := outputKernelInitrdTarball(base, kernel, initrd, cmdline); err != nil {
			return fmt.Errorf("Error writing kernel+initrd tarball output: %v", err)
		}
		return nil
	},
	"iso-bios": func(base string, image []byte, size int) error {
		err := outputIso(isoBios, base+".iso", image)
		if err != nil {
			return fmt.Errorf("Error writing iso-bios output: %v", err)
		}
		return nil
	},
	"iso-efi": func(base string, image []byte, size int) error {
		err := outputIso(isoEfi, base+"-efi.iso", image)
		if err != nil {
			return fmt.Errorf("Error writing iso-efi output: %v", err)
		}
		return nil
	},
	"raw-bios": func(base string, image []byte, size int) error {
		kernel, initrd, cmdline, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(rawBios, base+"-bios.img", kernel, initrd, cmdline)
		if err != nil {
			return fmt.Errorf("Error writing raw-bios output: %v", err)
		}
		return nil
	},
	"raw-efi": func(base string, image []byte, size int) error {
		kernel, initrd, cmdline, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(rawEfi, base+"-efi.img", kernel, initrd, cmdline)
		if err != nil {
			return fmt.Errorf("Error writing raw-efi output: %v", err)
		}
		return nil
	},
	"aws": func(base string, image []byte, size int) error {
		filename := base + ".raw"
		log.Infof("  %s", filename)
		kernel, initrd, cmdline, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputLinuxKit("raw", filename, kernel, initrd, cmdline, size)
		if err != nil {
			return fmt.Errorf("Error writing raw output: %v", err)
		}
		return nil
	},
	"gcp": func(base string, image []byte, size int) error {
		kernel, initrd, cmdline, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(gcp, base+".img.tar.gz", kernel, initrd, cmdline)
		if err != nil {
			return fmt.Errorf("Error writing gcp output: %v", err)
		}
		return nil
	},
	"qcow2-bios": func(base string, image []byte, size int) error {
		filename := base + ".qcow2"
		log.Infof("  %s", filename)
		kernel, initrd, cmdline, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputLinuxKit("qcow2", filename, kernel, initrd, cmdline, size)
		if err != nil {
			return fmt.Errorf("Error writing qcow2 output: %v", err)
		}
		return nil
	},
	"vhd": func(base string, image []byte, size int) error {
		kernel, initrd, cmdline, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(vhd, base+".vhd", kernel, initrd, cmdline)
		if err != nil {
			return fmt.Errorf("Error writing vhd output: %v", err)
		}
		return nil
	},
	"dynamic-vhd": func(base string, image []byte, size int) error {
		kernel, initrd, cmdline, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(dynamicvhd, base+".vhd", kernel, initrd, cmdline)
		if err != nil {
			return fmt.Errorf("Error writing vhd output: %v", err)
		}
		return nil
	},
	"vmdk": func(base string, image []byte, size int) error {
		kernel, initrd, cmdline, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(vmdk, base+".vmdk", kernel, initrd, cmdline)
		if err != nil {
			return fmt.Errorf("Error writing vmdk output: %v", err)
		}
		return nil
	},
	"rpi3": func(base string, image []byte, size int) error {
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
func Formats(base string, image []byte, formats []string, size int) error {
	log.Debugf("format: %v %s", formats, base)

	err := ValidateFormats(formats)
	if err != nil {
		return err
	}
	for _, o := range formats {
		f := outFuns[o]
		err := f(base, image, size)
		if err != nil {
			return err
		}
	}

	return nil
}

func tarToInitrd(image []byte) ([]byte, []byte, string, error) {
	w := new(bytes.Buffer)
	iw := initrd.NewWriter(w)
	r := bytes.NewReader(image)
	tr := tar.NewReader(r)
	kernel, cmdline, err := initrd.CopySplitTar(iw, tr)
	if err != nil {
		return []byte{}, []byte{}, "", err
	}
	iw.Close()
	return kernel, w.Bytes(), cmdline, nil
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

func outputIso(image, filename string, filesystem []byte) error {
	log.Debugf("output ISO: %s %s", image, filename)
	log.Infof("  %s", filename)
	output, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer output.Close()
	return dockerRun(bytes.NewBuffer(filesystem), output, true, image)
}

func outputRPi3(image, filename string, filesystem []byte) error {
	log.Debugf("output RPi3: %s %s", image, filename)
	log.Infof("  %s", filename)
	output, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer output.Close()
	return dockerRun(bytes.NewBuffer(filesystem), output, true, image)
}

func outputKernelInitrd(base string, kernel []byte, initrd []byte, cmdline string) error {
	log.Debugf("output kernel/initrd: %s %s", base, cmdline)
	log.Infof("  %s %s %s", base+"-kernel", base+"-initrd.img", base+"-cmdline")
	err := ioutil.WriteFile(base+"-initrd.img", initrd, os.FileMode(0644))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(base+"-kernel", kernel, os.FileMode(0644))
	if err != nil {
		return err
	}
	return ioutil.WriteFile(base+"-cmdline", []byte(cmdline), os.FileMode(0644))
}

func outputKernelInitrdTarball(base string, kernel []byte, initrd []byte, cmdline string) error {
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
	return tw.Close()
}
