// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.23

package command

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/docker/cli/cli/config"
	"github.com/docker/cli/cli/config/configfile"
	dcontext "github.com/docker/cli/cli/context"
	"github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
	"github.com/docker/cli/cli/debug"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/docker/cli/cli/streams"
	"github.com/docker/cli/cli/version"
	dopts "github.com/docker/cli/opts"
	"github.com/docker/docker/api"
	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const defaultInitTimeout = 2 * time.Second

// Streams is an interface which exposes the standard input and output streams
type Streams interface {
	In() *streams.In
	Out() *streams.Out
	Err() *streams.Out
}

// Cli represents the docker command line client.
type Cli interface {
	Client() client.APIClient
	Streams
	SetIn(in *streams.In)
	Apply(ops ...CLIOption) error
	config.Provider
	ServerInfo() ServerInfo
	DefaultVersion() string
	CurrentVersion() string
	ContentTrustEnabled() bool
	BuildKitEnabled() (bool, error)
	ContextStore() store.Store
	CurrentContext() string
	DockerEndpoint() docker.Endpoint
	TelemetryClient
}

// DockerCli is an instance the docker command line client.
// Instances of the client should be created using the [NewDockerCli]
// constructor to make sure they are properly initialized with defaults
// set.
type DockerCli struct {
	configFile         *configfile.ConfigFile
	options            *cliflags.ClientOptions
	in                 *streams.In
	out                *streams.Out
	err                *streams.Out
	client             client.APIClient
	serverInfo         ServerInfo
	contentTrust       bool
	contextStore       store.Store
	currentContext     string
	init               sync.Once
	initErr            error
	dockerEndpoint     docker.Endpoint
	contextStoreConfig *store.Config
	initTimeout        time.Duration
	res                telemetryResource

	// baseCtx is the base context used for internal operations. In the future
	// this may be replaced by explicitly passing a context to functions that
	// need it.
	baseCtx context.Context

	enableGlobalMeter, enableGlobalTracer bool
}

// DefaultVersion returns [api.DefaultVersion].
func (*DockerCli) DefaultVersion() string {
	return api.DefaultVersion
}

// CurrentVersion returns the API version currently negotiated, or the default
// version otherwise.
func (cli *DockerCli) CurrentVersion() string {
	_ = cli.initialize()
	if cli.client == nil {
		return api.DefaultVersion
	}
	return cli.client.ClientVersion()
}

// Client returns the APIClient
func (cli *DockerCli) Client() client.APIClient {
	if err := cli.initialize(); err != nil {
		_, _ = fmt.Fprintln(cli.Err(), "Failed to initialize:", err)
		os.Exit(1)
	}
	return cli.client
}

// Out returns the writer used for stdout
func (cli *DockerCli) Out() *streams.Out {
	return cli.out
}

// Err returns the writer used for stderr
func (cli *DockerCli) Err() *streams.Out {
	return cli.err
}

// SetIn sets the reader used for stdin
func (cli *DockerCli) SetIn(in *streams.In) {
	cli.in = in
}

// In returns the reader used for stdin
func (cli *DockerCli) In() *streams.In {
	return cli.in
}

// ShowHelp shows the command help.
func ShowHelp(err io.Writer) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		cmd.SetOut(err)
		cmd.HelpFunc()(cmd, args)
		return nil
	}
}

// ConfigFile returns the ConfigFile
func (cli *DockerCli) ConfigFile() *configfile.ConfigFile {
	// TODO(thaJeztah): when would this happen? Is this only in tests (where cli.Initialize() is not called first?)
	if cli.configFile == nil {
		cli.configFile = config.LoadDefaultConfigFile(cli.err)
	}
	return cli.configFile
}

// ServerInfo returns the server version details for the host this client is
// connected to
func (cli *DockerCli) ServerInfo() ServerInfo {
	_ = cli.initialize()
	return cli.serverInfo
}

// ContentTrustEnabled returns whether content trust has been enabled by an
// environment variable.
func (cli *DockerCli) ContentTrustEnabled() bool {
	return cli.contentTrust
}

// BuildKitEnabled returns buildkit is enabled or not.
func (cli *DockerCli) BuildKitEnabled() (bool, error) {
	// use DOCKER_BUILDKIT env var value if set and not empty
	if v := os.Getenv("DOCKER_BUILDKIT"); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return false, errors.Wrap(err, "DOCKER_BUILDKIT environment variable expects boolean value")
		}
		return enabled, nil
	}
	// if a builder alias is defined, we are using BuildKit
	aliasMap := cli.ConfigFile().Aliases
	if _, ok := aliasMap["builder"]; ok {
		return true, nil
	}

	si := cli.ServerInfo()
	if si.BuildkitVersion == build.BuilderBuildKit {
		// The daemon advertised BuildKit as the preferred builder; this may
		// be either a Linux daemon or a Windows daemon with experimental
		// BuildKit support enabled.
		return true, nil
	}

	// otherwise, assume BuildKit is enabled for Linux, but disabled for
	// Windows / WCOW, which does not yet support BuildKit by default.
	return si.OSType != "windows", nil
}

// HooksEnabled returns whether plugin hooks are enabled.
func (cli *DockerCli) HooksEnabled() bool {
	// use DOCKER_CLI_HOOKS env var value if set and not empty
	if v := os.Getenv("DOCKER_CLI_HOOKS"); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return false
		}
		return enabled
	}
	// legacy support DOCKER_CLI_HINTS env var
	if v := os.Getenv("DOCKER_CLI_HINTS"); v != "" {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return false
		}
		return enabled
	}
	featuresMap := cli.ConfigFile().Features
	if v, ok := featuresMap["hooks"]; ok {
		enabled, err := strconv.ParseBool(v)
		if err != nil {
			return false
		}
		return enabled
	}
	// default to false
	return false
}

// Initialize the dockerCli runs initialization that must happen after command
// line flags are parsed.
func (cli *DockerCli) Initialize(opts *cliflags.ClientOptions, ops ...CLIOption) error {
	for _, o := range ops {
		if err := o(cli); err != nil {
			return err
		}
	}
	cliflags.SetLogLevel(opts.LogLevel)

	if opts.ConfigDir != "" {
		config.SetDir(opts.ConfigDir)
	}

	if opts.Debug {
		debug.Enable()
	}
	if opts.Context != "" && len(opts.Hosts) > 0 {
		return errors.New("conflicting options: cannot specify both --host and --context")
	}

	if cli.contextStoreConfig == nil {
		// This path can be hit when calling Initialize on a DockerCli that's
		// not constructed through [NewDockerCli]. Using the default context
		// store without a config set will result in Endpoints from contexts
		// not being type-mapped correctly, and used as a generic "map[string]any",
		// instead of a [docker.EndpointMeta].
		//
		// When looking up the API endpoint (using [EndpointFromContext]), no
		// endpoint will be found, and a default, empty endpoint will be used
		// instead which in its turn, causes newAPIClientFromEndpoint to
		// be initialized with the default config instead of settings for
		// the current context (which may mean; connecting with the wrong
		// endpoint and/or TLS Config to be missing).
		//
		// [EndpointFromContext]: https://github.com/docker/cli/blob/33494921b80fd0b5a06acc3a34fa288de4bb2e6b/cli/context/docker/load.go#L139-L149
		if err := WithDefaultContextStoreConfig()(cli); err != nil {
			return err
		}
	}

	cli.options = opts
	cli.configFile = config.LoadDefaultConfigFile(cli.err)
	cli.currentContext = resolveContextName(cli.options, cli.configFile)
	cli.contextStore = &ContextStoreWithDefault{
		Store: store.New(config.ContextStoreDir(), *cli.contextStoreConfig),
		Resolver: func() (*DefaultContext, error) {
			return ResolveDefaultContext(cli.options, *cli.contextStoreConfig)
		},
	}

	// TODO(krissetto): pass ctx to the funcs instead of using this
	if cli.enableGlobalMeter {
		cli.createGlobalMeterProvider(cli.baseCtx)
	}
	if cli.enableGlobalTracer {
		cli.createGlobalTracerProvider(cli.baseCtx)
	}
	filterResourceAttributesEnvvar()

	return nil
}

// NewAPIClientFromFlags creates a new APIClient from command line flags
func NewAPIClientFromFlags(opts *cliflags.ClientOptions, configFile *configfile.ConfigFile) (client.APIClient, error) {
	if opts.Context != "" && len(opts.Hosts) > 0 {
		return nil, errors.New("conflicting options: cannot specify both --host and --context")
	}

	storeConfig := DefaultContextStoreConfig()
	contextStore := &ContextStoreWithDefault{
		Store: store.New(config.ContextStoreDir(), storeConfig),
		Resolver: func() (*DefaultContext, error) {
			return ResolveDefaultContext(opts, storeConfig)
		},
	}
	endpoint, err := resolveDockerEndpoint(contextStore, resolveContextName(opts, configFile))
	if err != nil {
		return nil, errors.Wrap(err, "unable to resolve docker endpoint")
	}
	return newAPIClientFromEndpoint(endpoint, configFile)
}

func newAPIClientFromEndpoint(ep docker.Endpoint, configFile *configfile.ConfigFile) (client.APIClient, error) {
	opts, err := ep.ClientOpts()
	if err != nil {
		return nil, err
	}
	if len(configFile.HTTPHeaders) > 0 {
		opts = append(opts, client.WithHTTPHeaders(configFile.HTTPHeaders))
	}
	opts = append(opts, withCustomHeadersFromEnv(), client.WithUserAgent(UserAgent()))
	return client.NewClientWithOpts(opts...)
}

func resolveDockerEndpoint(s store.Reader, contextName string) (docker.Endpoint, error) {
	if s == nil {
		return docker.Endpoint{}, errors.New("no context store initialized")
	}
	ctxMeta, err := s.GetMetadata(contextName)
	if err != nil {
		return docker.Endpoint{}, err
	}
	epMeta, err := docker.EndpointFromContext(ctxMeta)
	if err != nil {
		return docker.Endpoint{}, err
	}
	return docker.WithTLSData(s, contextName, epMeta)
}

// Resolve the Docker endpoint for the default context (based on config, env vars and CLI flags)
func resolveDefaultDockerEndpoint(opts *cliflags.ClientOptions) (docker.Endpoint, error) {
	// defaultToTLS determines whether we should use a TLS host as default
	// if nothing was configured by the user.
	defaultToTLS := opts.TLSOptions != nil
	host, err := getServerHost(opts.Hosts, defaultToTLS)
	if err != nil {
		return docker.Endpoint{}, err
	}

	var (
		skipTLSVerify bool
		tlsData       *dcontext.TLSData
	)

	if opts.TLSOptions != nil {
		skipTLSVerify = opts.TLSOptions.InsecureSkipVerify
		tlsData, err = dcontext.TLSDataFromFiles(opts.TLSOptions.CAFile, opts.TLSOptions.CertFile, opts.TLSOptions.KeyFile)
		if err != nil {
			return docker.Endpoint{}, err
		}
	}

	return docker.Endpoint{
		EndpointMeta: docker.EndpointMeta{
			Host:          host,
			SkipTLSVerify: skipTLSVerify,
		},
		TLSData: tlsData,
	}, nil
}

func (cli *DockerCli) getInitTimeout() time.Duration {
	if cli.initTimeout != 0 {
		return cli.initTimeout
	}
	return defaultInitTimeout
}

func (cli *DockerCli) initializeFromClient() {
	ctx, cancel := context.WithTimeout(cli.baseCtx, cli.getInitTimeout())
	defer cancel()

	ping, err := cli.client.Ping(ctx)
	if err != nil {
		// Default to true if we fail to connect to daemon
		cli.serverInfo = ServerInfo{HasExperimental: true}

		if ping.APIVersion != "" {
			cli.client.NegotiateAPIVersionPing(ping)
		}
		return
	}

	cli.serverInfo = ServerInfo{
		HasExperimental: ping.Experimental,
		OSType:          ping.OSType,
		BuildkitVersion: ping.BuilderVersion,
		SwarmStatus:     ping.SwarmStatus,
	}
	cli.client.NegotiateAPIVersionPing(ping)
}

// ContextStore returns the ContextStore
func (cli *DockerCli) ContextStore() store.Store {
	return cli.contextStore
}

// CurrentContext returns the current context name, based on flags,
// environment variables and the cli configuration file, in the following
// order of preference:
//
//  1. The "--context" command-line option.
//  2. The "DOCKER_CONTEXT" environment variable ([EnvOverrideContext]).
//  3. The current context as configured through the in "currentContext"
//     field in the CLI configuration file ("~/.docker/config.json").
//  4. If no context is configured, use the "default" context.
//
// # Fallbacks for backward-compatibility
//
// To preserve backward-compatibility with the "pre-contexts" behavior,
// the "default" context is used if:
//
//   - The "--host" option is set
//   - The "DOCKER_HOST" ([client.EnvOverrideHost]) environment variable is set
//     to a non-empty value.
//
// In these cases, the default context is used, which uses the host as
// specified in "DOCKER_HOST", and TLS config from flags/env vars.
//
// Setting both the "--context" and "--host" flags is ambiguous and results
// in an error when the cli is started.
//
// CurrentContext does not validate if the given context exists or if it's
// valid; errors may occur when trying to use it.
func (cli *DockerCli) CurrentContext() string {
	return cli.currentContext
}

// CurrentContext returns the current context name, based on flags,
// environment variables and the cli configuration file. It does not
// validate if the given context exists or if it's valid; errors may
// occur when trying to use it.
//
// Refer to [DockerCli.CurrentContext] above for further details.
func resolveContextName(opts *cliflags.ClientOptions, cfg *configfile.ConfigFile) string {
	if opts != nil && opts.Context != "" {
		return opts.Context
	}
	if opts != nil && len(opts.Hosts) > 0 {
		return DefaultContextName
	}
	if os.Getenv(client.EnvOverrideHost) != "" {
		return DefaultContextName
	}
	if ctxName := os.Getenv(EnvOverrideContext); ctxName != "" {
		return ctxName
	}
	if cfg != nil && cfg.CurrentContext != "" {
		// We don't validate if this context exists: errors may occur when trying to use it.
		return cfg.CurrentContext
	}
	return DefaultContextName
}

// DockerEndpoint returns the current docker endpoint
func (cli *DockerCli) DockerEndpoint() docker.Endpoint {
	if err := cli.initialize(); err != nil {
		// Note that we're not terminating here, as this function may be used
		// in cases where we're able to continue.
		_, _ = fmt.Fprintln(cli.Err(), cli.initErr)
	}
	return cli.dockerEndpoint
}

func (cli *DockerCli) getDockerEndPoint() (ep docker.Endpoint, err error) {
	cn := cli.CurrentContext()
	if cn == DefaultContextName {
		return resolveDefaultDockerEndpoint(cli.options)
	}
	return resolveDockerEndpoint(cli.contextStore, cn)
}

func (cli *DockerCli) initialize() error {
	cli.init.Do(func() {
		cli.dockerEndpoint, cli.initErr = cli.getDockerEndPoint()
		if cli.initErr != nil {
			cli.initErr = errors.Wrap(cli.initErr, "unable to resolve docker endpoint")
			return
		}
		if cli.client == nil {
			if cli.client, cli.initErr = newAPIClientFromEndpoint(cli.dockerEndpoint, cli.configFile); cli.initErr != nil {
				return
			}
		}
		if cli.baseCtx == nil {
			cli.baseCtx = context.Background()
		}
		cli.initializeFromClient()
	})
	return cli.initErr
}

// Apply all the operation on the cli
func (cli *DockerCli) Apply(ops ...CLIOption) error {
	for _, op := range ops {
		if err := op(cli); err != nil {
			return err
		}
	}
	return nil
}

// ServerInfo stores details about the supported features and platform of the
// server
type ServerInfo struct {
	HasExperimental bool
	OSType          string
	BuildkitVersion build.BuilderVersion

	// SwarmStatus provides information about the current swarm status of the
	// engine, obtained from the "Swarm" header in the API response.
	//
	// It can be a nil struct if the API version does not provide this header
	// in the ping response, or if an error occurred, in which case the client
	// should use other ways to get the current swarm status, such as the /swarm
	// endpoint.
	SwarmStatus *swarm.Status
}

// NewDockerCli returns a DockerCli instance with all operators applied on it.
// It applies by default the standard streams, and the content trust from
// environment.
func NewDockerCli(ops ...CLIOption) (*DockerCli, error) {
	defaultOps := []CLIOption{
		WithContentTrustFromEnv(),
		WithDefaultContextStoreConfig(),
		WithStandardStreams(),
	}
	ops = append(defaultOps, ops...)

	cli := &DockerCli{baseCtx: context.Background()}
	if err := cli.Apply(ops...); err != nil {
		return nil, err
	}
	return cli, nil
}

func getServerHost(hosts []string, defaultToTLS bool) (string, error) {
	switch len(hosts) {
	case 0:
		return dopts.ParseHost(defaultToTLS, os.Getenv(client.EnvOverrideHost))
	case 1:
		return dopts.ParseHost(defaultToTLS, hosts[0])
	default:
		return "", errors.New("Specify only one -H")
	}
}

// UserAgent returns the user agent string used for making API requests
func UserAgent() string {
	return "Docker-Client/" + version.Version + " (" + runtime.GOOS + ")"
}

var defaultStoreEndpoints = []store.NamedTypeGetter{
	store.EndpointTypeGetter(docker.DockerEndpoint, func() any { return &docker.EndpointMeta{} }),
}

// RegisterDefaultStoreEndpoints registers a new named endpoint
// metadata type with the default context store config, so that
// endpoint will be supported by stores using the config returned by
// DefaultContextStoreConfig.
func RegisterDefaultStoreEndpoints(ep ...store.NamedTypeGetter) {
	defaultStoreEndpoints = append(defaultStoreEndpoints, ep...)
}

// DefaultContextStoreConfig returns a new store.Config with the default set of endpoints configured.
func DefaultContextStoreConfig() store.Config {
	return store.NewConfig(
		func() any { return &DockerContext{} },
		defaultStoreEndpoints...,
	)
}
