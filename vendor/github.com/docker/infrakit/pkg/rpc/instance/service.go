package instance

import (
	"fmt"
	"net/http"

	"github.com/docker/infrakit/pkg/spi"
	"github.com/docker/infrakit/pkg/spi/instance"
)

// PluginServer returns a RPCService that conforms to the net/rpc rpc call convention.
func PluginServer(p instance.Plugin) *Instance {
	return &Instance{plugin: p, typedPlugins: map[string]instance.Plugin{}}
}

// PluginServerWithTypes which supports multiple types of instance plugins. The de-multiplexing
// is done by the server's RPC method implementations.
func PluginServerWithTypes(typed map[string]instance.Plugin) *Instance {
	return &Instance{typedPlugins: typed}
}

// Instance is the JSON RPC service representing the Instance Plugin.  It must be exported in order to be
// registered by the rpc server package.
type Instance struct {
	plugin       instance.Plugin            // the default plugin
	typedPlugins map[string]instance.Plugin // by type, as qualified in the name of the plugin
}

// VendorInfo returns a metadata object about the plugin, if the plugin implements it.
func (p *Instance) VendorInfo() *spi.VendorInfo {
	// TODO(chungers) - support typed plugins
	if p.plugin == nil {
		return nil
	}

	if m, is := p.plugin.(spi.Vendor); is {
		return m.VendorInfo()
	}
	return nil
}

// SetExampleProperties sets the rpc request with any example properties/ custom type
func (p *Instance) SetExampleProperties(request interface{}) {
	// TODO(chungers) - support typed plugins
	if p.plugin == nil {
		return
	}

	i, is := p.plugin.(spi.InputExample)
	if !is {
		return
	}
	example := i.ExampleProperties()
	if example == nil {
		return
	}

	switch request := request.(type) {
	case *ValidateRequest:
		request.Properties = example
	case *ProvisionRequest:
		request.Spec.Properties = example
	}
}

// ImplementedInterface returns the interface implemented by this RPC service.
func (p *Instance) ImplementedInterface() spi.InterfaceSpec {
	return instance.InterfaceSpec
}

func (p *Instance) getPlugin(instanceType string) instance.Plugin {
	if instanceType == "" {
		return p.plugin
	}
	if p, has := p.typedPlugins[instanceType]; has {
		return p
	}
	return nil
}

// Validate performs local validation on a provision request.
func (p *Instance) Validate(_ *http.Request, req *ValidateRequest, resp *ValidateResponse) error {
	c := p.getPlugin(req.Type)
	if c == nil {
		return fmt.Errorf("no-plugin:%s", req.Type)
	}
	resp.Type = req.Type
	err := c.Validate(req.Properties)
	if err != nil {
		return err
	}
	resp.OK = true
	return nil
}

// Provision creates a new instance based on the spec.
func (p *Instance) Provision(_ *http.Request, req *ProvisionRequest, resp *ProvisionResponse) error {
	resp.Type = req.Type
	c := p.getPlugin(req.Type)
	if c == nil {
		return fmt.Errorf("no-plugin:%s", req.Type)
	}
	id, err := c.Provision(req.Spec)
	if err != nil {
		return err
	}
	resp.ID = id
	return nil
}

// Label labels the instance
func (p *Instance) Label(_ *http.Request, req *LabelRequest, resp *LabelResponse) error {
	resp.Type = req.Type
	c := p.getPlugin(req.Type)
	if c == nil {
		return fmt.Errorf("no-plugin:%s", req.Type)
	}
	err := c.Label(req.Instance, req.Labels)
	if err != nil {
		return err
	}
	resp.OK = true
	return nil
}

// Destroy terminates an existing instance.
func (p *Instance) Destroy(_ *http.Request, req *DestroyRequest, resp *DestroyResponse) error {
	resp.Type = req.Type
	c := p.getPlugin(req.Type)
	if c == nil {
		return fmt.Errorf("no-plugin:%s", req.Type)
	}
	err := c.Destroy(req.Instance)
	if err != nil {
		return err
	}
	resp.OK = true
	return nil
}

// DescribeInstances returns descriptions of all instances matching all of the provided tags.
func (p *Instance) DescribeInstances(_ *http.Request, req *DescribeInstancesRequest, resp *DescribeInstancesResponse) error {
	resp.Type = req.Type
	c := p.getPlugin(req.Type)
	if c == nil {
		return fmt.Errorf("no-plugin:%s", req.Type)
	}
	desc, err := c.DescribeInstances(req.Tags)
	if err != nil {
		return err
	}
	resp.Descriptions = desc
	return nil
}
