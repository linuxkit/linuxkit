// Package dockercontext is the single source of Docker context selection for
// LinuxKit. Resolve a selection once, then use it for every operation on that daemon.
package dockercontext

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/containerd/errdefs"
	"github.com/docker/cli/cli/command"
	"github.com/docker/cli/cli/config"
	dockerendpoint "github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/moby/moby/client"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

// Selection identifies a Docker context and the configuration directory containing it.
// The special context "default" still uses DOCKER_HOST and TLS environment settings.
type Selection struct {
	Name      string
	Source    string
	ConfigDir string
}

func configDir() (string, error) {
	if dir := os.Getenv("DOCKER_CONFIG"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find Docker configuration directory: %w", err)
	}
	return filepath.Join(home, ".docker"), nil
}

// Resolve applies --context, DOCKER_CONTEXT, DOCKER_HOST, desktop-linux, default
// precedence. The saved currentContext is never used. An explicit context named
// "default" is preserved as such.
func Resolve(explicit string) (Selection, error) {
	return ResolveBuilder("", explicit, "")
}

// ResolveBuilder first honors a per-platform mapping, then the general explicit
// and environment overrides. Only without overrides is linuxkit-<platform>
// discovered before desktop-linux and finally the literal default context.
func ResolveBuilder(mapped, explicit, platform string) (Selection, error) {
	dir, err := configDir()
	if err != nil {
		return Selection{}, err
	}
	s := Selection{ConfigDir: dir}
	switch {
	case mapped != "":
		s.Name, s.Source = mapped, "platform builder mapping"
	case explicit != "":
		s.Name, s.Source = explicit, "--context"
	case os.Getenv("DOCKER_CONTEXT") != "":
		s.Name, s.Source = os.Getenv("DOCKER_CONTEXT"), "DOCKER_CONTEXT"
	case os.Getenv("DOCKER_HOST") != "":
		s.Name, s.Source = "default", "DOCKER_HOST"
	default:
		candidates := []string{}
		if platform != "" {
			candidates = append(candidates, "linuxkit-"+strings.ReplaceAll(platform, "/", "-"))
		}
		candidates = append(candidates, "desktop-linux")
		st := store.New(filepath.Join(dir, "contexts"), command.DefaultContextStoreConfig())
		for _, name := range candidates {
			if _, err := st.GetMetadata(name); err == nil {
				s.Name, s.Source = name, "automatic context discovery"
				break
			} else if !errdefs.IsNotFound(err) {
				return Selection{}, fmt.Errorf("inspect Docker context %q: %w", name, err)
			}
		}
		if s.Name == "" {
			s.Name, s.Source = "default", "Docker platform default"
		}
	}
	log.Debugf("Docker context %q selected by %s (config %s)", s.Name, s.Source, s.ConfigDir)
	return s, nil
}

// Args makes the context and configuration explicit on every Docker CLI invocation.
func (s Selection) Args(args ...string) []string {
	return append([]string{"--config", s.ConfigDir, "--context", s.Name}, args...)
}

// CommandError retains the command's original error and diagnostic output.
type CommandError struct {
	Context string
	Args    []string
	Stderr  string
	Err     error
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("docker context %q: docker %s: %v\n%s", e.Context, strings.Join(e.Args, " "), e.Err, strings.TrimSpace(e.Stderr))
}
func (e *CommandError) Unwrap() error { return e.Err }

// Run always captures stderr, even when the caller suppresses or redirects it.
func (s Selection) Run(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, args ...string) error {
	argv := s.Args(args...)
	cmd := exec.CommandContext(ctx, "docker", argv...)
	cmd.Stdin, cmd.Stdout = stdin, stdout
	var diagnostics bytes.Buffer
	cmd.Stderr = &diagnostics
	if stderr != nil {
		cmd.Stderr = io.MultiWriter(stderr, &diagnostics)
	}
	log.Debugf("Executing Docker command in context %q: %v", s.Name, argv)
	if err := cmd.Run(); err != nil {
		return &CommandError{Context: s.Name, Args: argv, Stderr: diagnostics.String(), Err: err}
	}
	return nil
}

// BuilderURL pins BuildKit's docker exec connection to the same context.
func (s Selection) BuilderURL(container string) string {
	return (&url.URL{Scheme: "docker-container", Host: container, RawQuery: url.Values{"context": {s.Name}}.Encode()}).String()
}

// Client uses Docker's endpoint resolver, including context TLS and SSH settings.
// Passing the already selected name explicitly also enforces DOCKER_CONTEXT over
// DOCKER_HOST, independent of the vendored CLI's ambient selection behavior.
func (s Selection) Client() (client.APIClient, error) {
	cfg, err := config.Load(s.ConfigDir)
	if err != nil {
		return nil, fmt.Errorf("docker context %q: load configuration: %w", s.Name, err)
	}
	if s.Name != "default" {
		st := store.New(filepath.Join(s.ConfigDir, "contexts"), command.DefaultContextStoreConfig())
		metadata, err := st.GetMetadata(s.Name)
		if err != nil {
			return nil, fmt.Errorf("docker context %q: %w", s.Name, err)
		}
		endpoint, err := dockerendpoint.EndpointFromContext(metadata)
		if err != nil {
			return nil, fmt.Errorf("docker context %q: %w", s.Name, err)
		}
		ep, err := dockerendpoint.WithTLSData(st, s.Name, endpoint)
		if err != nil {
			return nil, fmt.Errorf("docker context %q: load TLS settings: %w", s.Name, err)
		}
		opts, err := ep.ClientOpts()
		if err != nil {
			return nil, fmt.Errorf("docker context %q: configure endpoint: %w", s.Name, err)
		}
		opts = append(opts, client.WithHTTPHeaders(cfg.HTTPHeaders))
		cli, err := client.New(opts...)
		if err != nil {
			return nil, fmt.Errorf("initialize Docker context %q: %w", s.Name, err)
		}
		return cli, nil
	}
	opts := cliflags.NewClientOptions()
	flags := pflag.NewFlagSet("docker", pflag.ContinueOnError)
	opts.InstallFlags(flags)
	opts.Context = s.Name
	opts.TLS = os.Getenv("DOCKER_TLS") != ""
	opts.TLSVerify = os.Getenv("DOCKER_TLS_VERIFY") != ""
	certDir := os.Getenv("DOCKER_CERT_PATH")
	if certDir == "" {
		certDir = s.ConfigDir
	}
	opts.TLSOptions.CAFile = filepath.Join(certDir, "ca.pem")
	opts.TLSOptions.CertFile = filepath.Join(certDir, "cert.pem")
	opts.TLSOptions.KeyFile = filepath.Join(certDir, "key.pem")
	opts.SetDefaultOptions(flags)
	cli, err := command.NewAPIClientFromFlags(opts, cfg)
	if err != nil {
		return nil, fmt.Errorf("initialize Docker context %q (selected by %s): %w", s.Name, s.Source, err)
	}
	return cli, nil
}
