package pkglib

// Thin wrappers around Docker CLI invocations

//go:generate ./gen

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/containerd/containerd/v2/core/content"
	"github.com/containerd/containerd/v2/pkg/reference"
	"github.com/docker/buildx/util/confutil"
	"github.com/docker/buildx/util/progress"
	dockercontainertypes "github.com/docker/docker/api/types/container"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	versioncompare "github.com/hashicorp/go-version"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/registry"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	buildkitClient "github.com/moby/buildkit/client"
	"github.com/moby/buildkit/cmd/buildctl/build"
	"github.com/moby/buildkit/frontend/dockerui"
	"github.com/moby/buildkit/session"
	"github.com/moby/buildkit/util/progress/progressui"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"

	// golint requires comments on non-main(test)
	// package for blank import
	dockerconfig "github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	dockerconfigtypes "github.com/docker/cli/cli/config/types"
	_ "github.com/moby/buildkit/client/connhelper/dockercontainer"
	_ "github.com/moby/buildkit/client/connhelper/ssh"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/linter"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/moby/buildkit/frontend/dockerfile/shell"
	"github.com/moby/buildkit/session/auth/authprovider"
	"github.com/moby/buildkit/session/sshforward/sshprovider"
	"github.com/moby/buildkit/session/upload/uploadprovider"
	log "github.com/sirupsen/logrus"
)

const (
	buildkitBuilderName    = "linuxkit-builder"
	buildkitSocketPath     = "/run/buildkit/buildkitd.sock"
	buildkitWaitServer     = 30 // seconds
	buildkitCheckInterval  = 1  // seconds
	sbomFrontEndKey        = "attest:sbom"
	buildkitConfigDir      = "/etc/buildkit"
	buildkitConfigFileName = "buildkitd.toml"
	buildkitConfigPath     = buildkitConfigDir + "/" + buildkitConfigFileName
)

type dockerRunnerImpl struct {
	cache bool
}

func newDockerRunner(cache bool) DockerRunner {
	return &dockerRunnerImpl{cache: cache}
}

func isExecErrNotFound(err error) bool {
	eerr, ok := err.(*exec.Error)
	if !ok {
		return false
	}
	return eerr.Err == exec.ErrNotFound
}

// these are the standard 4 build-args supported by `docker build`
// plus the all_proxy/ALL_PROXY which is a socks standard one
var proxyEnvVars = []string{
	"http_proxy",
	"https_proxy",
	"no_proxy",
	"ftp_proxy",
	"all_proxy",
	"HTTP_PROXY",
	"HTTPS_PROXY",
	"NO_PROXY",
	"FTP_PROXY",
	"ALL_PROXY",
}

func (dr *dockerRunnerImpl) command(stdin io.Reader, stdout, stderr io.Writer, args ...string) error {
	cmd := exec.Command("docker", args...)
	if stdin == nil {
		stdin = os.Stdin
	}
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		if runtime.GOOS != "windows" {
			stderr = os.Stderr
		} else {
			// On Windows directly setting stderr to os.Stderr results in the output being written to stdout,
			// corrupting the image tar. Adding an explicit indirection via a pipe works around the issue.
			r, w := io.Pipe()
			stderr = w
			go func() {
				_, _ = io.Copy(os.Stderr, r)
			}()
		}
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = stdin
	cmd.Env = os.Environ()

	log.Debugf("Executing: %v", cmd.Args)

	err := cmd.Run()
	if err != nil {
		if isExecErrNotFound(err) {
			return fmt.Errorf("linuxkit pkg requires docker to be installed")
		}
		return err
	}
	return nil
}

// versionCheck returns the client version and server version, and compares them both
// against the minimum required version.
//
//nolint:unused // will be used when linuxkit cache is eliminated and we return to docker image cache
func (dr *dockerRunnerImpl) versionCheck(version string) (string, string, error) {
	var stdout bytes.Buffer
	if err := dr.command(nil, &stdout, nil, "version", "--format", "json"); err != nil {
		return "", "", err
	}

	// we can build a struct for everything, but all we really need is .Client.Version and .Server.Version
	jsonMap := make(map[string]map[string]interface{})
	b := stdout.Bytes()
	if err := json.Unmarshal(b, &jsonMap); err != nil {
		return "", "", fmt.Errorf("unable to parse docker version output: %v, output is: %s", err, string(b))
	}
	client, ok := jsonMap["Client"]
	if !ok {
		return "", "", errors.New("docker version output did not have 'Client' field")
	}
	clientVersionInt, ok := client["Version"]
	if !ok {
		return "", "", errors.New("docker version output did not have 'Client.Version' field")
	}
	clientVersionString, ok := clientVersionInt.(string)
	if !ok {
		return "", "", errors.New("client version was not a string")
	}
	server, ok := jsonMap["Server"]
	if !ok {
		return "", "", errors.New("docker version output did not have 'Server' field")
	}
	serverVersionInt, ok := server["Version"]
	if !ok {
		return clientVersionString, "", errors.New("docker version output did not have 'Server.Version' field")
	}
	serverVersionString, ok := serverVersionInt.(string)
	if !ok {
		return clientVersionString, "", errors.New("server version was not a string")
	}

	// get the lower of each of those versions
	clientVersion, err := versioncompare.NewVersion(clientVersionString)
	if err != nil {
		return clientVersionString, serverVersionString, fmt.Errorf("invalid client version %s: %v", clientVersionString, err)
	}
	serverVersion, err := versioncompare.NewVersion(serverVersionString)
	if err != nil {
		return clientVersionString, serverVersionString, fmt.Errorf("invalid server version %s: %v", serverVersionString, err)
	}
	compareVersion, err := versioncompare.NewVersion(version)
	if err != nil {
		return clientVersionString, serverVersionString, fmt.Errorf("invalid provided version %s: %v", version, err)
	}
	if serverVersion.LessThan(compareVersion) {
		return clientVersionString, serverVersionString, fmt.Errorf("server version %s less than compare version %s", serverVersion, compareVersion)
	}
	if clientVersion.LessThan(compareVersion) {
		return clientVersionString, serverVersionString, fmt.Errorf("client version %s less than compare version %s", clientVersion, compareVersion)
	}
	return clientVersionString, serverVersionString, nil
}

// ContextSupportCheck checks if contexts are supported. This is necessary because github uses some strange versions
// of docker in Actions, which makes it difficult to tell if context is supported.
// See https://github.community/t/what-really-is-docker-3-0-6/16171
func (dr *dockerRunnerImpl) ContextSupportCheck() error {
	return dr.command(nil, io.Discard, io.Discard, "context", "ls")
}

// Builder ensure that a builder container exists or return an error.
//
// Process:
//
// 1. Get an appropriate docker context.
// 2. Using the appropriate context, try to find a docker container named `linuxkit-builder` in that context.
// 3. Return a reference to that container.
//
// To get the appropriate docker context:
//
// 1. if dockerContext is provided, try to create a builder with that context; if it succeeds, we are done; if not, return an error.
// 2. try to find an existing named runner with the pattern; if it succeeds, we are done; if not, try next.
// 3. try to create a generic builder using the default context named "linuxkit".
func (dr *dockerRunnerImpl) Builder(ctx context.Context, dockerContext, builderImage, builderConfigPath, platform string, restart bool) (*buildkitClient.Client, error) {
	// if we were given a context, we must find a builder and use it, or create one and use it
	if dockerContext != "" {
		// does the context exist?
		if err := dr.command(nil, io.Discard, io.Discard, "context", "inspect", dockerContext); err != nil {
			return nil, fmt.Errorf("provided docker context '%s' not found", dockerContext)
		}
		client, err := dr.builderEnsureContainer(ctx, buildkitBuilderName, builderImage, builderConfigPath, platform, dockerContext, restart)
		if err != nil {
			return nil, fmt.Errorf("error preparing builder based on context '%s': %v", dockerContext, err)
		}
		return client, nil
	}

	// no provided dockerContext, so look for one based on platform-specific name
	dockerContext = fmt.Sprintf("%s-%s", "linuxkit", strings.ReplaceAll(platform, "/", "-"))
	if err := dr.command(nil, io.Discard, io.Discard, "context", "inspect", dockerContext); err == nil {
		// we found an appropriately named context, so let us try to use it or error out
		if client, err := dr.builderEnsureContainer(ctx, buildkitBuilderName, builderImage, builderConfigPath, platform, dockerContext, restart); err == nil {
			return client, nil
		}
	}

	// create a generic builder
	client, err := dr.builderEnsureContainer(ctx, buildkitBuilderName, builderImage, builderConfigPath, "", "default", restart)
	if err != nil {
		return nil, fmt.Errorf("error ensuring builder container in default context: %v", err)
	}
	return client, nil
}

// builderEnsureContainer provided a name of a docker context, ensure that the builder container exists and
// is running the appropriate version of buildkit. If it does not exist, create it; if it is running
// but has the wrong version of buildkit, or not running buildkit at all, remove it and create an appropriate
// one.
// Returns a network connection to the buildkit builder in the container.
func (dr *dockerRunnerImpl) builderEnsureContainer(ctx context.Context, name, image, configPath, platform, dockerContext string, forceRestart bool) (*buildkitClient.Client, error) {
	// if no error, then we have a builder already
	// inspect it to make sure it is of the right type
	var (
		// recreate by default (true) unless we already have one that meets all of the requirements - image, permissions, etc.
		recreate = true
		// stop existing one
		stop   = false
		remove = false
		found  = false
	)

	const (

		// we will retry starting the container 3 times, waiting 1 second between each retry
		// this is to allow for race conditions, where we inspected, didn't find it,
		// some other process created it, and we are now trying to create it.
		buildkitCheckInterval   = 1 * time.Second
		buildKitCheckRetryCount = 3
	)
	for range buildKitCheckRetryCount {
		var b bytes.Buffer
		var cid string
		var filesToLoadIntoContainer map[string][]byte
		if err := dr.command(nil, &b, io.Discard, "--context", dockerContext, "container", "inspect", name); err == nil {
			// we already have a container named "linuxkit-builder" in the provided context.
			// get its state and config
			var containerJSON []dockercontainertypes.InspectResponse
			if err := json.Unmarshal(b.Bytes(), &containerJSON); err != nil || len(containerJSON) < 1 {
				return nil, fmt.Errorf("unable to read results of 'container inspect %s': %v", name, err)
			}

			cid = containerJSON[0].ID
			existingImage := containerJSON[0].Config.Image
			isRunning := containerJSON[0].State.Status == "running"
			// need to check for mounts, in case the builder-config is provided
			// by default, we assume the configPath is correct
			var configPathCorrect = true
			if configPath != "" {
				// if it is provided, we assume it is false until proven true
				log.Debugf("checking if configPath %s is correct in container %s", configPath, name)
				configPathCorrect = false
				var configB bytes.Buffer
				// we cannot exactly use the local config file, as it gets modified to get loaded into the container
				// so we preprocess it using the same library that would load it up
				filesToLoadIntoContainer, err = confutil.LoadConfigFiles(configPath)
				if err != nil {
					return nil, fmt.Errorf("failed to load buildkit config file %s: %v", configPath, err)
				}
				if err := dr.command(nil, &configB, io.Discard, "--context", dockerContext, "container", "exec", name, "cat", buildkitConfigPath); err == nil {
					// sha256sum the config file to see if it matches the provided configPath
					containerConfigFileHash := sha256.Sum256(configB.Bytes())
					log.Debugf("container %s has configPath %s with sha256sum %x", name, buildkitConfigPath, containerConfigFileHash)
					log.Tracef("container %s has configPath %s with contents:\n%s", name, buildkitConfigPath, configB.String())
					configFileContents, ok := filesToLoadIntoContainer[buildkitConfigFileName]
					if !ok {
						return nil, fmt.Errorf("unable to read provided buildkit config file %s: %v", configPath, err)
					}
					localConfigFileHash := sha256.Sum256(configFileContents)
					log.Debugf("local %s has configPath %s with sha256sum %x", name, configPath, localConfigFileHash)
					log.Tracef("local %s has configPath %s with contents:\n%s", name, buildkitConfigPath, string(configFileContents))
					if bytes.Equal(containerConfigFileHash[:], localConfigFileHash[:]) {
						log.Debugf("configPath %s in container %s matches local configPath %s", buildkitConfigPath, name, configPath)
						configPathCorrect = true
					} else {
						log.Debugf("configPath %s in container %s does not match local configPath %s", buildkitConfigPath, name, configPath)
					}
				} else {
					log.Debugf("could not read configPath %s from container %s, assuming it is not correct", buildkitConfigPath, name)
				}
			}

			switch {
			case forceRestart:
				// if forceRestart==true, we always recreate, else we check if it matches our requirements
				fmt.Printf("told to force restart, replacing existing container %s\n", name)
				recreate = true
				stop = isRunning
				remove = true
			case existingImage != image:
				// if image mismatches, recreate
				fmt.Printf("existing container %s is running image %s instead of target %s, replacing\n", name, existingImage, image)
				recreate = true
				stop = isRunning
				remove = true
			case !containerJSON[0].HostConfig.Privileged:
				// if unprivileged, we need to remove it and start a new container with the right permissions
				fmt.Printf("existing container %s is unprivileged, replacing\n", name)
				recreate = true
				stop = isRunning
				remove = true
			case !configPathCorrect:
				fmt.Printf("existing container has wrong configPath contents, restarting\n")
				recreate = true
				stop = isRunning
				remove = true
			case isRunning:
				// if already running with the right image and permissions, just use it
				fmt.Printf("using existing container %s\n", name)
				return buildkitClient.New(ctx, fmt.Sprintf("docker-container://%s?context=%s", name, dockerContext))
			default:
				// we have an existing container, but it isn't running, so start it
				// note that if it somehow got started in a parallel process or thread,
				// `container start` is a no-op, so we will get no errors; this just works.
				fmt.Printf("starting existing container %s\n", name)
				if err := dr.command(nil, io.Discard, io.Discard, "--context", dockerContext, "container", "start", name); err != nil {
					return nil, fmt.Errorf("failed to start existing container %s", name)
				}
				recreate = false
				stop = false
				remove = false
			}
		}
		// if we made it here, we need to stop and remove the container, either because of a config mismatch,
		// or because we received the CLI option
		if stop {
			if cid == "" {
				// we don't have a container ID, so we can't stop it
				return nil, fmt.Errorf("unable to stop existing container %s, no ID found", name)
			}
			if err := dr.command(nil, io.Discard, io.Discard, "--context", dockerContext, "container", "stop", cid); err != nil {
				// if we failed, do a retry; maybe it does not even exist anymore
				time.Sleep(buildkitCheckInterval)
				continue
			}
		}
		if remove {
			if cid == "" {
				// we don't have a container ID, so we can't remove it
				return nil, fmt.Errorf("unable to remove existing container %s, no ID found", name)
			}
			if err := dr.command(nil, io.Discard, io.Discard, "--context", dockerContext, "container", "rm", cid); err != nil {
				// mark the existing container as non-existent
				cid = ""
				// if we failed, do a retry; maybe it does not even exist anymore
				time.Sleep(buildkitCheckInterval)
				continue
			}
		}
		if recreate {
			// create the builder
			// this could be a single line, but it would be long. And it is easier to read when the
			// docker command args, the image name, and the image args are all on separate lines.
			args := []string{"--context", dockerContext, "container", "create", "--name", name, "--privileged"}
			args = append(args, image)
			args = append(args, "--allow-insecure-entitlement", "network.host", "--addr", fmt.Sprintf("unix://%s", buildkitSocketPath), "--debug")
			if configPath != "" {
				// set the config path explicitly
				args = append(args, "--config", buildkitConfigPath)
			}
			msg := fmt.Sprintf("creating builder container '%s' in context '%s'", name, dockerContext)
			fmt.Println(msg)
			if err := dr.command(nil, nil, io.Discard, args...); err != nil {
				// if we failed, do a retry
				time.Sleep(buildkitCheckInterval)
				continue
			}
			// copy in the buildkit config file, if provided
			if configPath != "" {
				if err := dr.copyFilesToContainer(name, filesToLoadIntoContainer); err != nil {
					return nil, fmt.Errorf("failed to copy buildkit config file %s and certificates into container %s: %v", configPath, name, err)
				}
			}

			// and now start the container
			if err := dr.command(nil, io.Discard, io.Discard, "--context", dockerContext, "container", "start", name); err != nil {
				// if we failed, do a retry; maybe it does not even exist anymore
				return nil, fmt.Errorf("failed to start newly created container %s: %v", name, err)
			}
		}
		found = true
		break
	}
	if !found {
		return nil, fmt.Errorf("unable to create or find builder container %s in context %s after %d retries", name, dockerContext, buildKitCheckRetryCount)
	}

	// wait for buildkit socket to be ready up to the timeout
	fmt.Printf("waiting for buildkit builder to be ready, up to %d seconds\n", buildkitWaitServer)
	timeout := time.After(buildkitWaitServer * time.Second)
	ticker := time.NewTicker(buildkitCheckInterval)
	// Keep trying until we're timed out or get a success
	for {
		select {
		// Got a timeout! fail with a timeout error
		case <-timeout:
			return nil, fmt.Errorf("could not communicate with buildkit builder at context/container %s/%s after %d seconds", dockerContext, name, buildkitWaitServer)
			// Got a tick, we should try again
		case <-ticker.C:
			client, err := buildkitClient.New(ctx, fmt.Sprintf("docker-container://%s?context=%s", name, dockerContext))
			if err == nil {
				_, err = client.Info(ctx)
				if err == nil {
					fmt.Println("buildkit builder ready!")
					return client, nil
				}
				_ = client.Close()
			}

			// got an error, wait 1 second and try again
			log.Debugf("buildkitclient error: %v, waiting %d seconds and trying again", err, buildkitCheckInterval)
		}
	}
}

func (dr *dockerRunnerImpl) Pull(img string) (bool, error) {
	err := dr.command(nil, nil, nil, "image", "pull", img)
	if err == nil {
		return true, nil
	}
	switch err.(type) {
	case *exec.ExitError:
		return false, nil
	default:
		return false, err
	}
}

//nolint:unused // will be used when linuxkit cache is eliminated and we return to docker image cache
func (dr dockerRunnerImpl) push(img string) error {
	return dr.command(nil, nil, nil, "image", "push", img)
}

//nolint:unused // will be used when linuxkit cache is eliminated and we return to docker image cache
func (dr *dockerRunnerImpl) pushWithManifest(img, suffix string, pushImage, pushManifest bool) error {
	var err error
	if pushImage {
		fmt.Printf("Pushing %s\n", img+suffix)
		if err := dr.push(img + suffix); err != nil {
			return err
		}
	} else {
		fmt.Print("Image push disabled, skipping...\n")
	}

	if pushManifest {
		fmt.Printf("Pushing %s to manifest %s\n", img+suffix, img)
		_, _, err = registry.PushManifest(img, remote.WithAuthFromKeychain(authn.DefaultKeychain))
		if err != nil {
			return err
		}
	} else {
		fmt.Print("Manifest push disabled, skipping...\n")
	}
	return nil
}

func (dr *dockerRunnerImpl) Tag(ref, tag string) error {
	fmt.Printf("Tagging %s as %s\n", ref, tag)
	return dr.command(nil, nil, nil, "image", "tag", ref, tag)
}

func (dr *dockerRunnerImpl) Build(ctx context.Context, tag, pkg, dockerContext, builderImage, builderConfigPath, platform string, restart, preCacheImages bool, c spec.CacheProvider, stdin io.Reader, stdout io.Writer, sbomScan bool, sbomScannerImage, progressType string, imageBuildOpts spec.ImageBuildOptions) error {
	// ensure we have a builder
	client, err := dr.Builder(ctx, dockerContext, builderImage, builderConfigPath, platform, restart)
	if err != nil {
		return fmt.Errorf("unable to ensure builder container: %v", err)
	}

	frontendAttrs := map[string]string{}

	for _, proxyVarName := range proxyEnvVars {
		if value, ok := os.LookupEnv(proxyVarName); ok {
			frontendAttrs[proxyVarName] = value
		}
	}
	// platform
	frontendAttrs["platform"] = platform

	// build-args
	for k, v := range imageBuildOpts.BuildArgs {
		frontendAttrs[fmt.Sprintf("build-arg:%s", k)] = *v
	}

	// no-cache option
	if !dr.cache {
		frontendAttrs["no-cache"] = ""
	}

	// network
	// translate to net modes understood by buildkit dockerfile frontend
	switch imageBuildOpts.NetworkMode {
	case "host", "none":
		frontendAttrs["force-network-mode"] = imageBuildOpts.NetworkMode
	case "default":
		frontendAttrs["force-network-mode"] = "sandbox"
	default:
		return fmt.Errorf("unsupported network mode %q", imageBuildOpts.NetworkMode)
	}

	for k, v := range imageBuildOpts.Labels {
		frontendAttrs[fmt.Sprintf("label:%s", k)] = v
	}

	if sbomScan {
		var sbomValue string
		if sbomScannerImage != "" {
			sbomValue = fmt.Sprintf("generator=%s", sbomScannerImage)
		}
		frontendAttrs[sbomFrontEndKey] = sbomValue
	}

	attachable := []session.Attachable{}
	localDirs := map[string]string{}

	// Add SSH agent provider if needed
	if len(imageBuildOpts.SSH) > 0 {
		configs, err := build.ParseSSH(imageBuildOpts.SSH)
		if err != nil {
			return err
		}
		sp, err := sshprovider.NewSSHAgentProvider(configs)
		if err != nil {
			return err
		}
		attachable = append(attachable, sp)
	}

	if stdin != nil {
		buf := io.NopCloser(bufio.NewReader(stdin))
		up := uploadprovider.New()
		frontendAttrs["context"] = up.Add(io.NopCloser(buf))
		attachable = append(attachable, up)
	} else {
		localDirs[dockerui.DefaultLocalNameDockerfile] = pkg
		localDirs[dockerui.DefaultLocalNameContext] = pkg
	}
	// add credentials
	var cf *configfile.ConfigFile
	if len(imageBuildOpts.RegistryAuths) > 0 {
		// if static ones were provided, use those
		cf = configfile.New("custom")
		// merge imageBuildOpts.RegistryAuths into dockercfg
		for registry, auth := range imageBuildOpts.RegistryAuths {
			// special case for docker.io
			registryWithoutScheme := strings.TrimPrefix(registry, "https://")
			registryWithoutScheme = strings.TrimPrefix(registryWithoutScheme, "http://")
			if registryWithoutScheme == "docker.io" || registryWithoutScheme == "index.docker.io" || registryWithoutScheme == "registry-1.docker.io" {
				registry = "https://index.docker.io/v1/"
			}
			cf.AuthConfigs[registry] = dockerconfigtypes.AuthConfig{
				Username:      auth.Username,
				Password:      auth.Password,
				RegistryToken: auth.RegistryToken,
			}
		}
	} else {
		// Else use Docker authentication provider so BuildKit can use ~/.docker/config.json or OS-specific credential helpers.
		cf = dockerconfig.LoadDefaultConfigFile(io.Discard)
	}
	attachable = append(attachable,
		authprovider.NewDockerAuthProvider(authprovider.DockerAuthProviderConfig{ConfigFile: cf}),
	)

	solveOpts := buildkitClient.SolveOpt{
		Frontend:      "dockerfile.v0",
		FrontendAttrs: frontendAttrs,
		Exports: []buildkitClient.ExportEntry{
			{
				Type: buildkitClient.ExporterOCI,
				Attrs: map[string]string{
					"name": tag,
				},
				Output: fixedWriteCloser(&writeNopCloser{stdout}),
			},
		},
		Session:   attachable,
		LocalDirs: localDirs,
	}

	frontendAttrs["filename"] = imageBuildOpts.Dockerfile

	// go through the dockerfile to see if we have any provided images cached
	// and if we should cache any
	if c != nil {
		dockerfileRef := path.Join(pkg, imageBuildOpts.Dockerfile)
		f, err := os.Open(dockerfileRef)
		if err != nil {
			return fmt.Errorf("error opening dockerfile %s: %v", dockerfileRef, err)
		}
		defer func() { _ = f.Close() }()
		ast, err := parser.Parse(f)
		if err != nil {
			return fmt.Errorf("error parsing dockerfile from bytes into AST %s: %v", dockerfileRef, err)
		}
		stages, metaArgs, err := instructions.Parse(ast.AST, linter.New(&linter.Config{}))
		if err != nil {
			return fmt.Errorf("error parsing dockerfile from AST into stages %s: %v", dockerfileRef, err)
		}

		// fill optMetaArgs with args found while parsing Dockerfile
		optMetaArgs := make(map[string]string)
		for _, cmd := range metaArgs {
			for _, metaArg := range cmd.Args {
				optMetaArgs[metaArg.Key] = metaArg.ValueString()
			}
		}
		// replace parsed args with provided BuildArgs if keys found
		for k, v := range imageBuildOpts.BuildArgs {
			if _, found := optMetaArgs[k]; found {
				optMetaArgs[k] = *v
			}
		}
		optMetaArgsSlice := make([]string, 0, len(optMetaArgs))
		for k, v := range optMetaArgs {
			optMetaArgsSlice = append(optMetaArgsSlice, fmt.Sprintf("%s=%s", k, v))
		}

		shlex := shell.NewLex(ast.EscapeToken)
		// go through each stage, get the basename of the image, see if we have it in the linuxkit cache
		imageStores := map[string]string{}
		for _, stage := range stages {
			// check if we have args in FROM and replace them:
			//   ARG IMAGE=linuxkit/img
			//   FROM ${IMAGE} as src
			// will be parsed as:
			//   FROM linuxkit/img as src
			name, _, err := shlex.ProcessWord(stage.BaseName, shell.EnvsFromSlice(optMetaArgsSlice))
			if err != nil {
				return fmt.Errorf("could not process word for image %s: %v", stage.BaseName, err)
			}
			if name == "" {
				return fmt.Errorf("base name (%s) should not be blank", stage.BaseName)
			}
			// see if the provided image name is tagged (docker.io/linuxkit/foo:latest) or digested (docker.io/linuxkit/foo@sha256:abcdefg)
			// if neither, we have an error
			ref, err := reference.Parse(util.ReferenceExpand(name))
			if err != nil {
				return fmt.Errorf("could not resolve references for image %s: %v", name, err)
			}
			gdesc, err := c.FindDescriptor(&ref)
			if err != nil {
				return fmt.Errorf("invalid name %s", name)
			}
			// 3 possibilities:
			// 1. we found it, so we can use it
			// 2. we did not find it, but we were told to pre-cache images, so we pull it down and then use it
			// 3. we did not find it, and we were not told to pre-cache images, so we just skip it
			switch {
			case gdesc == nil && !preCacheImages:
				log.Debugf("image %s not found in cache, buildkit will pull directly", name)
				continue
			case gdesc == nil && preCacheImages:
				log.Debugf("image %s not found in cache, pulling to pre-cache", name)
				parts := strings.SplitN(platform, "/", 2)
				if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
					return fmt.Errorf("invalid platform %s, expected format os/arch", platform)
				}
				plats := []imagespec.Platform{{OS: parts[0], Architecture: parts[1]}}

				if err := c.ImagePull(&ref, plats, false); err != nil {
					return fmt.Errorf("unable to pull image %s for caching: %v", name, err)
				}
				gdesc2, err := c.FindDescriptor(&ref)
				if err != nil {
					return fmt.Errorf("invalid name %s", name)
				}
				if gdesc2 == nil {
					return fmt.Errorf("image %s not found in cache after pulling", name)
				}
				imageStores[name] = gdesc2.Digest.String()
			default:
				log.Debugf("image %s found in cache", name)
				imageStores[name] = gdesc.Digest.String()
			}
		}
		if len(imageStores) > 0 {
			// if we made it here, we found the reference
			store, err := c.Store()
			if err != nil {
				return fmt.Errorf("unable to get content store from cache: %v", err)
			}
			if solveOpts.OCIStores == nil {
				solveOpts.OCIStores = map[string]content.Store{}
			}
			solveOpts.OCIStores["linuxkit-cache"] = store
			for image, hash := range imageStores {
				solveOpts.FrontendAttrs["context:"+image] = fmt.Sprintf("oci-layout:%s@%s", "linuxkit-cache", hash)
			}
		}
	}

	ctx2, cancel := context.WithCancel(context.TODO())
	defer cancel()
	buildkitProgressType := progressui.DisplayMode(progressType)
	if buildkitProgressType == progressui.DefaultMode {
		buildkitProgressType = progressui.AutoMode
	}
	printer, err := progress.NewPrinter(ctx2, os.Stderr, buildkitProgressType)
	if err != nil {
		return fmt.Errorf("unable to create progress printer: %v", err)
	}
	pw := progress.WithPrefix(printer, "", false)
	ch, done := progress.NewChannel(pw)
	defer func() { <-done }()

	fmt.Printf("building for platform %s\n", platform)

	_, err = client.Solve(ctx, nil, solveOpts, ch)
	return err
}

func (dr *dockerRunnerImpl) Save(tgt string, refs ...string) error {
	args := append([]string{"image", "save", "-o", tgt}, refs...)
	return dr.command(nil, nil, nil, args...)
}

func (dr *dockerRunnerImpl) Load(src io.Reader) error {
	args := []string{"image", "load"}
	return dr.command(src, nil, nil, args...)
}

func (dr *dockerRunnerImpl) copyFilesToContainer(containerID string, files map[string][]byte) error {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)

	for path, content := range files {
		hdr := &tar.Header{
			Name:     path,
			Mode:     0644,
			Size:     int64(len(content)),
			ModTime:  time.Now(),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("write tar header: %w", err)
		}
		if _, err := tw.Write(content); err != nil {
			return fmt.Errorf("write tar content: %w", err)
		}
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("close tar: %w", err)
	}

	// Send the TAR archive to the container at /
	return dr.command(&buf, os.Stdout, os.Stderr, "container", "cp", "-", containerID+":"+buildkitConfigDir)
}

func fixedWriteCloser(wc io.WriteCloser) func(map[string]string) (io.WriteCloser, error) {
	return func(map[string]string) (io.WriteCloser, error) {
		return wc, nil
	}
}

type writeNopCloser struct {
	writer io.Writer
}

func (w *writeNopCloser) Close() error {
	return nil
}
func (w *writeNopCloser) Write(p []byte) (n int, err error) {
	return w.writer.Write(p)
}
