package instance

import (
	"github.com/docker/infrakit/pkg/plugin"
	rpc_client "github.com/docker/infrakit/pkg/rpc/client"
	"github.com/docker/infrakit/pkg/spi/instance"
	"github.com/docker/infrakit/pkg/types"
)

// NewClient returns a plugin interface implementation connected to a plugin
func NewClient(name plugin.Name, socketPath string) (instance.Plugin, error) {
	rpcClient, err := rpc_client.New(socketPath, instance.InterfaceSpec)
	if err != nil {
		return nil, err
	}
	return &client{name: name, client: rpcClient}, nil
}

type client struct {
	name   plugin.Name
	client rpc_client.Client
}

// Validate performs local validation on a provision request.
func (c client) Validate(properties *types.Any) error {
	_, instanceType := c.name.GetLookupAndType()
	req := ValidateRequest{Properties: properties, Type: instanceType}
	resp := ValidateResponse{}

	return c.client.Call("Instance.Validate", req, &resp)
}

// Provision creates a new instance based on the spec.
func (c client) Provision(spec instance.Spec) (*instance.ID, error) {
	_, instanceType := c.name.GetLookupAndType()
	req := ProvisionRequest{Spec: spec, Type: instanceType}
	resp := ProvisionResponse{}

	if err := c.client.Call("Instance.Provision", req, &resp); err != nil {
		return nil, err
	}

	return resp.ID, nil
}

// Label labels the instance
func (c client) Label(instance instance.ID, labels map[string]string) error {
	_, instanceType := c.name.GetLookupAndType()
	req := LabelRequest{Type: instanceType, Instance: instance, Labels: labels}
	resp := LabelResponse{}

	return c.client.Call("Instance.Label", req, &resp)
}

// Destroy terminates an existing instance.
func (c client) Destroy(instance instance.ID) error {
	_, instanceType := c.name.GetLookupAndType()
	req := DestroyRequest{Instance: instance, Type: instanceType}
	resp := DestroyResponse{}

	return c.client.Call("Instance.Destroy", req, &resp)
}

// DescribeInstances returns descriptions of all instances matching all of the provided tags.
func (c client) DescribeInstances(tags map[string]string) ([]instance.Description, error) {
	_, instanceType := c.name.GetLookupAndType()
	req := DescribeInstancesRequest{Tags: tags, Type: instanceType}
	resp := DescribeInstancesResponse{}

	err := c.client.Call("Instance.DescribeInstances", req, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Descriptions, nil
}
