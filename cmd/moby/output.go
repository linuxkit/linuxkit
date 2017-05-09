package main

import (
	"archive/tar"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/linuxkit/linuxkit/src/initrd"
)

const (
	bios = "linuxkit/mkimage-iso-bios:db791abed6f2b5320feb6cec255a635aee3756f6@sha256:e57483075307bcea4a7257f87eee733d3e24e7a964ba15dcc01111df6729ab3b"
	efi  = "linuxkit/mkimage-iso-efi:5c2fc616bde288476a14f4f6dd0d273a66832822@sha256:876ef47ec2b30af40e70f1e98f496206eb430915867c4f9f400e1af47fd58d7c"
	gcp  = "linuxkit/mkimage-gcp:46716b3d3f7aa1a7607a3426fe0ccebc554b14ee@sha256:18d8e0482f65a2481f5b6ba1e7ce77723b246bf13bdb612be5e64df90297940c"
	qcow = "linuxkit/mkimage-qcow:69890f35b55e4ff8a2c7a714907f988e57056d02@sha256:f89dc09f82bdbf86d7edae89604544f20b99d99c9b5cabcf1f93308095d8c244"
	vhd  = "linuxkit/mkimage-vhd:a04c8480d41ca9cef6b7710bd45a592220c3acb2@sha256:ba373dc8ae5dc72685dbe4b872d8f588bc68b2114abd8bdc6a74d82a2b62cce3"
	vmdk = "linuxkit/mkimage-vmdk:182b541474ca7965c8e8f987389b651859f760da@sha256:99638c5ddb17614f54c6b8e11bd9d49d1dea9d837f38e0f6c1a5f451085d449b"
)

func outputs(m *Moby, base string, image []byte) error {
	log.Debugf("output: %s %s", m.Outputs, base)

	for _, o := range m.Outputs {
		switch o.Format {
		case "tar":
			err := outputTar(base, image)
			if err != nil {
				return fmt.Errorf("Error writing %s output: %v", o.Format, err)
			}
		case "kernel+initrd":
			kernel, initrd, cmdline, err := tarToInitrd(image)
			if err != nil {
				return fmt.Errorf("Error converting to initrd: %v", err)
			}
			err = outputKernelInitrd(base, kernel, initrd, cmdline)
			if err != nil {
				return fmt.Errorf("Error writing %s output: %v", o.Format, err)
			}
		case "iso-bios":
			kernel, initrd, cmdline, err := tarToInitrd(image)
			if err != nil {
				return fmt.Errorf("Error converting to initrd: %v", err)
			}
			err = outputISO(bios, base+".iso", kernel, initrd, cmdline)
			if err != nil {
				return fmt.Errorf("Error writing %s output: %v", o.Format, err)
			}
		case "iso-efi":
			kernel, initrd, cmdline, err := tarToInitrd(image)
			if err != nil {
				return fmt.Errorf("Error converting to initrd: %v", err)
			}
			err = outputISO(efi, base+"-efi.iso", kernel, initrd, cmdline)
			if err != nil {
				return fmt.Errorf("Error writing %s output: %v", o.Format, err)
			}
		case "gcp-img":
			kernel, initrd, cmdline, err := tarToInitrd(image)
			if err != nil {
				return fmt.Errorf("Error converting to initrd: %v", err)
			}
			err = outputImg(gcp, base+".img.tar.gz", kernel, initrd, cmdline)
			if err != nil {
				return fmt.Errorf("Error writing %s output: %v", o.Format, err)
			}
		case "qcow", "qcow2":
			kernel, initrd, cmdline, err := tarToInitrd(image)
			if err != nil {
				return fmt.Errorf("Error converting to initrd: %v", err)
			}
			err = outputImg(qcow, base+".qcow2", kernel, initrd, cmdline)
			if err != nil {
				return fmt.Errorf("Error writing %s output: %v", o.Format, err)
			}
		case "vhd":
			kernel, initrd, cmdline, err := tarToInitrd(image)
			if err != nil {
				return fmt.Errorf("Error converting to initrd: %v", err)
			}
			err = outputImg(vhd, base+".vhd", kernel, initrd, cmdline)
			if err != nil {
				return fmt.Errorf("Error writing %s output: %v", o.Format, err)
			}
		case "vmdk":
			kernel, initrd, cmdline, err := tarToInitrd(image)
			if err != nil {
				return fmt.Errorf("Error converting to initrd: %v", err)
			}
			err = outputImg(vmdk, base+".vmdk", kernel, initrd, cmdline)
			if err != nil {
				return fmt.Errorf("Error writing %s output: %v", o.Format, err)
			}
		case "":
			return fmt.Errorf("No format specified for output")
		default:
			return fmt.Errorf("Unknown output type %s", o.Format)
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

func tarInitrdKernel(kernel, initrd []byte) (*bytes.Buffer, error) {
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
	err = tw.Close()
	if err != nil {
		return buf, err
	}
	return buf, nil
}

func outputImg(image, filename string, kernel []byte, initrd []byte, args ...string) error {
	log.Debugf("output img: %s %s", image, filename)
	log.Infof("  %s", filename)
	buf, err := tarInitrdKernel(kernel, initrd)
	if err != nil {
		return err
	}
	img, err := dockerRunInput(buf, image, args...)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename, img, os.FileMode(0644))
	if err != nil {
		return err
	}
	return nil
}

func outputISO(image, filename string, kernel []byte, initrd []byte, args ...string) error {
	log.Debugf("output iso: %s %s", image, filename)
	log.Infof("  %s", filename)
	buf, err := tarInitrdKernel(kernel, initrd)
	if err != nil {
		return err
	}
	iso, err := dockerRunInput(buf, image, args...)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename, iso, os.FileMode(0644))
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
