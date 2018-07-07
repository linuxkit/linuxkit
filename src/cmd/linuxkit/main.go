package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/version"
	"github.com/moby/tool/src/moby"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func init() {
	// Register LinuxKit images to build outputs with the vendored moby tool.
	// This allows us to overwrite the hashes locally without having
	// to re-vendor the 'github.com/moby/tool' when we update 'mkimage-*'
	imgs := map[string]string{
		"iso-bios":    "linuxkit/mkimage-iso-bios:fea1990f9703f24fc3c71ecd33695b60ca11043c",
		"iso-efi":     "linuxkit/mkimage-iso-efi:6948ef717ff8b0ca5831ff6b281404ae7f61d36c",
		"raw-bios":    "linuxkit/mkimage-raw-bios:89577baee3bf6e4b7c6669e9e7d16181a0aefbbf",
		"raw-efi":     "linuxkit/mkimage-raw-efi:276b61f502a7e4667e4a343df15e379f451e4cc3",
		"squashfs":    "linuxkit/mkimage-squashfs:3c5db8c90dfa828f9cfcf90f050c1db5ef177420",
		"gcp":         "linuxkit/mkimage-gcp:e6cdcf859ab06134c0c37a64ed5f886ec8dae1a1",
		"qcow2-efi":   "linuxkit/mkimage-qcow2-efi:fa3756a291319d0250a9eac1471296d8e4cae0f8",
		"vhd":         "linuxkit/mkimage-vhd:3820219e5c350fe8ab2ec6a217272ae82f4b9242",
		"dynamic-vhd": "linuxkit/mkimage-dynamic-vhd:743ac9959fe6d3912ebd78b4fd490b117c53f1a6",
		"vmdk":        "linuxkit/mkimage-vmdk:cee81a3ed9c44ae446ef7ebff8c42c1e77b3e1b5",
		"rpi3":        "linuxkit/mkimage-rpi3:c0645c61c71f7233a4a7cde2afecf41011668417",
	}
	if err := moby.UpdateOutputImages(imgs); err != nil {
		log.Fatalf("Failed to register mkimage-*. %v", err)
	}
}

// GlobalConfig is the global tool configuration
type GlobalConfig struct {
	Pkg PkgConfig `yaml:"pkg"`
}

// PkgConfig is the config specific to the `pkg` subcommand
type PkgConfig struct {
	// ContentTrustCommand is passed to `sh -c` and the stdout
	// (including whitespace and \n) is set as the content trust
	// passphrase. Can be used to execute a password manager.
	ContentTrustCommand string `yaml:"content-trust-passphrase-command"`
}

var (
	defaultLogFormatter = &log.TextFormatter{}

	// Config is the global tool configuration
	Config = GlobalConfig{}
)

// infoFormatter overrides the default format for Info() log events to
// provide an easier to read output
type infoFormatter struct {
}

func (f *infoFormatter) Format(entry *log.Entry) ([]byte, error) {
	if entry.Level == log.InfoLevel {
		return append([]byte(entry.Message), '\n'), nil
	}
	return defaultLogFormatter.Format(entry)
}

func printVersion() {
	fmt.Printf("%s version %s\n", filepath.Base(os.Args[0]), version.Version)
	if version.GitCommit != "" {
		fmt.Printf("commit: %s\n", version.GitCommit)
	}
	os.Exit(0)
}

func readConfig() {
	cfgPath := filepath.Join(os.Getenv("HOME"), ".moby", "linuxkit", "config.yml")
	cfgBytes, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		fmt.Printf("Failed to read %q\n", cfgPath)
		os.Exit(1)
	}
	if err := yaml.Unmarshal(cfgBytes, &Config); err != nil {
		fmt.Printf("Failed to parse %q\n", cfgPath)
		os.Exit(1)
	}
}

func main() {
	flag.Usage = func() {
		fmt.Printf("USAGE: %s [options] COMMAND\n\n", filepath.Base(os.Args[0]))
		fmt.Printf("Commands:\n")
		fmt.Printf("  build       Build an image from a YAML file\n")
		fmt.Printf("  metadata    Metadata utilities\n")
		fmt.Printf("  pkg         Package building\n")
		fmt.Printf("  push        Push a VM image to a cloud or image store\n")
		fmt.Printf("  run         Run a VM image on a local hypervisor or remote cloud\n")
		fmt.Printf("  serve       Run a local http server (for iPXE booting)\n")
		fmt.Printf("  version     Print version information\n")
		fmt.Printf("  help        Print this message\n")
		fmt.Printf("\n")
		fmt.Printf("Run '%s COMMAND --help' for more information on the command\n", filepath.Base(os.Args[0]))
		fmt.Printf("\n")
		fmt.Printf("Options:\n")
		flag.PrintDefaults()
	}
	flagQuiet := flag.Bool("q", false, "Quiet execution")
	flagVerbose := flag.Bool("v", false, "Verbose execution")

	readConfig()

	// Set up logging
	log.SetFormatter(new(infoFormatter))
	log.SetLevel(log.InfoLevel)
	flag.Parse()
	if *flagQuiet && *flagVerbose {
		fmt.Printf("Can't set quiet and verbose flag at the same time\n")
		os.Exit(1)
	}
	if *flagQuiet {
		log.SetLevel(log.ErrorLevel)
	}
	if *flagVerbose {
		// Switch back to the standard formatter
		log.SetFormatter(defaultLogFormatter)
		log.SetLevel(log.DebugLevel)
	}

	args := flag.Args()
	if len(args) < 1 {
		fmt.Printf("Please specify a command.\n\n")
		flag.Usage()
		os.Exit(1)
	}

	switch args[0] {
	case "build":
		build(args[1:])
	case "metadata":
		metadata(args[1:])
	case "pkg":
		pkg(args[1:])
	case "push":
		push(args[1:])
	case "run":
		run(args[1:])
	case "serve":
		serve(args[1:])
	case "version":
		printVersion()
	case "help":
		flag.Usage()
	default:
		fmt.Printf("%q is not valid command.\n\n", args[0])
		flag.Usage()
		os.Exit(1)
	}
}
