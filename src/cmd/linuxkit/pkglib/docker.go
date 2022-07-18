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
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/docker/buildx/util/progress"
	"github.com/docker/docker/api/types"
	versioncompare "github.com/hashicorp/go-version"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/registry"
	buildkitClient "github.com/moby/buildkit/client"

	// golint requires comments on non-main(test)
	// package for blank import
	_ "github.com/moby/buildkit/client/connhelper/dockercontainer"
	_ "github.com/moby/buildkit/client/connhelper/ssh"
	"github.com/moby/buildkit/frontend/dockerfile/builder"
	"github.com/moby/buildkit/session/upload/uploadprovider"
	log "github.com/sirupsen/logrus"
)

const (
	registryServer        = "https://index.docker.io/v1/"
	buildkitBuilderName   = "linuxkit-builder"
	buildkitSocketPath    = "/run/buildkit/buildkitd.sock"
	buildkitWaitServer    = 30 // seconds
	buildkitCheckInterval = 1  // seconds
)

type dockerRunner interface {
	tag(ref, tag string) error
	build(ctx context.Context, tag, pkg, dockerContext, builderImage, platform string, restart bool, stdin io.Reader, stdout io.Writer, imageBuildOpts types.ImageBuildOptions) error
	save(tgt string, refs ...string) error
	load(src io.Reader) error
	pull(img string) (bool, error)
	contextSupportCheck() error
}

type dockerRunnerImpl struct {
	cache bool
}

type buildContext interface {
	// Copy copies the build context to the supplied WriterCloser
	Copy(io.WriteCloser) error
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
	return dr.command(nil, ioutil.Discard, ioutil.Discard, "context", "ls")
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
		if err := dr.command(nil, ioutil.Discard, ioutil.Discard, "context", "inspect", dockerContext); err != nil {
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
	if err := dr.command(nil, ioutil.Discard, ioutil.Discard, "context", "inspect", dockerContext); err == nil {
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
	var b bytes.Buffer
	if err := dr.command(nil, &b, ioutil.Discard, "--context", dockerContext, "container", "inspect", name); err == nil {
		// we already have a container named "linuxkit-builder" in the provided context.
		var restart bool
		// get its state and config
		var containerJSON []types.ContainerJSON
		if err := json.Unmarshal(b.Bytes(), &containerJSON); err != nil || len(containerJSON) < 1 {
			return nil, fmt.Errorf("unable to read results of 'container inspect %s': %v", name, err)
		}

		existingImage := containerJSON[0].Config.Image

		switch {
		case forceRestart:
			// if restart==true, we always restart, else we check if it matches our requirements
			fmt.Printf("told to force restart, replacing existing container %s\n", name)
			restart = true
		case existingImage != image:
			// if image mismatches, restart
			fmt.Printf("existing container %s is running image %s instead of target %s, replacing\n", name, existingImage, image)
			restart = true
		case !containerJSON[0].HostConfig.Privileged:
			fmt.Printf("existing container %s is unprivileged, replacing\n", name)
			restart = true
		}
		if !restart {
			fmt.Printf("using existing container %s\n", name)
			return buildkitClient.New(ctx, fmt.Sprintf("docker-container://%s?context=%s", name, dockerContext))
		}

		// if we made it here, we need to stop and remove the container, either because of a config mismatch,
		// or because we received the CLI option
		if containerJSON[0].State.Status == "running" {
			if err := dr.command(nil, ioutil.Discard, ioutil.Discard, "--context", dockerContext, "container", "stop", name); err != nil {
				return nil, fmt.Errorf("failed to stop existing container %s", name)
			}
		}
		if err := dr.command(nil, ioutil.Discard, ioutil.Discard, "--context", dockerContext, "container", "rm", name); err != nil {
			return nil, fmt.Errorf("failed to remove existing container %s", name)
		}
	}
	// create the builder
	args := []string{"container", "run", "-d", "--name", name, "--privileged", image, "--allow-insecure-entitlement", "network.host", "--addr", fmt.Sprintf("unix://%s", buildkitSocketPath), "--debug"}
	msg := fmt.Sprintf("creating builder container '%s' in context '%s'", name, dockerContext)
	fmt.Println(msg)
	if err := dr.command(nil, ioutil.Discard, ioutil.Discard, args...); err != nil {
		return nil, err
	}
	// wait for buildkit socket to be ready up to the timeout
	fmt.Printf("waiting for buildkit builder to be ready, up to %d seconds\n", buildkitWaitServer)
	timeout := time.After(buildkitWaitServer * time.Second)
	ticker := time.Tick(buildkitCheckInterval * time.Second)
	// Keep trying until we're timed out or get a success
	for {
		select {
		// Got a timeout! fail with a timeout error
		case <-timeout:
			return nil, fmt.Errorf("could not communicate with buildkit builder at context/container %s/%s after %d seconds", dockerContext, name, buildkitWaitServer)
			// Got a tick, we should try again
		case <-ticker:
			client, err := buildkitClient.New(ctx, fmt.Sprintf("docker-container://%s?context=%s", name, dockerContext))
			if err == nil {
				fmt.Println("buildkit builder ready!")
				return client, nil
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

func (dr dockerRunnerImpl) push(img string) error {
	return dr.command(nil, nil, nil, "image", "push", img)
}

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

	auth, err := registry.GetDockerAuth()
	if err != nil {
		return fmt.Errorf("failed to get auth: %v", err)
	}

	if pushManifest {
		fmt.Printf("Pushing %s to manifest %s\n", img+suffix, img)
		_, _, err = registry.PushManifest(img, auth)
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

func (dr *dockerRunnerImpl) build(ctx context.Context, tag, pkg, dockerContext, builderImage, platform string, restart bool, stdin io.Reader, stdout io.Writer, imageBuildOpts types.ImageBuildOptions) error {
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
	frontendAttrs["network"] = imageBuildOpts.NetworkMode

	for k, v := range imageBuildOpts.Labels {
		frontendAttrs[fmt.Sprintf("label:%s", k)] = v
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
