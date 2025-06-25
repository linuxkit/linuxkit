// Package ssh provides the connection helper for ssh:// URL.
package ssh

import (
	"errors"
	"fmt"
	"net/url"
)

// ParseURL creates a [Spec] from the given ssh URL. It returns an error if
// the URL is using the wrong scheme, contains fragments, query-parameters,
// or contains a password.
func ParseURL(daemonURL string) (*Spec, error) {
	u, err := url.Parse(daemonURL)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Unwrap()
		}
		return nil, fmt.Errorf("invalid SSH URL: %w", err)
	}
	return NewSpec(u)
}

// NewSpec creates a [Spec] from the given ssh URL's properties. It returns
// an error if the URL is using the wrong scheme, contains fragments,
// query-parameters, or contains a password.
func NewSpec(sshURL *url.URL) (*Spec, error) {
	s, err := newSpec(sshURL)
	if err != nil {
		return nil, fmt.Errorf("invalid SSH URL: %w", err)
	}
	return s, nil
}

func newSpec(u *url.URL) (*Spec, error) {
	if u == nil {
		return nil, errors.New("URL is nil")
	}
	if u.Scheme == "" {
		return nil, errors.New("no scheme provided")
	}
	if u.Scheme != "ssh" {
		return nil, errors.New("incorrect scheme: " + u.Scheme)
	}

	var sp Spec

	if u.User != nil {
		sp.User = u.User.Username()
		if _, ok := u.User.Password(); ok {
			return nil, errors.New("plain-text password is not supported")
		}
	}
	sp.Host = u.Hostname()
	if sp.Host == "" {
		return nil, errors.New("hostname is empty")
	}
	sp.Port = u.Port()
	sp.Path = u.Path
	if u.RawQuery != "" {
		return nil, fmt.Errorf("query parameters are not allowed: %q", u.RawQuery)
	}
	if u.Fragment != "" {
		return nil, fmt.Errorf("fragments are not allowed: %q", u.Fragment)
	}

	return &sp, nil
}

// Spec of SSH URL
type Spec struct {
	User string
	Host string
	Port string
	Path string
}

// Args returns args except "ssh" itself combined with optional additional command args
func (sp *Spec) Args(add ...string) []string {
	var args []string
	if sp.User != "" {
		args = append(args, "-l", sp.User)
	}
	if sp.Port != "" {
		args = append(args, "-p", sp.Port)
	}
	args = append(args, "--", sp.Host)
	args = append(args, add...)
	return args
}
