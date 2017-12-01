package pkglib

// Thin wrappers around Docker CLI invocations

//go:generate ./gen

import (
	"fmt"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

const dctEnableEnv = "DOCKER_CONTENT_TRUST=1"

type dockerRunner struct {
	dct   bool
	cache bool
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

	log.Debugf("Executing: %s%v", dct, cmd.Args)

	err := cmd.Run()
	if isExecErrNotFound(err) {
		return fmt.Errorf("linuxkit pkg requires docker to be installed")
	}
	return err
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
