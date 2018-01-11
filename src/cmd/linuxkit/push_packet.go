package main

import (
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	log "github.com/sirupsen/logrus"
)

var (
	packetDefaultArch       = "x86_64"
	packetDefaultDecompress = false
)

func init() {
	if runtime.GOARCH == "arm64" {
		packetDefaultArch = "aarch64"
		// decompress on arm64. iPXE/kernel does not
		// seem to grok compressed kernels/initrds.
		packetDefaultDecompress = true
	}
}

// Process the run arguments and execute run
func pushPacket(args []string) {
	flags := flag.NewFlagSet("packet", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s push packet [options] [name]\n\n", invoked)
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}
	baseURLFlag := flags.String("base-url", "", "Base URL that the kernel, initrd and iPXE script are served from (or "+packetBaseURL+")")
	nameFlag := flags.String("img-name", "", "Overrides the prefix used to identify the files. Defaults to [name] (or "+packetNameVar+")")
	archFlag := flags.String("arch", packetDefaultArch, "Image architecture (x86_64 or aarch64)")
	decompressFlag := flags.Bool("decompress", packetDefaultDecompress, "Decompress kernel/initrd before pushing")
	dstFlag := flags.String("destination", "", "URL where to push the image to. Currently only 'file' is supported as a scheme (which is also the default if omitted)")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	remArgs := flags.Args()
	prefix := "packet"
	if len(remArgs) > 0 {
		prefix = remArgs[0]
	}

	baseURL := getStringValue(packetBaseURL, *baseURLFlag, "")
	if baseURL == "" {
		log.Fatal("Need to specify a value for --base-url from where the kernel, initrd and iPXE script will be loaded from.")
	}

	if *dstFlag == "" {
		log.Fatal("Need to specify the destination where to push to.")
	}

	name := getStringValue(packetNameVar, *nameFlag, prefix)

	if _, err := os.Stat(fmt.Sprintf("%s-kernel", name)); os.IsNotExist(err) {
		log.Fatalf("kernel file does not exist: %v", err)
	}
	if _, err := os.Stat(fmt.Sprintf("%s-initrd.img", name)); os.IsNotExist(err) {
		log.Fatalf("initrd file does not exist: %v", err)
	}

	// Read kernel command line
	var cmdline string
	if c, err := ioutil.ReadFile(prefix + "-cmdline"); err != nil {
		log.Fatalf("Cannot open cmdline file: %v", err)
	} else {
		cmdline = string(c)
	}

	ipxeScript := packetIPXEScript(name, baseURL, cmdline, *archFlag)

	// Parse the destination
	dst, err := url.Parse(*dstFlag)
	if err != nil {
		log.Fatalf("Cannot parse destination: %v", err)
	}
	switch dst.Scheme {
	case "", "file":
		packetPushFile(dst, *decompressFlag, name, cmdline, ipxeScript)
	default:
		log.Fatalf("Unknown destination format: %s", dst.Scheme)
	}
}

func packetPushFile(dst *url.URL, decompress bool, name, cmdline, ipxeScript string) {
	// Make sure the destination exists
	dstPath := filepath.Clean(dst.Path)
	if err := os.MkdirAll(dstPath, 0755); err != nil {
		log.Fatalf("Could not create destination directory: %v", err)
	}

	kernelName := fmt.Sprintf("%s-kernel", name)
	if err := packetCopy(filepath.Join(dstPath, kernelName), kernelName, decompress); err != nil {
		log.Fatalf("Error copying kernel: %v", err)
	}

	initrdName := fmt.Sprintf("%s-initrd.img", name)
	if err := packetCopy(filepath.Join(dstPath, initrdName), initrdName, decompress); err != nil {
		log.Fatalf("Error copying initrd: %v", err)
	}

	ipxeScriptName := fmt.Sprintf("%s-packet.ipxe", name)
	if err := ioutil.WriteFile(filepath.Join(dstPath, ipxeScriptName), []byte(ipxeScript), 0644); err != nil {
		log.Fatalf("Error writing iPXE script: %v", err)
	}
}

func packetCopy(dst, src string, decompress bool) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	var r io.Reader = in
	if decompress {
		if rd, err := gzip.NewReader(in); err != nil {
			log.Warnf("%s does not seem to be gzip'ed (%v). Ignore decompress.", src, err)
		} else {
			r = rd
		}
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, r)
	if err != nil {
		return err
	}
	return out.Close()
}
