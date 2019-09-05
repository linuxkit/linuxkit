package pkglib

// Thin wrappers around Docker CLI invocations

//go:generate ./gen

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const dctEnableEnv = "DOCKER_CONTENT_TRUST=1"

type dockerRunner struct {
	dct   bool
	cache bool

	// Optional build context to use
	ctx buildContext
}

type buildContext interface {
	// Copy copies the build context to the supplied WriterCloser
	Copy(io.WriteCloser) error
}

func newDockerRunner(dct, cache bool) dockerRunner {
	return dockerRunner{dct: dct, cache: cache}
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

func (dr dockerRunner) command(args ...string) error {
	cmd := exec.Command("docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	dct := ""
	if dr.dct {
		cmd.Env = append(cmd.Env, dctEnableEnv)
		dct = dctEnableEnv + " "
	}

	var eg errgroup.Group

	if args[0] == "build" {
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

	log.Debugf("Executing: %s%v", dct, cmd.Args)

	if err := cmd.Run(); err != nil {
		if isExecErrNotFound(err) {
			return fmt.Errorf("linuxkit pkg requires docker to be installed")
		}
		return err
	}
	return eg.Wait()
}

func (dr dockerRunner) pull(img string) (bool, error) {
	err := dr.command("image", "pull", img)
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
	return dr.command("image", "push", img)
}

func (dr dockerRunner) pushWithManifest(img, suffix string) error {
	fmt.Printf("Pushing %s\n", img+suffix)
	if err := dr.push(img + suffix); err != nil {
		return err
	}

	var dctArg string
	if dr.dct {
		dctArg = "1"
	}

	fmt.Printf("Pushing %s to manifest %s\n", img+suffix, img)
	cmd := exec.Command("/bin/sh", "-c", manifestPushScript, "manifest-push-script", img, dctArg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Debugf("Executing: %v", cmd.Args)

	return cmd.Run()
}

func (dr dockerRunner) tag(ref, tag string) error {
	fmt.Printf("Tagging %s as %s\n", ref, tag)
	return dr.command("image", "tag", ref, tag)
}

func (dr dockerRunner) build(tag, pkg string, opts ...string) error {
	args := []string{"build"}
	if !dr.cache {
		args = append(args, "--no-cache")
	}
	args = append(args, opts...)
	args = append(args, "-t", tag, pkg)
	return dr.command(args...)
}

func (dr dockerRunner) save(tgt string, refs ...string) error {
	args := append([]string{"image", "save", "-o", tgt}, refs...)
	return dr.command(args...)
}
