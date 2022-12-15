package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	// drop-in 100% compatible replacement and 17% faster than compress/gzip.
	gzip "github.com/klauspost/pgzip"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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

func pushPacketCmd() *cobra.Command {
	var (
		baseURLFlag string
		nameFlag    string
		arch        string
		dst         string
		decompress  bool
	)
	cmd := &cobra.Command{
		Use:   "packet",
		Short: "push image to Equinix Metal / Packet",
		Long: `Push image to Equinix Metal / Packet.
		Single argument is the prefix to use for the image, defualts to "packet".
		`,
		Example: "linuxkit push packet [options] [name]",
		RunE: func(cmd *cobra.Command, args []string) error {
			prefix := "packet"
			if len(args) > 0 {
				prefix = args[0]
			}

			baseURL := getStringValue(packetBaseURL, baseURLFlag, "")
			if baseURL == "" {
				return fmt.Errorf("Need to specify a value for --base-url from where the kernel, initrd and iPXE script will be loaded from.")
			}

			if dst == "" {
				return fmt.Errorf("Need to specify the destination where to push to.")
			}

			name := getStringValue(packetNameVar, nameFlag, prefix)

			if _, err := os.Stat(fmt.Sprintf("%s-kernel", name)); os.IsNotExist(err) {
				return fmt.Errorf("kernel file does not exist: %v", err)
			}
			if _, err := os.Stat(fmt.Sprintf("%s-initrd.img", name)); os.IsNotExist(err) {
				return fmt.Errorf("initrd file does not exist: %v", err)
			}

			// Read kernel command line
			var cmdline string
			if c, err := os.ReadFile(prefix + "-cmdline"); err != nil {
				return fmt.Errorf("Cannot open cmdline file: %v", err)
			} else {
				cmdline = string(c)
			}

			ipxeScript := packetIPXEScript(name, baseURL, cmdline, arch)

			// Parse the destination
			dst, err := url.Parse(dst)
			if err != nil {
				return fmt.Errorf("Cannot parse destination: %v", err)
			}
			switch dst.Scheme {
			case "", "file":
				packetPushFile(dst, decompress, name, cmdline, ipxeScript)
			default:
				return fmt.Errorf("Unknown destination format: %s", dst.Scheme)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&baseURLFlag, "base-url", "", "Base URL that the kernel, initrd and iPXE script are served from (or "+packetBaseURL+")")
	cmd.Flags().StringVar(&nameFlag, "img-name", "", "Overrides the prefix used to identify the files. Defaults to [name] (or "+packetNameVar+")")
	cmd.Flags().StringVar(&arch, "arch", packetDefaultArch, "Image architecture (x86_64 or aarch64)")
	cmd.Flags().BoolVar(&decompress, "decompress", packetDefaultDecompress, "Decompress kernel/initrd before pushing")
	cmd.Flags().StringVar(&dst, "destination", "", "URL where to push the image to. Currently only 'file' is supported as a scheme (which is also the default if omitted)")

	return cmd
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
	if err := os.WriteFile(filepath.Join(dstPath, ipxeScriptName), []byte(ipxeScript), 0644); err != nil {
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
