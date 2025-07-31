package registry

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// proxy is a map of registry names to proxy URLs.
var proxy = make(map[string]string)

// certs is a slice of certificates to be used for secure connections.
var certs = make([][]byte, 0)

func SetProxy(registry, url string) {
	if url == "" {
		delete(proxy, registry)
	} else {
		proxy[registry] = url
	}
}

func AddCert(cert []byte) {
	certs = append(certs, cert)
}

// Remote implements the functions of
// github.com/google/go-containerregistry/pkg/v1/remote, while possibly pre-configured for
// items like proxies, mirrors, authentication, or other settings.
type Remote struct {
	proxy map[string]string
}

// GetRemote returns a Remote
func GetRemote() *Remote {
	return &Remote{
		proxy: proxy,
	}
}

func (r *Remote) Get(ref name.Reference, options ...remote.Option) (*remote.Descriptor, error) {
	var err error
	ref, err = r.rewriteReference(ref)
	if err != nil {
		return nil, fmt.Errorf("rewriting reference %q: %w", ref.Name(), err)
	}
	opts, err := r.rewriteTLSTransport(options)
	if err != nil {
		return nil, fmt.Errorf("rewriting TLS transport for %q: %w", ref.Name(), err)
	}

	return remote.Get(ref, opts...)
}

func (r *Remote) Head(ref name.Reference, options ...remote.Option) (*v1.Descriptor, error) {
	var err error
	ref, err = r.rewriteReference(ref)
	if err != nil {
		return nil, fmt.Errorf("rewriting reference %q: %w", ref.Name(), err)
	}
	opts, err := r.rewriteTLSTransport(options)
	if err != nil {
		return nil, fmt.Errorf("rewriting TLS transport for %q: %w", ref.Name(), err)
	}

	return remote.Head(ref, opts...)
}

func (r *Remote) Tag(ref name.Tag, t remote.Taggable, options ...remote.Option) error {
	opts, err := r.rewriteTLSTransport(options)
	if err != nil {
		return fmt.Errorf("rewriting TLS transport for %q: %w", ref.Name(), err)
	}
	return remote.Tag(ref, t, opts...)
}

func (r *Remote) Push(ref name.Reference, t remote.Taggable, options ...remote.Option) error {
	var err error
	ref, err = r.rewriteReference(ref)
	if err != nil {
		return fmt.Errorf("rewriting reference %q: %w", ref.Name(), err)
	}
	opts, err := r.rewriteTLSTransport(options)
	if err != nil {
		return fmt.Errorf("rewriting TLS transport for %q: %w", ref.Name(), err)
	}

	return remote.Push(ref, t, opts...)
}

func (r *Remote) Put(ref name.Reference, t remote.Taggable, options ...remote.Option) error {
	var err error
	ref, err = r.rewriteReference(ref)
	if err != nil {
		return fmt.Errorf("rewriting reference %q: %w", ref.Name(), err)
	}
	opts, err := r.rewriteTLSTransport(options)
	if err != nil {
		return fmt.Errorf("rewriting TLS transport for %q: %w", ref.Name(), err)
	}

	return remote.Put(ref, t, opts...)
}

func (r *Remote) Write(ref name.Reference, img v1.Image, options ...remote.Option) error {
	var err error
	ref, err = r.rewriteReference(ref)
	if err != nil {
		return fmt.Errorf("rewriting reference %q: %w", ref.Name(), err)
	}
	opts, err := r.rewriteTLSTransport(options)
	if err != nil {
		return fmt.Errorf("rewriting TLS transport for %q: %w", ref.Name(), err)
	}

	return remote.Write(ref, img, opts...)
}

func (r *Remote) WriteIndex(ref name.Reference, ii v1.ImageIndex, options ...remote.Option) error {
	var err error
	ref, err = r.rewriteReference(ref)
	if err != nil {
		return fmt.Errorf("rewriting reference %q: %w", ref.Name(), err)
	}
	opts, err := r.rewriteTLSTransport(options)
	if err != nil {
		return fmt.Errorf("rewriting TLS transport for %q: %w", ref.Name(), err)
	}

	return remote.WriteIndex(ref, ii, opts...)
}

func (r *Remote) WriteLayer(repo name.Repository, layer v1.Layer, options ...remote.Option) error {
	var err error
	repo, err = r.rewriteRepository(repo)
	if err != nil {
		return fmt.Errorf("rewriting repository %q: %w", repo.Name(), err)
	}
	opts, err := r.rewriteTLSTransport(options)
	if err != nil {
		return fmt.Errorf("rewriting TLS transport for %q: %w", repo.Name(), err)
	}

	return remote.WriteLayer(repo, layer, opts...)
}

func (r *Remote) List(repo name.Repository, options ...remote.Option) ([]string, error) {
	var err error
	repo, err = r.rewriteRepository(repo)
	if err != nil {
		return nil, fmt.Errorf("rewriting repository %q: %w", repo.Name(), err)
	}
	opts, err := r.rewriteTLSTransport(options)
	if err != nil {
		return nil, fmt.Errorf("rewriting TLS transport for %q: %w", repo.Name(), err)
	}

	return remote.List(repo, opts...)
}

func (r *Remote) Layer(ref name.Digest, options ...remote.Option) (v1.Layer, error) {
	var err error
	ref, err = r.rewriteDigest(ref)
	if err != nil {
		return nil, fmt.Errorf("rewriting digest %q: %w", ref.Name(), err)
	}
	opts, err := r.rewriteTLSTransport(options)
	if err != nil {
		return nil, fmt.Errorf("rewriting TLS transport for %q: %w", ref.Name(), err)
	}
	return remote.Layer(ref, opts...)
}

func (r *Remote) Index(ref name.Reference, options ...remote.Option) (v1.ImageIndex, error) {
	var err error
	ref, err = r.rewriteReference(ref)
	if err != nil {
		return nil, fmt.Errorf("rewriting reference %q: %w", ref.Name(), err)
	}
	opts, err := r.rewriteTLSTransport(options)
	if err != nil {
		return nil, fmt.Errorf("rewriting TLS transport for %q: %w", ref.Name(), err)
	}

	return remote.Index(ref, opts...)
}

func (r *Remote) Image(ref name.Reference, options ...remote.Option) (v1.Image, error) {
	var err error
	ref, err = r.rewriteReference(ref)
	if err != nil {
		return nil, fmt.Errorf("rewriting reference %q: %w", ref.Name(), err)
	}
	opts, err := r.rewriteTLSTransport(options)
	if err != nil {
		return nil, fmt.Errorf("rewriting TLS transport for %q: %w", ref.Name(), err)
	}

	return remote.Image(ref, opts...)
}

func (r *Remote) Delete(ref name.Reference, options ...remote.Option) error {
	var err error
	ref, err = r.rewriteReference(ref)
	if err != nil {
		return fmt.Errorf("rewriting reference %q: %w", ref.Name(), err)
	}
	opts, err := r.rewriteTLSTransport(options)
	if err != nil {
		return fmt.Errorf("rewriting TLS transport for %q: %w", ref.Name(), err)
	}

	return remote.Delete(ref, opts...)
}

func (r *Remote) rewriteTLSTransport(options []remote.Option) ([]remote.Option, error) {
	// If there are no certs, return the options as is
	if len(certs) == 0 {
		return options, nil
	}

	caCertPool, err := x509.SystemCertPool()
	if err != nil || caCertPool == nil {
		caCertPool = x509.NewCertPool()
	}
	for _, caCert := range certs {
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to append CA certificate")
		}
	}

	baseTransport := remote.DefaultTransport.(*http.Transport).Clone()
	if baseTransport.TLSClientConfig == nil {
		baseTransport.TLSClientConfig = &tls.Config{}
	}
	baseTransport.TLSClientConfig.RootCAs = caCertPool

	// Add the certificates to the options
	newOptions := make([]remote.Option, 0, len(options)+1)
	newOptions = append(newOptions, options...)
	newOptions = append(newOptions, remote.WithTransport(
		baseTransport,
	))

	return newOptions, nil
}

func (r *Remote) rewriteReference(ref name.Reference) (name.Reference, error) {
	newRepo, opts, err := r.rewriteRepositoryBase(ref.Context())
	if err != nil {
		return nil, fmt.Errorf("rewriting repository %q: %w", ref.Context().Name(), err)
	}

	switch typed := ref.(type) {
	case name.Tag:
		return name.NewTag(newRepo+":"+typed.TagStr(), opts...)
	case name.Digest:
		return name.NewDigest(newRepo+"@"+typed.DigestStr(), opts...)
	default:
		return nil, fmt.Errorf("unsupported reference type: %T", ref)
	}
}

func (r *Remote) rewriteRepository(repo name.Repository) (name.Repository, error) {
	newRepo, opts, err := r.rewriteRepositoryBase(repo)
	if err != nil {
		return repo, fmt.Errorf("rewriting repository %q: %w", repo.Name(), err)
	}

	return name.NewRepository(newRepo, opts...)
}

func (r *Remote) rewriteDigest(dig name.Digest) (name.Digest, error) {
	newRepo, opts, err := r.rewriteRepositoryBase(dig.Context())
	if err != nil {
		return dig, fmt.Errorf("rewriting repository %q: %w", dig, err)
	}

	return name.NewDigest(newRepo, opts...)
}

func (r *Remote) rewriteRepositoryBase(repo name.Repository) (string, []name.Option, error) {
	originalRegistry := repo.RegistryStr()
	mirror := r.resolveMirror(originalRegistry)

	// No rewrite needed
	if mirror == "" || mirror == originalRegistry {
		return repo.RepositoryStr(), nil, nil
	}

	// get mirror protocol and separate host+path
	var (
		rest     string
		insecure bool
		opts     []name.Option
	)

	switch {
	case strings.HasPrefix(mirror, "http://"):
		insecure = true
		rest = mirror[len("http://"):]
	case strings.HasPrefix(mirror, "https://"):
		insecure = false
		rest = mirror[len("https://"):]
	default:
		insecure = false // Default to https if no protocol is specified
		rest = mirror
	}
	if insecure {
		opts = append(opts, name.Insecure)
	}
	opts = append(opts, name.WeakValidation)
	// Build the new repository: mirror/foo/bar
	// strip off trailing slash if present, so we do not end up with double slashes
	newRepo := strings.TrimSuffix(rest, "/") + "/" + repo.RepositoryStr()

	return newRepo, opts, nil
}

func (r *Remote) resolveMirror(registry string) string {
	if r.proxy == nil {
		return registry
	}
	if val, ok := r.proxy[registry]; ok {
		return val
	}
	if val, ok := r.proxy["*"]; ok {
		return val
	}
	return registry
}
