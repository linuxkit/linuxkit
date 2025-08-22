package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/pkglib"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
)

const (
	buildersEnvVar      = "LINUXKIT_BUILDERS"
	envVarCacheDir      = "LINUXKIT_CACHE"
	defaultBuilderImage = "moby/buildkit:v0.23.1"
)

type pkgBuilder struct {
	// flags
	force          bool
	pull           bool
	push           bool
	ignoreCache    bool
	docker         bool
	verbose        bool
	dry            bool
	platforms      string
	skipPlatforms  string
	builders       string
	builderImage   string
	builderConfig  string
	builderRestart bool
	preCacheImages bool
	release        string
	nobuild        bool
	manifest       bool
	cacheDir       flagOverEnvVarOverDefaultString
	sbomScanner    string
	dockerfile     string
	buildArgFiles  []string
	progress       string
	ssh            []string

	// build vars
	pkgs []pkglib.Pkg
}

func (pb *pkgBuilder) build(args []string) error {
	var err error
	pb.pkgs, err = pkglib.NewFromConfig(pkglibConfig, args...)
	if err != nil {
		return err
	}

	err = pb.checkFlagConflicts()
	if err != nil {
		return err
	}

	var opts []pkglib.BuildOpt
	opts = pb.convertCmdFlagsToDockerOpts(opts)

	// read any build arg files
	opts, err = pb.readAnyBuildArgFiles(opts)
	if err != nil {
		return err
	}

	// also need to parse the build args from the build.yml file for any special linuxkit values
	err = pb.buildArgsFromBuildYml()
	if err != nil {
		return err
	}

	// skipPlatformsMap contains platforms that should be skipped
	skipPlatformsMap := make(map[string]bool)
	if pb.skipPlatforms != "" {
		for _, platform := range strings.Split(pb.skipPlatforms, ",") {
			parts := strings.SplitN(platform, "/", 2)
			if len(parts) != 2 || parts[0] == "" || parts[0] != "linux" || parts[1] == "" {
				return fmt.Errorf("invalid target platform specification '%s'", platform)
			}
			skipPlatformsMap[strings.Trim(parts[1], " ")] = true
		}
	}
	// if requested specific platforms, build those. If not, then we will
	// retrieve the defaults in the loop over each package.
	var plats []imagespec.Platform
	// don't allow the use of --skip-platforms with --platforms
	if pb.platforms != "" && pb.skipPlatforms != "" {
		return errors.New("--skip-platforms and --platforms may not be used together")
	}
	// process the platforms if provided
	if pb.platforms != "" {
		for _, p := range strings.Split(pb.platforms, ",") {
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
	buildersMap, err = buildPlatformBuildersMap(pb.builders, buildersMap)
	if err != nil {
		return fmt.Errorf("error in --builders flag: %w", err)
	}
	if pb.builderConfig != "" {
		if _, err := os.Stat(pb.builderConfig); err != nil {
			return fmt.Errorf("error reading builder config file %s: %w", pb.builderConfig, err)
		}
		opts = append(opts, pkglib.WithBuildBuilderConfig(pb.builderConfig))
	}

	opts = append(opts, pkglib.WithBuildBuilders(buildersMap))
	opts = append(opts, pkglib.WithBuildBuilderImage(pb.builderImage))
	opts = append(opts, pkglib.WithBuildBuilderRestart(pb.builderRestart))
	opts = append(opts, pkglib.WithProgress(pb.progress))
	if len(pb.ssh) > 0 {
		opts = append(opts, pkglib.WithSSH(pb.ssh))
	}
	if len(registryCreds) > 0 {
		registryCredMap := make(map[string]spec.RegistryAuth)
		for _, cred := range registryCreds {
			parts := strings.SplitN(cred, "=", 2)
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				return fmt.Errorf("invalid registry auth specification '%s'", cred)
			}
			registryPart := strings.TrimSpace(parts[0])
			authPart := strings.TrimSpace(parts[1])
			var auth spec.RegistryAuth
			// if the auth is a token, we don't need a username
			credParts := strings.SplitN(authPart, ":", 2)
			var userPart, credPart string
			userPart = strings.TrimSpace(credParts[0])
			if len(credParts) == 2 {
				credPart = strings.TrimSpace(credParts[1])
			}
			switch {
			case len(registryPart) == 0:
				return fmt.Errorf("invalid registry auth specification '%s', registry must not be blank", cred)
			case len(credParts) == 2 && (len(userPart) == 0 || len(credPart) == 0):
				return fmt.Errorf("invalid registry auth specification '%s', username and password must not be blank", cred)
			case len(credParts) == 1 && len(userPart) == 0:
				return fmt.Errorf("invalid registry auth specification '%s', token must not be blank", cred)
			case len(credParts) == 2:
				auth = spec.RegistryAuth{
					Username: userPart,
					Password: credPart,
				}
			case len(credParts) == 1:
				auth = spec.RegistryAuth{
					RegistryToken: authPart,
				}
			default:
				return fmt.Errorf("invalid registry auth specification '%s'", cred)
			}
			registryCredMap[registryPart] = auth
		}
		opts = append(opts, pkglib.WithRegistryAuth(registryCredMap))
	}

	for _, p := range pb.pkgs {
		// things we need our own copies of
		var (
			pkgOpts  = make([]pkglib.BuildOpt, len(opts))
			pkgPlats = make([]imagespec.Platform, len(plats))
		)
		copy(pkgOpts, opts)
		copy(pkgPlats, plats)
		// unless overridden, platforms are specific to a package, so this needs to be inside the for loop
		if len(pkgPlats) == 0 {
			for _, a := range p.Arches {
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
		case !pb.push:
			msg = fmt.Sprintf("Building %q", p.Tag())
			action = "building"
		case pb.nobuild:
			msg = fmt.Sprintf("Pushing %q without building", p.Tag())
			action = "building and pushing"
		default:
			msg = fmt.Sprintf("Building and pushing %q", p.Tag())
			action = "building and pushing"
		}

		if pb.verbose {
			bs, err := json.MarshalIndent(p, "", "    ")
			if err != nil {
				log.Fatalf("could not marshal: %+v", err)
			}
			fmt.Println(string(bs))
		} else {
			fmt.Println(msg)
		}

		if !pb.dry {
			if err := p.Build(pkgOpts...); err != nil {
				return fmt.Errorf("error %s %q: %w", action, p.Tag(), err)
			}
		}
	}
	return nil

}

func (pb *pkgBuilder) buildArgsFromBuildYml() error {
	for i := range pb.pkgs {
		if err := pb.pkgs[i].ProcessBuildArgs(); err != nil {
			return fmt.Errorf("error processing build args for package %q: %w", pb.pkgs[i].Tag(), err)
		}
	}
	return nil
}

func (pb *pkgBuilder) convertCmdFlagsToDockerOpts(opts []pkglib.BuildOpt) []pkglib.BuildOpt {
	if pb.force {
		opts = append(opts, pkglib.WithBuildForce())
	}
	if pb.ignoreCache {
		opts = append(opts, pkglib.WithBuildIgnoreCache())
	}
	if pb.preCacheImages {
		opts = append(opts, pkglib.WithPreCacheImages())
	}
	if pb.pull {
		opts = append(opts, pkglib.WithBuildPull())
	}

	opts = append(opts, pkglib.WithBuildCacheDir(pb.cacheDir.String()))

	if pb.push {
		opts = append(opts, pkglib.WithBuildPush())
		if pb.nobuild {
			opts = append(opts, pkglib.WithBuildSkip())
		}
		if pb.release != "" {
			opts = append(opts, pkglib.WithRelease(pb.release))
		}
		if pb.manifest {
			opts = append(opts, pkglib.WithBuildManifest())
		}
	}
	if pb.docker {
		opts = append(opts, pkglib.WithBuildTargetDockerCache())
	}

	if pb.sbomScanner != "false" {
		opts = append(opts, pkglib.WithBuildSbomScanner(pb.sbomScanner))
	}
	opts = append(opts, pkglib.WithDockerfile(pb.dockerfile))
	return opts
}

func (pb *pkgBuilder) readAnyBuildArgFiles(opts []pkglib.BuildOpt) ([]pkglib.BuildOpt, error) {
	var buildArgs []string
	for _, filename := range pb.buildArgFiles {
		f, err := os.Open(filename)
		if err != nil {
			return nil, fmt.Errorf("error opening build args file %s: %w", filename, err)
		}
		defer func() { _ = f.Close() }()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			// check if the value is a special linuxkit value
			buildArg, err := pkglib.TransformBuildArgValue(line, filename)
			if err != nil {
				return nil, fmt.Errorf("error transforming build arg %s: %v", line, err)
			}

			buildArgs = append(buildArgs, buildArg...)
		}
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("error reading build args file %s: %w", filename, err)
		}
	}
	opts = append(opts, pkglib.WithBuildArgs(buildArgs))
	return opts, nil
}

func (pb *pkgBuilder) checkFlagConflicts() error {
	if pb.nobuild && pb.force {
		return errors.New("flags -force and -nobuild conflict")
	}
	if pb.pull && pb.force {
		return errors.New("flags -force and -pull conflict")
	}
	return nil
}

// some logic clarification:
// pkg build                           - builds unless is in cache or published in registry
// pkg build --pull                    - builds unless is in cache or published in registry; pulls from registry if not in cache
// pkg build --force                   - always builds even if is in cache or published in registry
// pkg build --force --pull            - always builds even if is in cache or published in registry; --pull ignored
// pkg build --push 		           - always builds unless is in cache or published in registry; pushes to registry
// pkg build --push --force            - always builds even if is in cache
// pkg build --push --nobuild          - skips build; if not in cache, fails
// pkg build --push --nobuild --force  - nonsensical
// pkg push                            - equivalent to pkg build --push

func pkgBuildCmd() *cobra.Command {
	b := pkgBuilder{}
	b.cacheDir = flagOverEnvVarOverDefaultString{def: defaultLinuxkitCache(), envVar: envVarCacheDir}
	cmd := &cobra.Command{
		Use:   "build",
		Short: "build an OCI package from a directory with a yaml configuration file",
		Long: `Build an OCI package from a directory with a yaml configuration file.
		'path' specifies the path to the package source directory.
`,
		Example: `  linuxkit pkg build [options] pkg/dir/`,
		Args:    cobra.MinimumNArgs(1),
		RunE:    func(cmd *cobra.Command, args []string) error { return b.build(args) },
	}
	cmd.Flags().BoolVar(&b.force, "force", false, "Force rebuild even if image is in local cache")
	cmd.Flags().BoolVar(&b.verbose, "verbose", false, "Print extra output as json before build")
	cmd.Flags().BoolVar(&b.dry, "dry", false, "Do not build, mostly makes sense to use in conjunction with '--verbose'")
	cmd.Flags().BoolVar(&b.pull, "pull", false, "Pull image if in registry but not in local cache; conflicts with --force")
	cmd.Flags().BoolVar(&b.push, "push", false, "After building, if successful, push the image to the registry; if --nobuild is set, just push")
	cmd.Flags().BoolVar(&b.ignoreCache, "ignore-cached", false, "Ignore cached intermediate images, always pulling from registry")
	cmd.Flags().BoolVar(&b.docker, "docker", false, "Store the built image in the docker image cache instead of the default linuxkit cache")
	cmd.Flags().StringVar(&b.platforms, "platforms", "", "Which platforms to build for, defaults to all of those for which the package can be built")
	cmd.Flags().StringVar(&b.skipPlatforms, "skip-platforms", "", "Platforms that should be skipped, even if present in build.yml")
	cmd.Flags().StringVar(&b.builders, "builders", "", "Which builders to use for which platforms, e.g. linux/arm64=docker-context-arm64, overrides defaults and environment variables, see https://github.com/linuxkit/linuxkit/blob/master/docs/packages.md#Providing-native-builder-nodes")
	cmd.Flags().StringVar(&b.builderImage, "builder-image", defaultBuilderImage, "buildkit builder container image to use")
	cmd.Flags().StringVar(&b.builderConfig, "builder-config", "", "path to buildkit builder config.toml file to use, overrides the default config.toml in the builder image. When provided, copied over into builder, along with all certs. Use paths for certificates relative to your local host, they will be adjusted on copying into the container. USE WITH CAUTION")
	cmd.Flags().BoolVar(&b.builderRestart, "builder-restart", false, "force restarting builder, even if container with correct name and image exists")
	cmd.Flags().BoolVar(&b.preCacheImages, "precache-images", false, "download all referenced images in the Dockerfile to the linuxkit cache before building, thus referencing the local cache instead of pulling from the registry; this is useful for handling mirrors and special connections")
	cmd.Flags().Var(&b.cacheDir, "cache", fmt.Sprintf("Directory for caching and finding cached image, overrides env var %s", envVarCacheDir))
	cmd.Flags().StringVar(&b.release, "release", "", "Release the given version")
	cmd.Flags().BoolVar(&b.nobuild, "nobuild", false, "Skip building the image before pushing, conflicts with -force")
	cmd.Flags().BoolVar(&b.manifest, "manifest", true, "Create and push multi-arch manifest")
	cmd.Flags().StringVar(&b.sbomScanner, "sbom-scanner", "", "SBOM scanner to use, must match the buildkit spec; set to blank to use the buildkit default; set to 'false' for no scanning")
	cmd.Flags().StringVar(&b.dockerfile, "dockerfile", "", "Dockerfile to use for building the image, must be in this directory or below, overrides what is in build.yml")
	cmd.Flags().StringArrayVar(&b.buildArgFiles, "build-arg-file", nil, "Files containing build arguments, one key=value per line, contents augment and override buildArgs in build.yml. Can be specified multiple times. File is relative to working directory when running `linuxkit pkg build`")
	cmd.Flags().StringVar(&b.progress, "progress", "auto", "Set type of progress output (auto, plain, tty). Use plain to show container output, tty for interactive build")
	cmd.Flags().StringArrayVar(&b.ssh, "ssh", nil, "SSH agent config to use for build, follows the syntax used for buildx and buildctl, see https://docs.docker.com/reference/dockerfile/#run---mounttypessh")

	return cmd
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
