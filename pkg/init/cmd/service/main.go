package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	defaultSocket     = "/run/containerd/containerd.sock"
	defaultPath       = "/containers/services"
	defaultContainerd = "/usr/bin/containerd"
	installPath       = "/usr/bin/service"
	onbootPath        = "/containers/onboot"
	shutdownPath      = "/containers/onshutdown"
)

var (
	defaultLogFormatter = &log.TextFormatter{}
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

func main() {
	flag.Usage = func() {
		fmt.Printf("USAGE: %s [options] COMMAND\n\n", filepath.Base(os.Args[0]))
		fmt.Printf("Commands:\n")
		fmt.Printf("  system-init Prepare the system at start of day\n")
		fmt.Printf("  start       Start a service\n")
		fmt.Printf("  help        Print this message\n")
		fmt.Printf("\n")
		fmt.Printf("Run '%s COMMAND --help' for more information on the command\n", filepath.Base(os.Args[0]))
		fmt.Printf("\n")
		fmt.Printf("Options:\n")
		flag.PrintDefaults()
	}
	flagQuiet := flag.Bool("q", false, "Quiet execution")
	flagVerbose := flag.Bool("v", false, "Verbose execution")

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
		// check if called form startup scripts
		command := os.Args[0]
		switch {
		case strings.Contains(command, "onboot"):
			os.Exit(runcInit(onbootPath))
		case strings.Contains(command, "onshutdown"):
			os.Exit(runcInit(shutdownPath))
		case strings.Contains(command, "containerd"):
			systemInitCmd([]string{})
			os.Exit(0)
		}
	}

	switch args[0] {
	case "start":
		startCmd(args[1:])
	case "system-init":
		systemInitCmd(args[1:])
	default:
		fmt.Printf("%q is not valid command.\n\n", args[0])
		flag.Usage()
		os.Exit(1)
	}
}
