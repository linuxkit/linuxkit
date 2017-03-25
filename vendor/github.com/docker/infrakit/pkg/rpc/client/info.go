package client

import (
	"encoding/json"
	"net"
	"net/http"

	"github.com/docker/infrakit/pkg/plugin"
	"github.com/docker/infrakit/pkg/rpc"
	"github.com/docker/infrakit/pkg/template"
)

// NewPluginInfoClient returns a plugin informer that can give metadata about a plugin
func NewPluginInfoClient(socketPath string) *InfoClient {
	dialUnix := func(proto, addr string) (conn net.Conn, err error) {
		return net.Dial("unix", socketPath)
	}
	return &InfoClient{client: &http.Client{Transport: &http.Transport{Dial: dialUnix}}}
}

// InfoClient is the client for retrieving plugin info
type InfoClient struct {
	client *http.Client
}

// GetInfo implements the Info interface and returns the metadata about the plugin
func (i *InfoClient) GetInfo() (plugin.Info, error) {
	meta := plugin.Info{}
	resp, err := i.client.Get("http://d" + rpc.URLAPI)
	if err != nil {
		return meta, err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&meta)
	return meta, err
}

// GetFunctions returns metadata about the plugin's template functions, if the plugin supports templating.
func (i *InfoClient) GetFunctions() (map[string][]template.Function, error) {
	meta := map[string][]template.Function{}
	resp, err := i.client.Get("http://d" + rpc.URLFunctions)
	if err != nil {
		return meta, err
	}
	defer resp.Body.Close()
	err = json.NewDecoder(resp.Body).Decode(&meta)
	return meta, err
}
