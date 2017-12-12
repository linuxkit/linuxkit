package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"github.com/moby/tool/src/moby"
	log "github.com/sirupsen/logrus"
)

var (
	defaultLogFormatter = &log.TextFormatter{}

	// Version is the human-readable version
	Version = "unknown"

	// GitCommit hash, set at compile time
	GitCommit = "unknown"
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

func version() {
	fmt.Printf("%s version %s\n", filepath.Base(os.Args[0]), Version)
	fmt.Printf("commit: %s\n", GitCommit)
	os.Exit(0)
}

const mobyDefaultDir string = ".moby"

func defaultMobyConfigDir() string {
	home := homeDir()
	return filepath.Join(home, mobyDefaultDir)
}

func main() {
	flag.Usage = func() {
		fmt.Printf("USAGE: %s [options] COMMAND\n\n", filepath.Base(os.Args[0]))
		fmt.Printf("Commands:\n")
		fmt.Printf("  build       Build a Moby image from a YAML file\n")
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
	flagCPUProfile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	flagMemProfile := flag.String("memprofile", "", "write mem profile to `file`")

	// config and cache directory
	flagConfigDir := flag.String("config", defaultMobyConfigDir(), "Configuration directory")

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

	mobyDir := *flagConfigDir
	err := os.MkdirAll(mobyDir, 0755)
	if err != nil {
		log.Fatalf("Could not create config directory [%s]: %v", mobyDir, err)
	}

	err = os.MkdirAll(filepath.Join(mobyDir, "tmp"), 0755)
	if err != nil {
		log.Fatalf("Could not create config tmp directory [%s]: %v", filepath.Join(mobyDir, "tmp"), err)
	}
	moby.MobyDir = mobyDir

	if *flagCPUProfile != "" {
		f, err := os.Create(*flagCPUProfile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}

	switch args[0] {
	case "build":
		build(args[1:])
	case "version":
		version()
	case "help":
		flag.Usage()
	default:
		fmt.Printf("%q is not valid command.\n\n", args[0])
		flag.Usage()
		os.Exit(1)
	}

	if *flagMemProfile != "" {
		f, err := os.Create(*flagMemProfile)
		if err != nil {
			log.Fatal("could not create memory profile: ", err)
		}
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			log.Fatal("could not write memory profile: ", err)
		}
		f.Close()
	}
}
