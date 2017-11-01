package moby

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/client/auth"
	"github.com/docker/distribution/registry/client/auth/challenge"
	"github.com/docker/distribution/registry/client/transport"
	"github.com/opencontainers/go-digest"
	log "github.com/sirupsen/logrus"
	notaryClient "github.com/theupdateframework/notary/client"
	"github.com/theupdateframework/notary/trustpinning"
	"github.com/theupdateframework/notary/tuf/data"
)

var (
	// ReleasesRole is the role named "releases"
	ReleasesRole = data.RoleName(path.Join(data.CanonicalTargetsRole.String(), "releases"))
)

// TrustedReference parses an image string, and does a notary lookup to verify and retrieve the signed digest reference
func TrustedReference(image string) (reference.Reference, error) {
	ref, err := reference.ParseAnyReference(image)
	if err != nil {
		return nil, err
	}

	// to mimic docker pull: if we have a digest already, it's implicitly trusted
	if digestRef, ok := ref.(reference.Digested); ok {
		return digestRef, nil
	}
	// to mimic docker pull: if we have a digest already, it's implicitly trusted
	if canonicalRef, ok := ref.(reference.Canonical); ok {
		return canonicalRef, nil
	}

	namedRef, ok := ref.(reference.Named)
	if !ok {
		return nil, errors.New("failed to resolve image digest using content trust: reference is not named")
	}
	taggedRef, ok := namedRef.(reference.NamedTagged)
	if !ok {
		return nil, errors.New("failed to resolve image digest using content trust: reference is not tagged")
	}

	gun := taggedRef.Name()
	targetName := taggedRef.Tag()
	server, err := getTrustServer(gun)
	if err != nil {
		return nil, err
	}

	rt, err := GetReadOnlyAuthTransport(server, []string{gun}, "", "", "")
	if err != nil {
		log.Debugf("failed to reach %s notary server for repo: %s, falling back to cache: %v", server, gun, err)
		rt = nil
	}

	nRepo, err := notaryClient.NewFileCachedRepository(
		trustDirectory(),
		data.GUN(gun),
		server,
		rt,
		nil,
		trustpinning.TrustPinConfig{},
	)
	if err != nil {
		return nil, err
	}
	target, err := nRepo.GetTargetByName(targetName, ReleasesRole, data.CanonicalTargetsRole)
	if err != nil {
		return nil, err
	}
	// Only get the tag if it's in the top level targets role or the releases delegation role
	// ignore it if it's in any other delegation roles
	if target.Role != ReleasesRole && target.Role != data.CanonicalTargetsRole {
		return nil, errors.New("not signed in valid role")
	}

	h, ok := target.Hashes["sha256"]
	if !ok {
		return nil, errors.New("no valid hash, expecting sha256")
	}

	dgst := digest.NewDigestFromHex("sha256", hex.EncodeToString(h))

	// Allow returning canonical reference with tag and digest
	return reference.WithDigest(taggedRef, dgst)
}

func getTrustServer(gun string) (string, error) {
	if strings.HasPrefix(gun, "docker.io/") {
		return "https://notary.docker.io", nil
	}
	return "", errors.New("non-hub images not yet supported")
}

func trustDirectory() string {
	return filepath.Join(MobyDir, "trust")
}

type credentialStore struct {
	username      string
	password      string
	refreshTokens map[string]string
}

func (tcs *credentialStore) Basic(url *url.URL) (string, string) {
	return tcs.username, tcs.password
}

// refresh tokens are the long lived tokens that can be used instead of a password
func (tcs *credentialStore) RefreshToken(u *url.URL, service string) string {
	return tcs.refreshTokens[service]
}

func (tcs *credentialStore) SetRefreshToken(u *url.URL, service string, token string) {
	if tcs.refreshTokens != nil {
		tcs.refreshTokens[service] = token
	}
}

// GetReadOnlyAuthTransport gets the Auth Transport used to communicate with notary
func GetReadOnlyAuthTransport(server string, scopes []string, username, password, rootCAPath string) (http.RoundTripper, error) {
	httpsTransport, err := httpsTransport(rootCAPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v2/", server), nil)
	if err != nil {
		return nil, err
	}
	pingClient := &http.Client{
		Transport: httpsTransport,
		Timeout:   5 * time.Second,
	}
	resp, err := pingClient.Do(req)
	if err != nil {
		return nil, err
	}
	challengeManager := challenge.NewSimpleManager()
	if err := challengeManager.AddResponse(resp); err != nil {
		return nil, err
	}

	creds := credentialStore{
		username:      username,
		password:      password,
		refreshTokens: make(map[string]string),
	}

	var scopeObjs []auth.Scope
	for _, scopeName := range scopes {
		scopeObjs = append(scopeObjs, auth.RepositoryScope{
			Repository: scopeName,
			Actions:    []string{"pull"},
		})
	}

	// allow setting multiple scopes so we don't have to reauth
	tokenHandler := auth.NewTokenHandlerWithOptions(auth.TokenHandlerOptions{
		Transport:   httpsTransport,
		Credentials: &creds,
		Scopes:      scopeObjs,
	})

	authedTransport := transport.NewTransport(httpsTransport, auth.NewAuthorizer(challengeManager, tokenHandler))
	return authedTransport, nil
}

func httpsTransport(caFile string) (*http.Transport, error) {
	tlsConfig := &tls.Config{}
	transport := http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     tlsConfig,
	}
	// Override with the system cert pool if the caFile was empty
	if caFile != "" {
		certPool := x509.NewCertPool()
		pems, err := ioutil.ReadFile(caFile)
		if err != nil {
			return nil, err
		}
		certPool.AppendCertsFromPEM(pems)
		transport.TLSClientConfig.RootCAs = certPool
	}
	return &transport, nil
}
