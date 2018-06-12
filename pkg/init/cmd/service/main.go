package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/containerd/containerd/namespaces"
	log "github.com/sirupsen/logrus"
)

const (
	defaultSocket              = "/run/containerd/containerd.sock"
	defaultPath                = "/containers/services"
	defaultContainerd          = "/usr/bin/containerd"
	installPath                = "/usr/bin/service"
	onbootPath                 = "/containers/onboot"
	shutdownPath               = "/containers/onshutdown"
	defaultContainerdNamespace = "services.linuxkit"
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
		fmt.Printf("  stop        Stop a service\n")
		fmt.Printf("  start       Start a service\n")
		fmt.Printf("  restart     Restart a service\n")
		fmt.Printf("  help        Print this message\n")
		fmt.Printf("\n")
		fmt.Printf("Run '%s COMMAND --help' for more information on the command\n", filepath.Base(os.Args[0]))
		fmt.Printf("\n")
		fmt.Printf("Options:\n")
		flag.PrintDefaults()
	}
	var (
		ctx                     = context.Background()
		flagQuiet               = flag.Bool("q", false, "Quiet execution")
		flagVerbose             = flag.Bool("v", false, "Verbose execution")
		flagContainerdNamespace = flag.String("containerd-namespace", defaultContainerdNamespace, "containerd namespace to use with services")
	)

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

	ctx = namespaces.WithNamespace(ctx, *flagContainerdNamespace)

	args := flag.Args()
	if len(args) < 1 {
		// check if called form startup scripts
		command := os.Args[0]
		switch {
		case strings.Contains(command, "onboot"):
			os.Exit(runcInit(onbootPath, "onboot"))
		case strings.Contains(command, "onshutdown"):
			os.Exit(runcInit(shutdownPath, "shutdown"))
		case strings.Contains(command, "containerd"):
			systemInitCmd(ctx, []string{})
			os.Exit(0)
		}
	}

	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	switch args[0] {
	case "stop":
		stopCmd(ctx, args[1:])
	case "start":
		startCmd(ctx, args[1:])
	case "restart":
		restartCmd(ctx, args[1:])
	case "system-init":
		systemInitCmd(ctx, args[1:])
	default:
		fmt.Printf("%q is not valid command.\n\n", args[0])
		flag.Usage()
		os.Exit(1)
	}
}
