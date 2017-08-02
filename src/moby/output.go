package moby

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/moby/tool/src/initrd"
)

const (
	bios       = "linuxkit/mkimage-iso-bios:1140a4f96b04d6744160f6e3ae485bf7f7a945a8@sha256:878c7d7162120be1c388fded863eef28908b3ebf1c0751b78193103c10d4f6d1"
	efi        = "linuxkit/mkimage-iso-efi:5c2fc616bde288476a14f4f6dd0d273a66832822@sha256:876ef47ec2b30af40e70f1e98f496206eb430915867c4f9f400e1af47fd58d7c"
	gcp        = "linuxkit/mkimage-gcp:46716b3d3f7aa1a7607a3426fe0ccebc554b14ee@sha256:18d8e0482f65a2481f5b6ba1e7ce77723b246bf13bdb612be5e64df90297940c"
	vhd        = "linuxkit/mkimage-vhd:a04c8480d41ca9cef6b7710bd45a592220c3acb2@sha256:ba373dc8ae5dc72685dbe4b872d8f588bc68b2114abd8bdc6a74d82a2b62cce3"
	vmdk       = "linuxkit/mkimage-vmdk:182b541474ca7965c8e8f987389b651859f760da@sha256:99638c5ddb17614f54c6b8e11bd9d49d1dea9d837f38e0f6c1a5f451085d449b"
	dynamicvhd = "linuxkit/mkimage-dynamic-vhd:a652b15c281499ecefa6a7a47d0f9c56d70ab208@sha256:10e2a9179d48934c864639df895a6efdee34c2865eb574934398209625b297ff"
)

var outFuns = map[string]func(string, []byte, int, bool) error{
	"kernel+initrd": func(base string, image []byte, size int, hyperkit bool) error {
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
	"iso-bios": func(base string, image []byte, size int, hyperkit bool) error {
		err := outputIso(bios, base+".iso", image)
		if err != nil {
			return fmt.Errorf("Error writing iso-bios output: %v", err)
		}
		return nil
	},
	"iso-efi": func(base string, image []byte, size int, hyperkit bool) error {
		kernel, initrd, cmdline, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(efi, base+"-efi.iso", kernel, initrd, cmdline)
		if err != nil {
			return fmt.Errorf("Error writing iso-efi output: %v", err)
		}
		return nil
	},
	"raw": func(base string, image []byte, size int, hyperkit bool) error {
		filename := base + ".raw"
		log.Infof("  %s", filename)
		kernel, initrd, cmdline, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputLinuxKit("raw", filename, kernel, initrd, cmdline, size, hyperkit)
		if err != nil {
			return fmt.Errorf("Error writing raw output: %v", err)
		}
		return nil
	},
	"gcp": func(base string, image []byte, size int, hyperkit bool) error {
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
	"qcow2": func(base string, image []byte, size int, hyperkit bool) error {
		filename := base + ".qcow2"
		log.Infof("  %s", filename)
		kernel, initrd, cmdline, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputLinuxKit("qcow2", filename, kernel, initrd, cmdline, size, hyperkit)
		if err != nil {
			return fmt.Errorf("Error writing qcow2 output: %v", err)
		}
		return nil
	},
	"vhd": func(base string, image []byte, size int, hyperkit bool) error {
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
	"dynamic-vhd": func(base string, image []byte, size int, hyperkit bool) error {
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
	"vmdk": func(base string, image []byte, size int, hyperkit bool) error {
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
}

var prereq = map[string]string{
	"raw":   "mkimage",
	"qcow2": "mkimage",
}

func ensurePrereq(out string) error {
	var err error
	p := prereq[out]
	if p != "" {
		err = ensureLinuxkitImage(p)
	}
	return err
}

// ValidateOutputs checks if the output type is known
func ValidateOutputs(out []string) error {
	log.Debugf("validating output: %v", out)

	for _, o := range out {
		f := outFuns[o]
		if f == nil {
			return fmt.Errorf("Unknown output type %s", o)
		}
		err := ensurePrereq(o)
		if err != nil {
			return fmt.Errorf("Failed to set up output type %s: %v", o, err)
		}
	}

	return nil
}

// Outputs generates all the specified output formats
func Outputs(base string, image []byte, out []string, size int, hyperkit bool) error {
	log.Debugf("output: %v %s", out, base)

	err := ValidateOutputs(out)
	if err != nil {
		return err
	}
	for _, o := range out {
		f := outFuns[o]
		err := f(base, image, size, hyperkit)
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
	err = tw.Close()
	if err != nil {
		return buf, err
	}
	return buf, nil
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
	return dockerRun(buf, output, image, cmdline)
}

// this should replace the other version for types that can specify a size
func outputImgSize(image, filename string, kernel []byte, initrd []byte, cmdline string, size int) error {
	log.Debugf("output img: %s %s size %d", image, filename, size)
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
	if size == 0 {
		return dockerRun(buf, output, image)
	}
	return dockerRun(buf, output, image, fmt.Sprintf("%dM", size))
}

func outputIso(image, filename string, filesystem []byte) error {
	log.Debugf("output ISO: %s %s", image, filename)
	log.Infof("  %s", filename)
	output, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer output.Close()
	return dockerRun(bytes.NewBuffer(filesystem), output, image)
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
	err = ioutil.WriteFile(base+"-cmdline", []byte(cmdline), os.FileMode(0644))
	if err != nil {
		return err
	}
	return nil
}
