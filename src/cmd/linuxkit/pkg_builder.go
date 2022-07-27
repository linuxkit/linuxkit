package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/pkglib"
	log "github.com/sirupsen/logrus"
)

func pkgBuilderUsage() {
	invoked := filepath.Base(os.Args[0])
	fmt.Printf("USAGE: %s builder command [options]\n\n", invoked)
	fmt.Printf("Supported commands are\n")
	// Please keep these in alphabetical order
	fmt.Printf("  du\n")
	fmt.Printf("  prune\n")
	fmt.Printf("\n")
	fmt.Printf("'options' are the backend specific options.\n")
	fmt.Printf("See '%s builder [command] --help' for details.\n\n", invoked)
}

// Process the builder
func pkgBuilder(args []string) {
	if len(args) < 1 {
		pkgBuilderUsage()
		os.Exit(1)
	}
	switch args[0] {
	// Please keep cases in alphabetical order
	case "du":
		pkgBuilderCommands(args[0], args[1:])
	case "prune":
		pkgBuilderCommands(args[0], args[1:])
	case "help", "-h", "-help", "--help":
		pkgBuilderUsage()
		os.Exit(0)
	default:
		log.Errorf("No 'builder' command specified.")
	}
}

func pkgBuilderCommands(command string, args []string) {
	flags := flag.NewFlagSet(command, flag.ExitOnError)
	builders := flags.String("builders", "", "Which builders to use for which platforms, e.g. linux/arm64=docker-context-arm64, overrides defaults and environment variables, see https://github.com/linuxkit/linuxkit/blob/master/docs/packages.md#Providing-native-builder-nodes")
	platforms := flags.String("platforms", fmt.Sprintf("linux/%s", runtime.GOARCH), "Which platforms we built images for")
	builderImage := flags.String("builder-image", defaultBuilderImage, "buildkit builder container image to use")
	verbose := flags.Bool("v", false, "Verbose output")
	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}
	// build the builders map
	buildersMap := make(map[string]string)
	// look for builders env var
	buildersMap, err := buildPlatformBuildersMap(os.Getenv(buildersEnvVar), buildersMap)
	if err != nil {
		log.Fatalf("%s in environment variable %s\n", err.Error(), buildersEnvVar)
	}
	// any CLI options override env var
	buildersMap, err = buildPlatformBuildersMap(*builders, buildersMap)
	if err != nil {
		log.Fatalf("%s in --builders flag\n", err.Error())
	}

	platformsToClean := strings.Split(*platforms, ",")
	switch command {
	case "du":
		if err := pkglib.DiskUsage(buildersMap, *builderImage, platformsToClean, *verbose); err != nil {
			log.Fatalf("Unable to print disk usage of builder: %v", err)
		}
	case "prune":
		if err := pkglib.PruneBuilder(buildersMap, *builderImage, platformsToClean, *verbose); err != nil {
			log.Fatalf("Unable to prune builder: %v", err)
		}
	default:
		log.Errorf("unexpected command %s", command)
		pkgBuilderUsage()
		os.Exit(1)
	}
}
