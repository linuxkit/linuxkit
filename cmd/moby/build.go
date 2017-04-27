package main

import (
	"archive/tar"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/linuxkit/linuxkit/src/initrd"
)

const defaultNameForStdin = "moby"

// Process the build arguments and execute build
func build(args []string) {
	buildCmd := flag.NewFlagSet("build", flag.ExitOnError)
	buildCmd.Usage = func() {
		fmt.Printf("USAGE: %s build [options] <file>[.yml] | -\n\n", os.Args[0])
		fmt.Printf("Options:\n")
		buildCmd.PrintDefaults()
	}
	buildName := buildCmd.String("name", "", "Name to use for output files")
	buildPull := buildCmd.Bool("pull", false, "Always pull images")

	if err := buildCmd.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}
	remArgs := buildCmd.Args()

	if len(remArgs) == 0 {
		fmt.Println("Please specify a configuration file")
		buildCmd.Usage()
		os.Exit(1)
	}
	name := *buildName
	var config []byte
	if conf := remArgs[0]; conf == "-" {
		var err error
		config, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("Cannot read stdin: %v", err)
		}
		if name == "" {
			name = defaultNameForStdin
		}
	} else {
		if !(filepath.Ext(conf) == ".yml" || filepath.Ext(conf) == ".yaml") {
			conf = conf + ".yml"
		}
		var err error
		config, err = ioutil.ReadFile(conf)
		if err != nil {
			log.Fatalf("Cannot open config file: %v", err)
		}
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(conf), filepath.Ext(conf))
		}
	}

	buildInternal(name, *buildPull, config)
}

func initrdAppend(iw *initrd.Writer, r io.Reader) {
	_, err := initrd.Copy(iw, r)
	if err != nil {
		log.Fatalf("initrd write error: %v", err)
	}
}

func enforceContentTrust(fullImageName string, config *TrustConfig) bool {
	for _, img := range config.Image {
		// First check for an exact name match
		if img == fullImageName {
			return true
		}
		// Also check for an image name only match
		// by removing a possible tag (with possibly added digest):
		if img == strings.TrimSuffix(fullImageName, ":") {
			return true
		}
		// and by removing a possible digest:
		if img == strings.TrimSuffix(fullImageName, "@sha256:") {
			return true
		}
	}

	for _, org := range config.Org {
		if strings.HasPrefix(fullImageName, org+"/") {
			return true
		}
	}
	return false
}

// Perform the actual build process
func buildInternal(name string, pull bool, config []byte) {
	m, err := NewConfig(config)
	if err != nil {
		log.Fatalf("Invalid config: %v", err)
	}

	w := new(bytes.Buffer)
	iw := initrd.NewWriter(w)

	if pull || enforceContentTrust(m.Kernel.Image, &m.Trust) {
		log.Infof("Pull kernel image: %s", m.Kernel.Image)
		err := dockerPull(m.Kernel.Image, enforceContentTrust(m.Kernel.Image, &m.Trust))
		if err != nil {
			log.Fatalf("Could not pull image %s: %v", m.Kernel.Image, err)
		}
	}
	// get kernel and initrd tarball from container
	// TODO examine contents to see what names they might have
	log.Infof("Extract kernel image: %s", m.Kernel.Image)
	const (
		kernelName = "kernel"
		ktarName   = "kernel.tar"
	)
	out, err := dockerRun(m.Kernel.Image, "tar", "cf", "-", kernelName, ktarName)
	if err != nil {
		log.Fatalf("Failed to extract kernel image and tarball: %v", err)
	}
	buf := bytes.NewBuffer(out)
	kernel, ktar, err := untarKernel(buf, kernelName, ktarName)
	if err != nil {
		log.Fatalf("Could not extract kernel image and filesystem from tarball. %v", err)
	}
	initrdAppend(iw, ktar)

	// convert init images to tarballs
	log.Infof("Add init containers:")
	for _, ii := range m.Init {
		if pull || enforceContentTrust(ii, &m.Trust) {
			log.Infof("Pull init image: %s", ii)
			err := dockerPull(ii, enforceContentTrust(ii, &m.Trust))
			if err != nil {
				log.Fatalf("Could not pull image %s: %v", ii, err)
			}
		}
		log.Infof("Process init image: %s", ii)
		init, err := ImageExtract(ii, "")
		if err != nil {
			log.Fatalf("Failed to build init tarball from %s: %v", ii, err)
		}
		buffer := bytes.NewBuffer(init)
		initrdAppend(iw, buffer)
	}

	log.Infof("Add onboot containers:")
	for i, image := range m.Onboot {
		if pull || enforceContentTrust(image.Image, &m.Trust) {
			log.Infof("  Pull: %s", image.Image)
			err := dockerPull(image.Image, enforceContentTrust(image.Image, &m.Trust))
			if err != nil {
				log.Fatalf("Could not pull image %s: %v", image.Image, err)
			}
		}
		log.Infof("  Create OCI config for %s", image.Image)
		config, err := ConfigToOCI(&image)
		if err != nil {
			log.Fatalf("Failed to create config.json for %s: %v", image.Image, err)
		}
		so := fmt.Sprintf("%03d", i)
		path := "containers/onboot/" + so + "-" + image.Name
		out, err := ImageBundle(path, image.Image, config)
		if err != nil {
			log.Fatalf("Failed to extract root filesystem for %s: %v", image.Image, err)
		}
		buffer := bytes.NewBuffer(out)
		initrdAppend(iw, buffer)
	}

	log.Infof("Add service containers:")
	for _, image := range m.Services {
		if pull || enforceContentTrust(image.Image, &m.Trust) {
			log.Infof("  Pull: %s", image.Image)
			err := dockerPull(image.Image, enforceContentTrust(image.Image, &m.Trust))
			if err != nil {
				log.Fatalf("Could not pull image %s: %v", image.Image, err)
			}
		}
		log.Infof("  Create OCI config for %s", image.Image)
		config, err := ConfigToOCI(&image)
		if err != nil {
			log.Fatalf("Failed to create config.json for %s: %v", image.Image, err)
		}
		path := "containers/services/" + image.Name
		out, err := ImageBundle(path, image.Image, config)
		if err != nil {
			log.Fatalf("Failed to extract root filesystem for %s: %v", image.Image, err)
		}
		buffer := bytes.NewBuffer(out)
		initrdAppend(iw, buffer)
	}

	// add files
	buffer, err := filesystem(m)
	if err != nil {
		log.Fatalf("failed to add filesystem parts: %v", err)
	}
	initrdAppend(iw, buffer)
	err = iw.Close()
	if err != nil {
		log.Fatalf("initrd close error: %v", err)
	}

	log.Infof("Create outputs:")
	err = outputs(m, name, kernel.Bytes(), w.Bytes())
	if err != nil {
		log.Fatalf("Error writing outputs: %v", err)
	}
}

func untarKernel(buf *bytes.Buffer, kernelName, ktarName string) (*bytes.Buffer, *bytes.Buffer, error) {
	tr := tar.NewReader(buf)

	var kernel, ktar *bytes.Buffer

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalln(err)
		}
		switch hdr.Name {
		case kernelName:
			kernel = new(bytes.Buffer)
			_, err := io.Copy(kernel, tr)
			if err != nil {
				return nil, nil, err
			}
		case ktarName:
			ktar = new(bytes.Buffer)
			_, err := io.Copy(ktar, tr)
			if err != nil {
				return nil, nil, err
			}
		default:
			continue
		}
	}

	if kernel == nil {
		return nil, nil, errors.New("did not find kernel in kernel image")
	}
	if ktar == nil {
		return nil, nil, errors.New("did not find kernel.tar in kernel image")
	}

	return kernel, ktar, nil
}
