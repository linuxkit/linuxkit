package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/pkglib"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

func pkgPush(args []string) {
	flags := flag.NewFlagSet("pkg push", flag.ExitOnError)
	flags.Usage = func() {
		invoked := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "USAGE: %s pkg push [options] path\n\n", invoked)
		fmt.Fprintf(os.Stderr, "'path' specifies the path to the package source directory.\n")
		fmt.Fprintf(os.Stderr, "\n")
		flags.PrintDefaults()
	}

	force := flags.Bool("force", false, "Force rebuild")
	release := flags.String("release", "", "Release the given version")
	nobuild := flags.Bool("nobuild", false, "Skip the build")
	docker := flags.Bool("docker", false, "Store the built image in the docker image cache instead of the default linuxkit cache")
	platforms := flags.String("platforms", "", "Which platforms to build for, defaults to all of those for which the package can be built")
	builders := flags.String("builders", "", "Which builders to use for which platforms, e.g. linux/arm64=docker-context-arm64, overrides defaults and environment variables, see https://github.com/linuxkit/linuxkit/blob/master/docs/packages.md#Providing-native-builder-nodes")
	manifest := flags.Bool("manifest", true, "Create and push multi-arch manifest")
	image := flags.Bool("image", true, "Build and push image for the current platform")
	buildCacheDir := flags.String("cache", defaultLinuxkitCache(), "Directory for storing built image, incompatible with --docker")

	p, err := pkglib.NewFromCLI(flags, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	var opts []pkglib.BuildOpt
	opts = append(opts, pkglib.WithBuildPush())
	if *force {
		opts = append(opts, pkglib.WithBuildForce())
	}
	if *nobuild {
		opts = append(opts, pkglib.WithBuildSkip())
	}
	if *release != "" {
		opts = append(opts, pkglib.WithRelease(*release))
	}
	if *manifest {
		opts = append(opts, pkglib.WithBuildManifest())
	}
	if *image {
		opts = append(opts, pkglib.WithBuildImage())
	}
	opts = append(opts, pkglib.WithBuildCacheDir(*buildCacheDir))
	if *docker {
		opts = append(opts, pkglib.WithBuildTargetDockerCache())
	}
	// if platforms requested is blank, use all from the config
	var plats []imagespec.Platform
	if *platforms == "" {
		for _, a := range p.Arches() {
			plats = append(plats, imagespec.Platform{OS: "linux", Architecture: a})
		}
	} else {
		for _, p := range strings.Split(*platforms, ",") {
			parts := strings.SplitN(p, "/", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				fmt.Fprintf(os.Stderr, "invalid target platform specification '%s'\n", p)
				os.Exit(1)
			}
			plats = append(plats, imagespec.Platform{OS: parts[0], Architecture: parts[1]})
		}
	}
	opts = append(opts, pkglib.WithBuildPlatforms(plats...))
	// build the builders map
	buildersMap := map[string]string{}
	// look for builders env var
	buildersMap, err = buildPlatformBuildersMap(os.Getenv(buildersEnvVar), buildersMap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s in environment variable %s\n", err.Error(), buildersEnvVar)
		os.Exit(1)
	}
	// any CLI options override env var
	buildersMap, err = buildPlatformBuildersMap(*builders, buildersMap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s in --builders flag\n", err.Error())
		os.Exit(1)
	}
	opts = append(opts, pkglib.WithBuildBuilders(buildersMap))

	if *nobuild {
		fmt.Printf("Pushing %q without building\n", p.Tag())
	} else {
		fmt.Printf("Building and pushing %q\n", p.Tag())
	}

	if err := p.Build(opts...); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
