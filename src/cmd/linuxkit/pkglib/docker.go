package pkglib

// Thin wrappers around Docker CLI invocations

//go:generate ./gen

import (
	"bufio"
	"bytes"
	"context"
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

	"github.com/containerd/containerd/content"
	"github.com/containerd/containerd/reference"
	"github.com/docker/buildx/util/progress"
	"github.com/docker/docker/api/types"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	versioncompare "github.com/hashicorp/go-version"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/registry"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	buildkitClient "github.com/moby/buildkit/client"

	// golint requires comments on non-main(test)
	// package for blank import
	_ "github.com/moby/buildkit/client/connhelper/dockercontainer"
	_ "github.com/moby/buildkit/client/connhelper/ssh"
	"github.com/moby/buildkit/frontend/dockerfile/builder"
	"github.com/moby/buildkit/frontend/dockerfile/instructions"
	"github.com/moby/buildkit/frontend/dockerfile/parser"
	"github.com/moby/buildkit/frontend/dockerfile/shell"
	"github.com/moby/buildkit/session/upload/uploadprovider"
	log "github.com/sirupsen/logrus"
)

const (
	buildkitBuilderName   = "linuxkit-builder"
	buildkitSocketPath    = "/run/buildkit/buildkitd.sock"
	buildkitWaitServer    = 30 // seconds
	buildkitCheckInterval = 1  // seconds
	sbomFrontEndKey       = "attest:sbom"
)

type dockerRunner interface {
	tag(ref, tag string) error
	build(ctx context.Context, tag, pkg, dockerfile, dockerContext, builderImage, platform string, restart bool, c spec.CacheProvider, r io.Reader, stdout io.Writer, sbomScan bool, sbomScannerImage string, imageBuildOpts types.ImageBuildOptions) error
	save(tgt string, refs ...string) error
	load(src io.Reader) error
	pull(img string) (bool, error)
	contextSupportCheck() error
	builder(ctx context.Context, dockerContext, builderImage, platform string, restart bool) (*buildkitClient.Client, error)
}

type dockerRunnerImpl struct {
	cache bool
}

func newDockerRunner(cache bool) dockerRunner {
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

// contextCheck checks if contexts are supported. This is necessary because github uses some strange versions
// of docker in Actions, which makes it difficult to tell if context is supported.
// See https://github.community/t/what-really-is-docker-3-0-6/16171
func (dr *dockerRunnerImpl) contextSupportCheck() error {
	return dr.command(nil, io.Discard, io.Discard, "context", "ls")
}

// builder ensure that a builder container exists or return an error.
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
func (dr *dockerRunnerImpl) builder(ctx context.Context, dockerContext, builderImage, platform string, restart bool) (*buildkitClient.Client, error) {
	// if we were given a context, we must find a builder and use it, or create one and use it
	if dockerContext != "" {
		// does the context exist?
		if err := dr.command(nil, io.Discard, io.Discard, "context", "inspect", dockerContext); err != nil {
			return nil, fmt.Errorf("provided docker context '%s' not found", dockerContext)
		}
		client, err := dr.builderEnsureContainer(ctx, buildkitBuilderName, builderImage, platform, dockerContext, restart)
		if err != nil {
			return nil, fmt.Errorf("error preparing builder based on context '%s': %v", dockerContext, err)
		}
		return client, nil
	}

	// no provided dockerContext, so look for one based on platform-specific name
	dockerContext = fmt.Sprintf("%s-%s", "linuxkit", strings.ReplaceAll(platform, "/", "-"))
	if err := dr.command(nil, io.Discard, io.Discard, "context", "inspect", dockerContext); err == nil {
		// we found an appropriately named context, so let us try to use it or error out
		if client, err := dr.builderEnsureContainer(ctx, buildkitBuilderName, builderImage, platform, dockerContext, restart); err == nil {
			return client, nil
		}
	}

	// create a generic builder
	client, err := dr.builderEnsureContainer(ctx, buildkitBuilderName, builderImage, "", "default", restart)
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
func (dr *dockerRunnerImpl) builderEnsureContainer(ctx context.Context, name, image, platform, dockerContext string, forceRestart bool) (*buildkitClient.Client, error) {
	// if no error, then we have a builder already
	// inspect it to make sure it is of the right type
	var (
		// recreate by default (true) unless we already have one that meets all of the requirements - image, permissions, etc.
		recreate = true
		// stop existing one
		stop   = false
		remove = false
		b      bytes.Buffer
	)

	if err := dr.command(nil, &b, io.Discard, "--context", dockerContext, "container", "inspect", name); err == nil {
		// we already have a container named "linuxkit-builder" in the provided context.
		// get its state and config
		var containerJSON []types.ContainerJSON
		if err := json.Unmarshal(b.Bytes(), &containerJSON); err != nil || len(containerJSON) < 1 {
			return nil, fmt.Errorf("unable to read results of 'container inspect %s': %v", name, err)
		}

		existingImage := containerJSON[0].Config.Image
		isRunning := containerJSON[0].State.Status == "running"

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
		case isRunning:
			// if already running with the right image and permissions, just use it
			fmt.Printf("using existing container %s\n", name)
			return buildkitClient.New(ctx, fmt.Sprintf("docker-container://%s?context=%s", name, dockerContext))
		default:
			// we have an existing container, but it isn't running, so start it
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
		if err := dr.command(nil, io.Discard, io.Discard, "--context", dockerContext, "container", "stop", name); err != nil {
			return nil, fmt.Errorf("failed to stop existing container %s", name)
		}
	}
	if remove {
		if err := dr.command(nil, io.Discard, io.Discard, "--context", dockerContext, "container", "rm", name); err != nil {
			return nil, fmt.Errorf("failed to remove existing container %s", name)
		}
	}
	if recreate {
		// create the builder
		args := []string{"--context", dockerContext, "container", "run", "-d", "--name", name, "--privileged", image, "--allow-insecure-entitlement", "network.host", "--addr", fmt.Sprintf("unix://%s", buildkitSocketPath), "--debug"}
		msg := fmt.Sprintf("creating builder container '%s' in context '%s'", name, dockerContext)
		fmt.Println(msg)
		if err := dr.command(nil, nil, nil, args...); err != nil {
			return nil, err
		}
	}
	// wait for buildkit socket to be ready up to the timeout
	fmt.Printf("waiting for buildkit builder to be ready, up to %d seconds\n", buildkitWaitServer)
	timeout := time.After(buildkitWaitServer * time.Second)
	ticker := time.NewTicker(buildkitCheckInterval * time.Second)
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

func (dr *dockerRunnerImpl) pull(img string) (bool, error) {
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

func (dr *dockerRunnerImpl) tag(ref, tag string) error {
	fmt.Printf("Tagging %s as %s\n", ref, tag)
	return dr.command(nil, nil, nil, "image", "tag", ref, tag)
}

func (dr *dockerRunnerImpl) build(ctx context.Context, tag, pkg, dockerfile, dockerContext, builderImage, platform string, restart bool, c spec.CacheProvider, stdin io.Reader, stdout io.Writer, sbomScan bool, sbomScannerImage string, imageBuildOpts types.ImageBuildOptions) error {
	// ensure we have a builder
	client, err := dr.builder(ctx, dockerContext, builderImage, platform, restart)
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
	}

	if stdin != nil {
		buf := bufio.NewReader(stdin)
		up := uploadprovider.New()
		frontendAttrs["context"] = up.Add(buf)
		solveOpts.Session = append(solveOpts.Session, up)
	} else {
		solveOpts.LocalDirs = map[string]string{
			builder.DefaultLocalNameDockerfile: pkg,
			builder.DefaultLocalNameContext:    pkg,
		}
	}
	frontendAttrs["filename"] = dockerfile

	// go through the dockerfile to see if we have any provided images cached
	if c != nil {
		dockerfileRef := path.Join(pkg, dockerfile)
		f, err := os.Open(dockerfileRef)
		if err != nil {
			return fmt.Errorf("error opening dockerfile %s: %v", dockerfileRef, err)
		}
		defer f.Close()
		ast, err := parser.Parse(f)
		if err != nil {
			return fmt.Errorf("error parsing dockerfile from bytes into AST %s: %v", dockerfileRef, err)
		}
		stages, metaArgs, err := instructions.Parse(ast.AST)
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

		shlex := shell.NewLex(ast.EscapeToken)
		// go through each stage, get the basename of the image, see if we have it in the linuxkit cache
		imageStores := map[string]string{}
		for _, stage := range stages {
			// check if we have args in FROM and replace them:
			//   ARG IMAGE=linuxkit/img
			//   FROM ${IMAGE} as src
			// will be parsed as:
			//   FROM linuxkit/img as src
			name, err := shlex.ProcessWordWithMap(stage.BaseName, optMetaArgs)
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
			// not found, so nothing to look up
			if gdesc == nil {
				continue
			}
			hash := gdesc.Digest
			imageStores[name] = hash.String()
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
	printer := progress.NewPrinter(ctx2, os.Stderr, os.Stderr, "auto")
	pw := progress.WithPrefix(printer, "", false)
	ch, done := progress.NewChannel(pw)
	defer func() { <-done }()

	fmt.Printf("building for platform %s\n", platform)

	_, err = client.Solve(ctx, nil, solveOpts, ch)
	return err
}

func (dr *dockerRunnerImpl) save(tgt string, refs ...string) error {
	args := append([]string{"image", "save", "-o", tgt}, refs...)
	return dr.command(nil, nil, nil, args...)
}

func (dr *dockerRunnerImpl) load(src io.Reader) error {
	args := []string{"image", "load"}
	return dr.command(src, nil, nil, args...)
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
