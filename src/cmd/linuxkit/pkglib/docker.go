package pkglib

// Thin wrappers around Docker CLI invocations

//go:generate ./gen

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	versioncompare "github.com/hashicorp/go-version"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/registry"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	registryServer      = "https://index.docker.io/v1/"
	buildkitBuilderName = "linuxkit"
)

var platforms = []string{
	"linux/amd64", "linux/arm64", "linux/s390x",
}

type dockerRunner struct {
	cache bool

	// Optional build context to use
	ctx buildContext
}

type buildContext interface {
	// Copy copies the build context to the supplied WriterCloser
	Copy(io.WriteCloser) error
}

func newDockerRunner(cache bool) dockerRunner {
	return dockerRunner{cache: cache}
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

func (dr dockerRunner) command(stdout, stderr io.Writer, args ...string) error {
	cmd := exec.Command("docker", args...)
	if stdout == nil {
		stdout = os.Stdout
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = os.Environ()

	var eg errgroup.Group

	// special handling for build-args
	if args[0] == "buildx" && args[1] == "build" {
		buildArgs := []string{}
		for _, proxyVarName := range proxyEnvVars {
			if value, ok := os.LookupEnv(proxyVarName); ok {
				buildArgs = append(buildArgs,
					[]string{"--build-arg", fmt.Sprintf("%s=%s", proxyVarName, value)}...)
			}
		}
		// cannot use usual append(append( because it overwrites part of it
		newArgs := make([]string, len(cmd.Args)+len(buildArgs))
		copy(newArgs[:2], cmd.Args[:2])
		copy(newArgs[2:], buildArgs)
		copy(newArgs[2+len(buildArgs):], cmd.Args[2:])
		cmd.Args = newArgs

		if dr.ctx != nil {
			stdin, err := cmd.StdinPipe()
			if err != nil {
				return err
			}
			eg.Go(func() error {
				defer stdin.Close()
				return dr.ctx.Copy(stdin)
			})

			cmd.Args = append(cmd.Args[:len(cmd.Args)-1], "-")
		}
	}

	log.Debugf("Executing: %v", cmd.Args)

	if err := cmd.Run(); err != nil {
		if isExecErrNotFound(err) {
			return fmt.Errorf("linuxkit pkg requires docker to be installed")
		}
		return err
	}
	return eg.Wait()
}

// versionCheck returns the client version and server version, and compares them both
// against the minimum required version.
func (dr dockerRunner) versionCheck(version string) (string, string, error) {
	var stdout bytes.Buffer
	if err := dr.command(&stdout, nil, "version", "--format", "json"); err != nil {
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
func (dr dockerRunner) buildkitCheck() error {
	return dr.command(nil, nil, "buildx", "ls")
}

// builder ensure that a builder of the given name exists
func (dr dockerRunner) builder(name string) error {
	if err := dr.command(nil, nil, "buildx", "inspect", name); err == nil {
		// if no error, then we have a builder already
		return nil
	}

	// create a builder
	return dr.command(nil, nil, "buildx", "create", "--name", name, "--driver", "docker-container", "--buildkitd-flags", "--allow-insecure-entitlement network.host")
}

func (dr dockerRunner) pull(img string) (bool, error) {
	err := dr.command(nil, nil, "image", "pull", img)
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

func (dr dockerRunner) push(img string) error {
	return dr.command(nil, nil, "image", "push", img)
}

func (dr dockerRunner) pushWithManifest(img, suffix string, pushImage, pushManifest bool) error {
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

func (dr dockerRunner) tag(ref, tag string) error {
	fmt.Printf("Tagging %s as %s\n", ref, tag)
	return dr.command(nil, nil, "image", "tag", ref, tag)
}

func (dr dockerRunner) build(tag, pkg string, stdout io.Writer, opts ...string) error {
	// ensure we have a builder
	if err := dr.builder(buildkitBuilderName); err != nil {
		return fmt.Errorf("unable to ensure proper buildx builder: %v", err)
	}

	args := []string{"buildx", "build"}
	if !dr.cache {
		args = append(args, "--no-cache")
	}
	args = append(args, opts...)
	args = append(args, fmt.Sprintf("--builder=%s", buildkitBuilderName))
	args = append(args, "-t", tag, pkg)
	return dr.command(stdout, nil, args...)
}

func (dr dockerRunner) save(tgt string, refs ...string) error {
	args := append([]string{"image", "save", "-o", tgt}, refs...)
	return dr.command(nil, nil, args...)
}
