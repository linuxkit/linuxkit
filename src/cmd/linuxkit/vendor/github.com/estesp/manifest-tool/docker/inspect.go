package docker

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/docker/cli/cli/config"
	"github.com/docker/distribution/manifest/manifestlist"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/api/errcode"
	v2 "github.com/docker/distribution/registry/api/v2"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/docker/api"
	engineTypes "github.com/docker/docker/api/types"
	registryTypes "github.com/docker/docker/api/types/registry"
	"github.com/docker/docker/api/types/versions"
	"github.com/docker/docker/distribution"
	"github.com/docker/docker/image"
	"github.com/docker/docker/registry"
	"github.com/estesp/manifest-tool/types"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

const (
	// DefaultHostname is the default built-in registry (DockerHub)
	DefaultHostname = "docker.io"
	// LegacyDefaultHostname is the old hostname used for DockerHub
	LegacyDefaultHostname = "index.docker.io"
	// DefaultRepoPrefix is the prefix used for official images in DockerHub
	DefaultRepoPrefix = "library/"
)

type existingTokenHandler struct {
	token string
}

type dumbCredentialStore struct {
	auth *engineTypes.AuthConfig
}

func (dcs dumbCredentialStore) Basic(*url.URL) (string, string) {
	return dcs.auth.Username, dcs.auth.Password
}

func (th *existingTokenHandler) Scheme() string {
	return "bearer"
}

func (th *existingTokenHandler) AuthorizeRequest(req *http.Request, params map[string]string) error {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", th.token))
	return nil
}

func (dcs dumbCredentialStore) RefreshToken(*url.URL, string) string {
	return dcs.auth.IdentityToken
}

func (dcs dumbCredentialStore) SetRefreshToken(*url.URL, string, string) {
}

// fallbackError wraps an error that can possibly allow fallback to a different
// endpoint.
type fallbackError struct {
	// err is the error being wrapped.
	err error
	// confirmedV2 is set to true if it was confirmed that the registry
	// supports the v2 protocol. This is used to limit fallbacks to the v1
	// protocol.
	confirmedV2 bool
	transportOK bool
}

// Error renders the FallbackError as a string.
func (f fallbackError) Error() string {
	return f.err.Error()
}

type manifestFetcher interface {
	Fetch(ctx context.Context, ref reference.Named) ([]types.ImageInspect, error)
}

func validateName(name string) error {
	distref, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return err
	}
	hostname, _ := splitHostname(distref.String())
	if hostname == "" {
		return fmt.Errorf("Please use a fully qualified repository name")
	}
	return nil
}

// splitHostname splits a repository name to hostname and remotename string.
// If no valid hostname is found, the default hostname is used. Repository name
// needs to be already validated before.
func splitHostname(name string) (hostname, remoteName string) {
	i := strings.IndexRune(name, '/')
	if i == -1 || (!strings.ContainsAny(name[:i], ".:") && name[:i] != "localhost") {
		hostname, remoteName = DefaultHostname, name
	} else {
		hostname, remoteName = name[:i], name[i+1:]
	}
	if hostname == LegacyDefaultHostname {
		hostname = DefaultHostname
	}
	if hostname == DefaultHostname && !strings.ContainsRune(remoteName, '/') {
		remoteName = DefaultRepoPrefix + remoteName
	}
	return
}

func checkHTTPRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 10 {
		return errors.New("stopped after 10 redirects")
	}

	if len(via) > 0 {
		for headerName, headerVals := range via[0].Header {
			if headerName != "Accept" && headerName != "Range" {
				continue
			}
			for _, val := range headerVals {
				// Don't add to redirected request if redirected
				// request already has a header with the same
				// name and value.
				hasValue := false
				for _, existingVal := range req.Header[headerName] {
					if existingVal == val {
						hasValue = true
						break
					}
				}
				if !hasValue {
					req.Header.Add(headerName, val)
				}
			}
		}
	}

	return nil
}

// GetImageData takes registry authentication information and a name of the image to return information about
func GetImageData(a *types.AuthInfo, name string, insecure, includeTags bool) ([]types.ImageInspect, *registry.RepositoryInfo, error) {
	if err := validateName(name); err != nil {
		return nil, nil, err
	}
	ref, err := reference.ParseNormalizedNamed(name)
	if err != nil {
		return nil, nil, err
	}
	repoInfo, err := registry.ParseRepositoryInfo(ref)
	if err != nil {
		return nil, nil, err
	}
	authConfig, err := getAuthConfig(a, repoInfo.Index)
	if err != nil {
		return nil, nil, err
	}
	if err := validateRepoName(repoInfo.Name.Name()); err != nil {
		return nil, nil, err
	}
	options := registry.ServiceOptions{}
	if insecure {
		options.InsecureRegistries = append(options.InsecureRegistries, reference.Domain(repoInfo.Name))
	}
	registryService, err := registry.NewService(options)
	if err != nil {
		return nil, nil, err
	}

	endpoints, err := registryService.LookupPullEndpoints(reference.Domain(repoInfo.Name))
	if err != nil {
		return nil, nil, err
	}
	logrus.Debugf("endpoints: %v", endpoints)

	var (
		ctx                    = context.Background()
		lastErr                error
		discardNoSupportErrors bool
		foundImages            []types.ImageInspect
		confirmedV2            bool
		confirmedTLSRegistries = make(map[string]struct{})
	)

	for _, endpoint := range endpoints {
		// make sure I can reach the registry, same as docker pull does
		if endpoint.Version == registry.APIVersion1 {
			logrus.Debugf("Skipping v1 endpoint %s; manifest list requires v2", endpoint.URL)
			continue
		}
		if insecure && endpoint.URL.Scheme == "https" {
			logrus.Debugf("Skipping https endpoint for insecure registry")
			continue
		}

		if endpoint.URL.Scheme != "https" {
			if _, confirmedTLS := confirmedTLSRegistries[endpoint.URL.Host]; confirmedTLS {
				logrus.Debugf("Skipping non-TLS endpoint %s for host/port that appears to use TLS", endpoint.URL)
				continue
			}
		}
		if insecure {
			endpoint.TLSConfig.InsecureSkipVerify = true
		}

		logrus.Debugf("Trying to fetch image manifest of %s repository from %s %s", repoInfo.Name.Name(), endpoint.URL, endpoint.Version)

		fetcher, err := newManifestFetcher(endpoint, repoInfo, authConfig, registryService, includeTags)
		if err != nil {
			lastErr = err
			continue
		}

		if foundImages, err = fetcher.Fetch(ctx, ref); err != nil {
			// Was this fetch cancelled? If so, don't try to fall back.
			fallback := false
			select {
			case <-ctx.Done():
			default:
				if fallbackErr, ok := err.(fallbackError); ok {
					fallback = true
					confirmedV2 = confirmedV2 || fallbackErr.confirmedV2
					if fallbackErr.transportOK && endpoint.URL.Scheme == "https" {
						confirmedTLSRegistries[endpoint.URL.Host] = struct{}{}
					}
					err = fallbackErr.err
				}
			}
			if fallback {
				if _, ok := err.(distribution.ErrNoSupport); !ok {
					// Because we found an error that's not ErrNoSupport, discard all subsequent ErrNoSupport errors.
					discardNoSupportErrors = true
					// save the current error
					lastErr = err
				} else if !discardNoSupportErrors {
					// Save the ErrNoSupport error, because it's either the first error or all encountered errors
					// were also ErrNoSupport errors.
					lastErr = err
				}
				continue
			}
			logrus.Infof("Not continuing with pull after error: %v", err)
			return nil, nil, err
		}

		return foundImages, repoInfo, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no endpoints found for %s", ref.String())
	}

	return nil, nil, lastErr
}

func newManifestFetcher(endpoint registry.APIEndpoint, repoInfo *registry.RepositoryInfo, authConfig engineTypes.AuthConfig, registryService registry.Service, includeTags bool) (manifestFetcher, error) {
	switch endpoint.Version {
	case registry.APIVersion2:
		return &v2ManifestFetcher{
			endpoint:    endpoint,
			authConfig:  authConfig,
			service:     registryService,
			repoInfo:    repoInfo,
			includeTags: includeTags,
		}, nil
	case registry.APIVersion1:
		return &v1ManifestFetcher{
			endpoint:   endpoint,
			authConfig: authConfig,
			service:    registryService,
			repoInfo:   repoInfo,
		}, nil
	}
	return nil, fmt.Errorf("unknown version %d for registry %s", endpoint.Version, endpoint.URL)
}

func getAuthConfig(a *types.AuthInfo, index *registryTypes.IndexInfo) (engineTypes.AuthConfig, error) {

	var (
		username      = a.Username
		password      = a.Password
		cfg           = a.DockerCfg
		defAuthConfig = engineTypes.AuthConfig{
			Username: a.Username,
			Password: a.Password,
			Email:    "stub@example.com",
		}
	)

	if username != "" && password != "" {
		return defAuthConfig, nil
	}

	confFile, err := config.Load(cfg)
	if err != nil {
		return engineTypes.AuthConfig{}, err
	}
	authConfig := registry.ResolveAuthConfig(confFile.AuthConfigs, index)
	logrus.Debugf("authConfig for %s: %v", index.Name, authConfig.Username)

	return authConfig, nil
}

func validateRepoName(name string) error {
	if name == "" {
		return fmt.Errorf("Repository name can't be empty")
	}
	if name == api.NoBaseImageSpecifier {
		return fmt.Errorf("'%s' is a reserved name", api.NoBaseImageSpecifier)
	}
	return nil
}

func makeImageInspect(img *image.Image, tag string, mfInfo manifestInfo, mediaType string, tagList []string) *types.ImageInspect {
	var digest string
	if err := mfInfo.digest.Validate(); err == nil {
		digest = mfInfo.digest.String()
	}

	// for manifest lists, we only want to display the basic info that this is
	// a manifest list and its digest information:
	if mediaType == manifestlist.MediaTypeManifestList {
		return &types.ImageInspect{
			MediaType: mediaType,
			Digest:    digest,
		}
	}

	var digests []string
	for _, blobDigest := range mfInfo.blobDigests {
		digests = append(digests, blobDigest.String())
	}
	return &types.ImageInspect{
		Size:            mfInfo.length,
		MediaType:       mediaType,
		Tag:             tag,
		Digest:          digest,
		RepoTags:        tagList,
		Comment:         img.Comment,
		Created:         img.Created.Format(time.RFC3339Nano),
		ContainerConfig: &img.ContainerConfig,
		DockerVersion:   img.DockerVersion,
		Author:          img.Author,
		Config:          img.Config,
		Architecture:    img.Architecture,
		Os:              img.OS,
		OSVersion:       img.OSVersion,
		OSFeatures:      img.OSFeatures,
		References:      digests,
		Layers:          mfInfo.layers,
		Platform:        mfInfo.platform,
		CanonicalJSON:   mfInfo.jsonBytes,
	}
}

func makeRawConfigFromV1Config(imageJSON []byte, rootfs *image.RootFS, history []image.History) (map[string]*json.RawMessage, error) {
	var dver struct {
		DockerVersion string `json:"docker_version"`
	}

	if err := json.Unmarshal(imageJSON, &dver); err != nil {
		return nil, err
	}

	useFallback := versions.LessThan(dver.DockerVersion, "1.8.3")

	if useFallback {
		var v1Image image.V1Image
		err := json.Unmarshal(imageJSON, &v1Image)
		if err != nil {
			return nil, err
		}
		imageJSON, err = json.Marshal(v1Image)
		if err != nil {
			return nil, err
		}
	}

	var c map[string]*json.RawMessage
	if err := json.Unmarshal(imageJSON, &c); err != nil {
		return nil, err
	}

	c["rootfs"] = rawJSON(rootfs)
	c["history"] = rawJSON(history)

	return c, nil
}

func rawJSON(value interface{}) *json.RawMessage {
	jsonval, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return (*json.RawMessage)(&jsonval)
}

func continueOnError(err error) bool {
	switch v := err.(type) {
	case errcode.Errors:
		if len(v) == 0 {
			return true
		}
		return continueOnError(v[0])
	case distribution.ErrNoSupport:
		return continueOnError(v.Err)
	case errcode.Error:
		return shouldV2Fallback(v)
	case *client.UnexpectedHTTPResponseError:
		return true
	case ImageConfigPullError:
		return false
	case error:
		return !strings.Contains(err.Error(), strings.ToLower(syscall.ENOSPC.Error()))
	}
	// let's be nice and fallback if the error is a completely
	// unexpected one.
	// If new errors have to be handled in some way, please
	// add them to the switch above.
	return true
}

// shouldV2Fallback returns true if this error is a reason to fall back to v1.
func shouldV2Fallback(err errcode.Error) bool {
	switch err.Code {
	case errcode.ErrorCodeUnauthorized, v2.ErrorCodeManifestUnknown, v2.ErrorCodeNameUnknown:
		return true
	}
	return false
}
