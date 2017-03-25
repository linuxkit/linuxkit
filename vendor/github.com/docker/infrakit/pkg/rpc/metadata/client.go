package metadata

import (
	rpc_client "github.com/docker/infrakit/pkg/rpc/client"
	"github.com/docker/infrakit/pkg/spi/metadata"
	"github.com/docker/infrakit/pkg/types"
)

// NewClient returns a plugin interface implementation connected to a remote plugin
func NewClient(socketPath string) (metadata.Plugin, error) {
	rpcClient, err := rpc_client.New(socketPath, metadata.InterfaceSpec)
	if err != nil {
		return nil, err
	}
	return &client{client: rpcClient}, nil
}

// Adapt converts a rpc client to a Metadata plugin object
func Adapt(rpcClient rpc_client.Client) metadata.Plugin {
	return &client{client: rpcClient}
}

type client struct {
	client rpc_client.Client
}

// List returns a list of nodes under path.
func (c client) List(path metadata.Path) ([]string, error) {
	req := ListRequest{Path: path}
	resp := ListResponse{}
	err := c.client.Call("Metadata.List", req, &resp)
	return resp.Nodes, err
}

// Get retrieves the metadata at path.
func (c client) Get(path metadata.Path) (*types.Any, error) {
	req := GetRequest{Path: path}
	resp := GetResponse{}
	err := c.client.Call("Metadata.Get", req, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Value, err
}
