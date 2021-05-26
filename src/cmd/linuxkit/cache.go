package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	log "github.com/sirupsen/logrus"
)

func cacheUsage() {
	invoked := filepath.Base(os.Args[0])
	fmt.Printf("USAGE: %s cache command [options]\n\n", invoked)
	fmt.Printf("Supported commands are\n")
	// Please keep these in alphabetical order
	fmt.Printf("  clean\n")
	fmt.Printf("  export\n")
	fmt.Printf("  ls\n")
	fmt.Printf("\n")
	fmt.Printf("'options' are the backend specific options.\n")
	fmt.Printf("See '%s cache [command] --help' for details.\n\n", invoked)
}

// Process the cache
func cache(args []string) {
	if len(args) < 1 {
		cacheUsage()
		os.Exit(1)
	}
	switch args[0] {
	// Please keep cases in alphabetical order
	case "clean":
		cacheClean(args[1:])
	case "ls":
		cacheList(args[1:])
	case "export":
		cacheExport(args[1:])
	case "help", "-h", "-help", "--help":
		cacheUsage()
		os.Exit(0)
	default:
		log.Errorf("No 'cache' command specified.")
	}
}

func defaultLinuxkitCache() string {
	lktDir := ".linuxkit"
	home := util.HomeDir()
	return filepath.Join(home, lktDir, "cache")
}
