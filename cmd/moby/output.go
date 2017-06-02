package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/linuxkit/linuxkit/src/initrd"
)

const (
	bios = "linuxkit/mkimage-iso-bios:db791abed6f2b5320feb6cec255a635aee3756f6@sha256:e57483075307bcea4a7257f87eee733d3e24e7a964ba15dcc01111df6729ab3b"
	efi  = "linuxkit/mkimage-iso-efi:5c2fc616bde288476a14f4f6dd0d273a66832822@sha256:876ef47ec2b30af40e70f1e98f496206eb430915867c4f9f400e1af47fd58d7c"
	vhd  = "linuxkit/mkimage-vhd:a04c8480d41ca9cef6b7710bd45a592220c3acb2@sha256:ba373dc8ae5dc72685dbe4b872d8f588bc68b2114abd8bdc6a74d82a2b62cce3"
	vmdk = "linuxkit/mkimage-vmdk:182b541474ca7965c8e8f987389b651859f760da@sha256:99638c5ddb17614f54c6b8e11bd9d49d1dea9d837f38e0f6c1a5f451085d449b"
)

var outFuns = map[string]func(string, []byte, int, bool) error{
	"tar": func(base string, image []byte, size int, hyperkit bool) error {
		err := outputTar(base, image)
		if err != nil {
			return fmt.Errorf("Error writing tar output: %v", err)
		}
		return nil
	},
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
		kernel, initrd, cmdline, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputImg(bios, base+".iso", kernel, initrd, cmdline)
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
	"img": func(base string, image []byte, size int, hyperkit bool) error {
		filename := base + ".img"
		log.Infof("  %s", filename)
		kernel, initrd, cmdline, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		err = outputLinuxKit("raw", filename, kernel, initrd, cmdline, size, hyperkit)
		if err != nil {
			return fmt.Errorf("Error writing qcow2 output: %v", err)
		}
		return nil
	},
	"img-gz": func(base string, image []byte, size int, hyperkit bool) error {
		filename := base + ".img.gz"
		log.Infof("  %s", filename)
		kernel, initrd, cmdline, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		tmp, err := ioutil.TempDir(filepath.Join(MobyDir, "tmp"), "img-gz")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)
		err = outputLinuxKit("raw", filepath.Join(tmp, "uncompressed.img"), kernel, initrd, cmdline, size, hyperkit)
		if err != nil {
			return fmt.Errorf("Error writing img-gz output: %v", err)
		}
		out, err := os.Create(filename)
		if err != nil {
			return err
		}
		in, err := os.Open(filepath.Join(tmp, "uncompressed.img"))
		if err != nil {
			return err
		}
		zw := gzip.NewWriter(out)
		io.Copy(zw, in)
		err = zw.Close()
		if err != nil {
			return err
		}
		err = in.Close()
		if err != nil {
			return err
		}
		err = out.Close()
		if err != nil {
			return err
		}
		return nil
	},
	"gcp-img": func(base string, image []byte, size int, hyperkit bool) error {
		filename := base + ".img.tar.gz"
		log.Infof("  %s", filename)
		kernel, initrd, cmdline, err := tarToInitrd(image)
		if err != nil {
			return fmt.Errorf("Error converting to initrd: %v", err)
		}
		tmp, err := ioutil.TempDir(filepath.Join(MobyDir, "tmp"), "gcp-img")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)
		err = outputLinuxKit("raw", filepath.Join(tmp, "disk.raw"), kernel, initrd, cmdline, size, hyperkit)
		if err != nil {
			return fmt.Errorf("Error writing gcp-img output: %v", err)
		}
		out, err := os.Create(filename)
		if err != nil {
			return err
		}
		in, err := os.Open(filepath.Join(tmp, "disk.raw"))
		if err != nil {
			return err
		}
		fi, err := in.Stat()
		if err != nil {
			return err
		}
		zw := gzip.NewWriter(out)
		tw := tar.NewWriter(zw)
		hdr := &tar.Header{
			Name: "disk.raw",
			Mode: 0600,
			Size: fi.Size(),
		}
		err = tw.WriteHeader(hdr)
		if err != nil {
			return err
		}
		io.Copy(tw, in)
		err = tw.Close()
		if err != nil {
			return err
		}
		err = zw.Close()
		if err != nil {
			return err
		}
		err = in.Close()
		if err != nil {
			return err
		}
		err = out.Close()
		if err != nil {
			return err
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
			return fmt.Errorf("Error writingvhd output: %v", err)
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
	"img":     "mkimage",
	"img-gz":  "mkimage",
	"gcp-img": "mkimage",
	"qcow2":   "mkimage",
}

func ensurePrereq(out string) error {
	var err error
	p := prereq[out]
	if p != "" {
		err = ensureLinuxkitImage(p)
	}
	return err
}

func validateOutputs(out outputList) error {
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

func outputs(base string, image []byte, out outputList, size int, hyperkit bool) error {
	log.Debugf("output: %v %s", out, base)

	err := validateOutputs(out)
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
	img, err := dockerRunInput(buf, image, cmdline)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename, img, os.FileMode(0644))
	if err != nil {
		return err
	}
	return nil
}

// this should replace the other version for types that can specify a size
func outputImgSize(image, filename string, kernel []byte, initrd []byte, cmdline string, size int) error {
	log.Debugf("output img: %s %s size %d", image, filename, size)
	log.Infof("  %s", filename)
	buf, err := tarInitrdKernel(kernel, initrd, cmdline)
	if err != nil {
		return err
	}
	var img []byte
	if size == 0 {
		img, err = dockerRunInput(buf, image)
	} else {
		img, err = dockerRunInput(buf, image, fmt.Sprintf("%dM", size))
	}
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename, img, os.FileMode(0644))
	if err != nil {
		return err
	}
	return nil
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

func outputTar(base string, initrd []byte) error {
	log.Debugf("output tar: %s", base)
	log.Infof("  %s", base+".tar")
	return ioutil.WriteFile(base+".tar", initrd, os.FileMode(0644))
}
