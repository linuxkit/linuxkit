package pkglib

// Thin wrappers around Docker CLI invocations

//go:generate ./gen

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	versioncompare "github.com/hashicorp/go-version"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/registry"
	log "github.com/sirupsen/logrus"
)

const (
	registryServer      = "https://index.docker.io/v1/"
	buildkitBuilderName = "linuxkit"
)

type dockerRunner interface {
	buildkitCheck() error
	tag(ref, tag string) error
	build(tag, pkg, dockerContext, platform string, stdin io.Reader, stdout io.Writer, opts ...string) error
	save(tgt string, refs ...string) error
	load(src io.Reader) error
	pull(img string) (bool, error)
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
		stderr = os.Stderr
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

// buildkitCheck checks if buildkit is supported. This is necessary because github uses some strange versions
// of docker in Actions, which makes it difficult to tell if buildkit is supported.
// See https://github.community/t/what-really-is-docker-3-0-6/16171
func (dr *dockerRunnerImpl) buildkitCheck() error {
	return dr.command(nil, ioutil.Discard, ioutil.Discard, "buildx", "ls")
}

// builder ensure that a builder exists. Works as follows.
// 1. if dockerContext is provided, try to create a builder with that context; if it succeeds, we are done; if not, return an error.
// 2. try to find an existing named runner with the pattern; if it succeeds, we are done; if not, try next.
// 3. try to create a generic builder using the default context named "linuxkit".
func (dr *dockerRunnerImpl) builder(dockerContext, platform string) (string, error) {
	var (
		builderName string
		args        = []string{"buildx", "create", "--driver", "docker-container", "--buildkitd-flags", "--allow-insecure-entitlement network.host"}
	)

	// if we were given a context, we must find a builder and use it, or create one and use it
	if dockerContext != "" {
		// does the context exist?
		if err := dr.command(nil, ioutil.Discard, ioutil.Discard, "context", "inspect", dockerContext); err != nil {
			return "", fmt.Errorf("provided docker context '%s' not found", dockerContext)
		}
		builderName = fmt.Sprintf("%s-%s-%s-builder", buildkitBuilderName, dockerContext, strings.ReplaceAll(platform, "/", "-"))
		if err := dr.builderEnsureContainer(builderName, platform, dockerContext, args...); err != nil {
			return "", fmt.Errorf("error preparing builder based on context '%s': %v", dockerContext, err)
		}
		return builderName, nil
	}

	// no provided dockerContext, so look for one based on platform-specific name
	dockerContext = fmt.Sprintf("%s-%s", buildkitBuilderName, strings.ReplaceAll(platform, "/", "-"))
	if err := dr.command(nil, ioutil.Discard, ioutil.Discard, "context", "inspect", dockerContext); err == nil {
		// we found an appropriately named context, so let us try to use it or error out
		builderName = fmt.Sprintf("%s-builder", dockerContext)
		if err := dr.builderEnsureContainer(builderName, platform, dockerContext, args...); err == nil {
			return builderName, nil
		}
	}

	// create a generic builder
	builderName = buildkitBuilderName
	if err := dr.builderEnsureContainer(builderName, "", "", args...); err != nil {
		return "", fmt.Errorf("error ensuring default builder '%s': %v", builderName, err)
	}
	return builderName, nil
}

// builderEnsureContainer provided a name of a builder, ensure that the builder exists, and if not, create it
// based on the provided docker context, for the target platform.. Assumes the dockerContext already exists.
func (dr *dockerRunnerImpl) builderEnsureContainer(name, platform, dockerContext string, args ...string) error {
	// if no error, then we have a builder already
	// inspect it to make sure it is of the right type
	var b bytes.Buffer
	if err := dr.command(nil, &b, ioutil.Discard, "buildx", "inspect", name); err != nil {
		// we did not have the named builder, so create the builder
		args = append(args, "--name", name)
		msg := fmt.Sprintf("creating builder '%s'", name)
		if platform != "" {
			args = append(args, "--platform", platform)
			msg = fmt.Sprintf("%s for platform '%s'", msg, platform)
		} else {
			msg = fmt.Sprintf("%s for all supported platforms", msg)
		}
		if dockerContext != "" {
			args = append(args, dockerContext)
			msg = fmt.Sprintf("%s based on docker context '%s'", msg, dockerContext)
		}
		fmt.Println(msg)
		return dr.command(nil, ioutil.Discard, ioutil.Discard, args...)
	}
	// if we got here, we found a builder already, so let us check its type
	var (
		scanner = bufio.NewScanner(&b)
		driver  string
	)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		if fields[0] != "Driver:" {
			continue
		}
		driver = fields[1]
		break
	}

	switch driver {
	case "":
		return fmt.Errorf("builder '%s' exists but has no driver type", name)
	case "docker-container":
		return nil
	default:
		return fmt.Errorf("builder '%s' exists but has wrong driver type '%s'", name, driver)
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

func (dr *dockerRunnerImpl) build(tag, pkg, dockerContext, platform string, stdin io.Reader, stdout io.Writer, opts ...string) error {
	// ensure we have a builder
	builderName, err := dr.builder(dockerContext, platform)
	if err != nil {
		return fmt.Errorf("unable to ensure proper buildx builder: %v", err)
	}

	args := []string{"buildx", "build"}

	for _, proxyVarName := range proxyEnvVars {
		if value, ok := os.LookupEnv(proxyVarName); ok {
			args = append(args,
				[]string{"--build-arg", fmt.Sprintf("%s=%s", proxyVarName, value)}...)
		}
	}
	if !dr.cache {
		args = append(args, "--no-cache")
	}
	args = append(args, opts...)
	args = append(args, fmt.Sprintf("--builder=%s", builderName))
	args = append(args, "-t", tag)

	// should docker read from the build path or stdin?
	buildPath := pkg
	if stdin != nil {
		buildPath = "-"
	}
	args = append(args, buildPath)

	fmt.Printf("building for platform %s using builder %s\n", platform, builderName)
	return dr.command(stdin, stdout, nil, args...)
}

func (dr *dockerRunnerImpl) save(tgt string, refs ...string) error {
	args := append([]string{"image", "save", "-o", tgt}, refs...)
	return dr.command(nil, nil, nil, args...)
}

func (dr *dockerRunnerImpl) load(src io.Reader) error {
	args := []string{"image", "load"}
	return dr.command(src, nil, nil, args...)
}
