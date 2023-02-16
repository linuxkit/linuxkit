package util

import (
	"crypto/tls"
	"net/http"

	"github.com/containerd/containerd/remotes"
	"github.com/containerd/containerd/remotes/docker"
)

func NewResolver(username, password string, insecure, plainHTTP bool, configs ...string) remotes.Resolver {

	opts := docker.ResolverOptions{
		PlainHTTP: plainHTTP,
	}
	client := http.DefaultClient
	if insecure {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}
	opts.Client = client

	if username != "" || password != "" {
		opts.Credentials = func(hostName string) (string, string, error) {
			return username, password, nil
		}
		return docker.NewResolver(opts)
	}
	return docker.NewResolver(opts)
}
