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

const (
	buildersEnvVar = "LINUXKIT_BUILDERS"
)

func pkgBuild(args []string) {
	flags := flag.NewFlagSet("pkg build", flag.ExitOnError)
	flags.Usage = func() {
		invoked := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "USAGE: %s pkg build [options] path\n\n", invoked)
		fmt.Fprintf(os.Stderr, "'path' specifies the path to the package source directory.\n")
		fmt.Fprintf(os.Stderr, "\n")
		flags.PrintDefaults()
	}

	force := flags.Bool("force", false, "Force rebuild")
	docker := flags.Bool("docker", false, "Store the built image in the docker image cache instead of the default linuxkit cache")
	platforms := flags.String("platforms", "", "Which platforms to build for, defaults to all of those for which the package can be built")
	skipPlatforms := flags.String("skip-platforms", "", "Platforms that should be skipped, even if present in build.yml")
	builders := flags.String("builders", "", "Which builders to use for which platforms, e.g. linux/arm64=docker-context-arm64, overrides defaults and environment variables, see https://github.com/linuxkit/linuxkit/blob/master/docs/packages.md#Providing-native-builder-nodes")
	buildCacheDir := flags.String("cache", defaultLinuxkitCache(), "Directory for storing built image, incompatible with --docker")

	p, err := pkglib.NewFromCLI(flags, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Building %q\n", p.Tag())

	opts := []pkglib.BuildOpt{pkglib.WithBuildImage()}
	if *force {
		opts = append(opts, pkglib.WithBuildForce())
	}
	opts = append(opts, pkglib.WithBuildCacheDir(*buildCacheDir))
	if *docker {
		opts = append(opts, pkglib.WithBuildTargetDockerCache())
	}

	// skipPlatformsMap contains platforms that should be skipped
	skipPlatformsMap := make(map[string]bool)
	if *skipPlatforms != "" {
		for _, platform := range strings.Split(*skipPlatforms, ",") {
			parts := strings.SplitN(platform, "/", 2)
			if len(parts) != 2 || parts[0] == "" || parts[0] != "linux" || parts[1] == "" {
				fmt.Fprintf(os.Stderr, "invalid target platform specification '%s'\n", platform)
				os.Exit(1)
			}
			skipPlatformsMap[strings.Trim(parts[1], " ")] = true
		}
	}
	// if platforms requested is blank, use all from the config
	var plats []imagespec.Platform
	if *platforms == "" {
		for _, a := range p.Arches() {
			if _, ok := skipPlatformsMap[a]; ok {
				continue
			}
			plats = append(plats, imagespec.Platform{OS: "linux", Architecture: a})
		}
	} else {
		// don't allow the use of --skip-platforms with --platforms
		if *skipPlatforms != "" {
			fmt.Fprintln(os.Stderr, "--skip-platforms and --platforms may not be used together")
			os.Exit(1)
		}
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
	if err := p.Build(opts...); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func buildPlatformBuildersMap(inputs string, existing map[string]string) (map[string]string, error) {
	if inputs == "" {
		return existing, nil
	}
	for _, platformBuilder := range strings.Split(inputs, ",") {
		parts := strings.SplitN(platformBuilder, "=", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return existing, fmt.Errorf("invalid platform=builder specification '%s'", platformBuilder)
		}
		platform, builder := parts[0], parts[1]
		parts = strings.SplitN(platform, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return existing, fmt.Errorf("invalid platform specification '%s'", platform)
		}
		existing[platform] = builder
	}
	return existing, nil
}
