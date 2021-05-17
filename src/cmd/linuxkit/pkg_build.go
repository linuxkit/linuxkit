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
	pkgBuildPush(args, false)
}

func pkgBuildPush(args []string, withPush bool) {
	flags := flag.NewFlagSet("pkg build", flag.ExitOnError)
	flags.Usage = func() {
		invoked := filepath.Base(os.Args[0])
		name := "build"
		if withPush {
			name = "push"
		}
		fmt.Fprintf(os.Stderr, "USAGE: %s pkg %s [options] path\n\n", name, invoked)
		fmt.Fprintf(os.Stderr, "'path' specifies the path to the package source directory.\n")
		fmt.Fprintf(os.Stderr, "\n")
		flags.PrintDefaults()
	}

	force := flags.Bool("force", false, "Force rebuild even if image is in local cache")
	docker := flags.Bool("docker", false, "Store the built image in the docker image cache instead of the default linuxkit cache")
	platforms := flags.String("platforms", "", "Which platforms to build for, defaults to all of those for which the package can be built")
	skipPlatforms := flags.String("skip-platforms", "", "Platforms that should be skipped, even if present in build.yml")
	builders := flags.String("builders", "", "Which builders to use for which platforms, e.g. linux/arm64=docker-context-arm64, overrides defaults and environment variables, see https://github.com/linuxkit/linuxkit/blob/master/docs/packages.md#Providing-native-builder-nodes")
	buildCacheDir := flags.String("cache", defaultLinuxkitCache(), "Directory for storing built image, incompatible with --docker")

	// some logic clarification:
	// pkg build                   - always builds unless is in cache
	// pkg build --force           - always builds even if is in cache
	// pkg push                    - always builds unless is in cache
	// pkg push --force            - always builds even if is in cache
	// pkg push --nobuild          - skips build; if not in cache, fails
	// pkg push --nobuild --force  - nonsensical

	var (
		release           *string
		nobuild, manifest *bool
		nobuildRef        = false
	)
	nobuild = &nobuildRef
	if withPush {
		release = flags.String("release", "", "Release the given version")
		nobuild = flags.Bool("nobuild", false, "Skip building the image before pushing, conflicts with -force")
		manifest = flags.Bool("manifest", true, "Create and push multi-arch manifest")
	}

	pkgs, err := pkglib.NewFromCLI(flags, args...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if *nobuild && *force {
		fmt.Fprint(os.Stderr, "flags -force and -nobuild conflict")
		os.Exit(1)
	}

	var opts []pkglib.BuildOpt
	if *force {
		opts = append(opts, pkglib.WithBuildForce())
	}
	opts = append(opts, pkglib.WithBuildCacheDir(*buildCacheDir))

	if withPush {
		opts = append(opts, pkglib.WithBuildPush())
		if *nobuild {
			opts = append(opts, pkglib.WithBuildSkip())
		}
		if *release != "" {
			opts = append(opts, pkglib.WithRelease(*release))
		}
		if *manifest {
			opts = append(opts, pkglib.WithBuildManifest())
		}
	}
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
	// if requested specific platforms, build those. If not, then we will
	// retrieve the defaults in the loop over each package.
	var plats []imagespec.Platform
	// don't allow the use of --skip-platforms with --platforms
	if *platforms != "" && *skipPlatforms != "" {
		fmt.Fprintln(os.Stderr, "--skip-platforms and --platforms may not be used together")
		os.Exit(1)
	}
	// process the platforms if provided
	if *platforms != "" {
		for _, p := range strings.Split(*platforms, ",") {
			parts := strings.SplitN(p, "/", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				fmt.Fprintf(os.Stderr, "invalid target platform specification '%s'\n", p)
				os.Exit(1)
			}
			plats = append(plats, imagespec.Platform{OS: parts[0], Architecture: parts[1]})
		}
	}

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

	for _, p := range pkgs {
		// things we need our own copies of
		var (
			pkgOpts  = make([]pkglib.BuildOpt, len(opts))
			pkgPlats = make([]imagespec.Platform, len(plats))
		)
		copy(pkgOpts, opts)
		copy(pkgPlats, plats)
		// unless overridden, platforms are specific to a package, so this needs to be inside the for loop
		if len(pkgPlats) == 0 {
			for _, a := range p.Arches() {
				if _, ok := skipPlatformsMap[a]; ok {
					continue
				}
				pkgPlats = append(pkgPlats, imagespec.Platform{OS: "linux", Architecture: a})
			}
		}
		pkgOpts = append(pkgOpts, pkglib.WithBuildPlatforms(pkgPlats...))

		var msg, action string
		switch {
		case !withPush:
			msg = fmt.Sprintf("Building %q", p.Tag())
			action = "building"
		case *nobuild:
			msg = fmt.Sprintf("Pushing %q without building", p.Tag())
			action = "building and pushing"
		default:
			msg = fmt.Sprintf("Building and pushing %q", p.Tag())
			action = "building and pushing"
		}

		fmt.Println(msg)

		if err := p.Build(pkgOpts...); err != nil {
			fmt.Fprintf(os.Stderr, "Error %s %q: %v\n", action, p.Tag(), err)
			os.Exit(1)
		}
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
