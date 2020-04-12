package pkglib

// Thin wrappers around Docker CLI invocations

//go:generate ./gen

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/docker/cli/cli/config"
	"github.com/docker/distribution/manifest/manifestlist"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/estesp/manifest-tool/docker"
	"github.com/estesp/manifest-tool/types"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

const (
	dctEnableEnv                     = "DOCKER_CONTENT_TRUST=1"
	registry                         = "https://index.docker.io/v1/"
	notaryServer                     = "https://notary.docker.io"
	notaryDelegationPassphraseEnvVar = "NOTARY_DELEGATION_PASSPHRASE"
	notaryAuthEnvVar                 = "NOTARY_AUTH"
	dctEnvVar                        = "DOCKER_CONTENT_TRUST_REPOSITORY_PASSPHRASE"
)

var platforms = []string{
	"linux/amd64", "linux/arm64", "linux/s390x",
}

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

	auth, err := getDockerAuth()
	if err != nil {
		return fmt.Errorf("failed to get auth: %v", err)
	}

	fmt.Printf("Pushing %s to manifest %s\n", img+suffix, img)
	digest, l, err := manifestPush(img, auth)
	if err != nil {
		return err
	}
	// if trust is not enabled, nothing more to do
	if !dr.dct {
		fmt.Println("trust disabled, not signing")
		return nil
	}
	fmt.Printf("Signing manifest for %s\n", img)
	return signManifest(img, digest, l, auth)
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

func getDockerAuth() (dockertypes.AuthConfig, error) {
	cfgFile := config.LoadDefaultConfigFile(os.Stderr)
	return cfgFile.GetAuthConfig(registry)
}

func manifestPush(img string, auth dockertypes.AuthConfig) (hash string, length int, err error) {
	srcImages := []types.ManifestEntry{}

	for i, platform := range platforms {
		osArchArr := strings.Split(platform, "/")
		if len(osArchArr) != 2 && len(osArchArr) != 3 {
			return hash, length, fmt.Errorf("platform argument %d is not of form 'os/arch': '%s'", i, platform)
		}
		variant := ""
		os, arch := osArchArr[0], osArchArr[1]
		if len(osArchArr) == 3 {
			variant = osArchArr[2]
		}
		srcImages = append(srcImages, types.ManifestEntry{
			Image: fmt.Sprintf("%s-%s", img, arch),
			Platform: manifestlist.PlatformSpec{
				OS:           os,
				Architecture: arch,
				Variant:      variant,
			},
		})
	}

	yamlInput := types.YAMLInput{
		Image:     img,
		Manifests: srcImages,
	}

	a := types.AuthInfo{
		Username: auth.Username,
		Password: auth.Password,
	}

	// push the manifest list with the auth as given, ignore missing, do not allow insecure
	return docker.PutManifestList(&a, yamlInput, true, false)
}

func signManifest(img, digest string, length int, auth dockertypes.AuthConfig) error {
	imgParts := strings.Split(img, ":")
	if len(imgParts) < 2 {
		return fmt.Errorf("image not composed of <repo>:<tag> '%s'", img)
	}
	repo := imgParts[0]
	tag := imgParts[1]

	digestParts := strings.Split(digest, ":")
	if len(digestParts) < 2 {
		return fmt.Errorf("digest not composed of <algo>:<hash> '%s'", digest)
	}
	algo, hash := digestParts[0], digestParts[1]
	if algo != "sha256" {
		return fmt.Errorf("notary works with sha256 hash, not the provided %s", algo)
	}

	notaryAuth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", auth.Username, auth.Password)))
	// run the notary command to sign
	args := []string{
		"-s",
		notaryServer,
		"-d",
		path.Join(os.Getenv("HOME"), ".docker/trust"),
		"addhash",
		"-p",
		fmt.Sprintf("docker.io/%s", repo),
		tag,
		strconv.Itoa(length),
		"--sha256",
		hash,
		"-r",
		"targets/releases",
	}
	cmd := exec.Command("notary", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", notaryDelegationPassphraseEnvVar, os.Getenv(dctEnvVar)), fmt.Sprintf("%s=%s", notaryAuthEnvVar, notaryAuth))
	log.Debugf("Executing: %v", cmd.Args)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute notary-tool: %v", err)
	}

	// report output
	fmt.Printf("New signed multi-arch image: %s:%s\n", repo, tag)

	return nil
}
