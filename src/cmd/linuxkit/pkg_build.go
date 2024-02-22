package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/pkglib"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
)

const (
	buildersEnvVar      = "LINUXKIT_BUILDERS"
	envVarCacheDir      = "LINUXKIT_CACHE"
	defaultBuilderImage = "moby/buildkit:v0.12.3"
)

// some logic clarification:
// pkg build                   - builds unless is in cache or published in registry
// pkg build --pull            - builds unless is in cache or published in registry; pulls from registry if not in cache
// pkg build --force           - always builds even if is in cache or published in registry
// pkg build --force --pull    - always builds even if is in cache or published in registry; --pull ignored
// pkg push                    - always builds unless is in cache
// pkg push --force            - always builds even if is in cache
// pkg push --nobuild          - skips build; if not in cache, fails
// pkg push --nobuild --force  - nonsensical

// addCmdRunPkgBuildPush adds the RunE function and flags to a cobra.Command
// for "pkg build" or "pkg push".
func addCmdRunPkgBuildPush(cmd *cobra.Command, withPush bool) *cobra.Command {
	var (
		force          bool
		pull           bool
		ignoreCache    bool
		docker         bool
		platforms      string
		skipPlatforms  string
		builders       string
		builderImage   string
		builderRestart bool
		release        string
		nobuild        bool
		manifest       bool
		cacheDir       = flagOverEnvVarOverDefaultString{def: defaultLinuxkitCache(), envVar: envVarCacheDir}
		sbomScanner    string
		dockerfile     string
		buildArgFiles  []string
	)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		pkgs, err := pkglib.NewFromConfig(pkglibConfig, args...)
		if err != nil {
			return err
		}

		if nobuild && force {
			return errors.New("flags -force and -nobuild conflict")
		}
		if pull && force {
			return errors.New("flags -force and -pull conflict")
		}

		var opts []pkglib.BuildOpt
		if force {
			opts = append(opts, pkglib.WithBuildForce())
		}
		if ignoreCache {
			opts = append(opts, pkglib.WithBuildIgnoreCache())
		}
		if pull {
			opts = append(opts, pkglib.WithBuildPull())
		}

		opts = append(opts, pkglib.WithBuildCacheDir(cacheDir.String()))

		if withPush {
			opts = append(opts, pkglib.WithBuildPush())
			if nobuild {
				opts = append(opts, pkglib.WithBuildSkip())
			}
			if release != "" {
				opts = append(opts, pkglib.WithRelease(release))
			}
			if manifest {
				opts = append(opts, pkglib.WithBuildManifest())
			}
		}
		if docker {
			opts = append(opts, pkglib.WithBuildTargetDockerCache())
		}

		if sbomScanner != "false" {
			opts = append(opts, pkglib.WithBuildSbomScanner(sbomScanner))
		}
		opts = append(opts, pkglib.WithDockerfile(dockerfile))

		// read any build arg files
		var buildArgs []string
		for _, filename := range buildArgFiles {
			f, err := os.Open(filename)
			if err != nil {
				return fmt.Errorf("error opening build args file %s: %w", filename, err)
			}
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				buildArgs = append(buildArgs, scanner.Text())
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("error reading build args file %s: %w", filename, err)
			}
		}
		opts = append(opts, pkglib.WithBuildArgs(buildArgs))

		// skipPlatformsMap contains platforms that should be skipped
		skipPlatformsMap := make(map[string]bool)
		if skipPlatforms != "" {
			for _, platform := range strings.Split(skipPlatforms, ",") {
				parts := strings.SplitN(platform, "/", 2)
				if len(parts) != 2 || parts[0] == "" || parts[0] != "linux" || parts[1] == "" {
					return fmt.Errorf("invalid target platform specification '%s'\n", platform)
				}
				skipPlatformsMap[strings.Trim(parts[1], " ")] = true
			}
		}
		// if requested specific platforms, build those. If not, then we will
		// retrieve the defaults in the loop over each package.
		var plats []imagespec.Platform
		// don't allow the use of --skip-platforms with --platforms
		if platforms != "" && skipPlatforms != "" {
			return errors.New("--skip-platforms and --platforms may not be used together")
		}
		// process the platforms if provided
		if platforms != "" {
			for _, p := range strings.Split(platforms, ",") {
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
			return fmt.Errorf("error in environment variable %s: %w", buildersEnvVar, err)
		}
		// any CLI options override env var
		buildersMap, err = buildPlatformBuildersMap(builders, buildersMap)
		if err != nil {
			return fmt.Errorf("error in --builders flag: %w", err)
		}
		opts = append(opts, pkglib.WithBuildBuilders(buildersMap))
		opts = append(opts, pkglib.WithBuildBuilderImage(builderImage))
		opts = append(opts, pkglib.WithBuildBuilderRestart(builderRestart))

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

			// if there are no platforms to build for, do nothing.
			// note that this is *not* an error; we simply skip it
			if len(pkgPlats) == 0 {
				fmt.Printf("Skipping %s with no architectures to build\n", p.Tag())
				continue
			}

			pkgOpts = append(pkgOpts, pkglib.WithBuildPlatforms(pkgPlats...))

			var msg, action string
			switch {
			case !withPush:
				msg = fmt.Sprintf("Building %q", p.Tag())
				action = "building"
			case nobuild:
				msg = fmt.Sprintf("Pushing %q without building", p.Tag())
				action = "building and pushing"
			default:
				msg = fmt.Sprintf("Building and pushing %q", p.Tag())
				action = "building and pushing"
			}

			fmt.Println(msg)

			if err := p.Build(pkgOpts...); err != nil {
				return fmt.Errorf("error %s %q: %w", action, p.Tag(), err)
			}
		}
		return nil
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force rebuild even if image is in local cache")
	cmd.Flags().BoolVar(&pull, "pull", false, "Pull image if in registry but not in local cache; conflicts with --force")
	cmd.Flags().BoolVar(&ignoreCache, "ignore-cached", false, "Ignore cached intermediate images, always pulling from registry")
	cmd.Flags().BoolVar(&docker, "docker", false, "Store the built image in the docker image cache instead of the default linuxkit cache")
	cmd.Flags().StringVar(&platforms, "platforms", "", "Which platforms to build for, defaults to all of those for which the package can be built")
	cmd.Flags().StringVar(&skipPlatforms, "skip-platforms", "", "Platforms that should be skipped, even if present in build.yml")
	cmd.Flags().StringVar(&builders, "builders", "", "Which builders to use for which platforms, e.g. linux/arm64=docker-context-arm64, overrides defaults and environment variables, see https://github.com/linuxkit/linuxkit/blob/master/docs/packages.md#Providing-native-builder-nodes")
	cmd.Flags().StringVar(&builderImage, "builder-image", defaultBuilderImage, "buildkit builder container image to use")
	cmd.Flags().BoolVar(&builderRestart, "builder-restart", false, "force restarting builder, even if container with correct name and image exists")
	cmd.Flags().Var(&cacheDir, "cache", fmt.Sprintf("Directory for caching and finding cached image, overrides env var %s", envVarCacheDir))
	cmd.Flags().StringVar(&release, "release", "", "Release the given version")
	cmd.Flags().BoolVar(&nobuild, "nobuild", false, "Skip building the image before pushing, conflicts with -force")
	cmd.Flags().BoolVar(&manifest, "manifest", true, "Create and push multi-arch manifest")
	cmd.Flags().StringVar(&sbomScanner, "sbom-scanner", "", "SBOM scanner to use, must match the buildkit spec; set to blank to use the buildkit default; set to 'false' for no scanning")
	cmd.Flags().StringVar(&dockerfile, "dockerfile", "", "Dockerfile to use for building the image, must be in this directory or below, overrides what is in build.yml")
	cmd.Flags().StringArrayVar(&buildArgFiles, "build-arg-file", nil, "Files containing build arguments, one key=value per line, contents augment and override buildArgs in build.yml. Can be specified multiple times. File is relative to working directory when running `linuxkit pkg build`")

	return cmd
}
func pkgBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "build an OCI package from a directory with a yaml configuration file",
		Long: `Build an OCI package from a directory with a yaml configuration file.
		'path' specifies the path to the package source directory.
`,
		Example: `  linuxkit pkg build [options] pkg/dir/`,
		Args:    cobra.MinimumNArgs(1),
	}
	return addCmdRunPkgBuildPush(cmd, false)
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
