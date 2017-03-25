package metadata

import (
	"net/http"
	"sort"

	"github.com/docker/infrakit/pkg/spi"
	"github.com/docker/infrakit/pkg/spi/metadata"
	"github.com/docker/infrakit/pkg/template"
)

// PluginServer returns a Metadata that conforms to the net/rpc rpc call convention.
func PluginServer(p metadata.Plugin) *Metadata {
	return &Metadata{plugin: p}
}

// PluginServerWithTypes which supports multiple types of metadata plugins. The de-multiplexing
// is done by the server's RPC method implementations.
func PluginServerWithTypes(typed map[string]metadata.Plugin) *Metadata {
	return &Metadata{typedPlugins: typed}
}

// Metadata the exported type needed to conform to json-rpc call convention
type Metadata struct {
	plugin       metadata.Plugin
	typedPlugins map[string]metadata.Plugin // by type, as qualified in the name of the plugin
}

// WithBase sets the base plugin to the given plugin object
func (p *Metadata) WithBase(m metadata.Plugin) *Metadata {
	p.plugin = m
	return p
}

// WithTypes sets the typed plugins to the given map of plugins (by type name)
func (p *Metadata) WithTypes(typed map[string]metadata.Plugin) *Metadata {
	p.typedPlugins = typed
	return p
}

// VendorInfo returns a metadata object about the plugin, if the plugin implements it.  See spi.Vendor
func (p *Metadata) VendorInfo() *spi.VendorInfo {
	// TODO(chungers) - support typed plugins
	if p.plugin == nil {
		return nil
	}

	if m, is := p.plugin.(spi.Vendor); is {
		return m.VendorInfo()
	}
	return nil
}

// Funcs implements the template.FunctionExporter method to expose help for plugin's template functions
func (p *Metadata) Funcs() []template.Function {
	f, is := p.plugin.(template.FunctionExporter)
	if !is {
		return []template.Function{}
	}
	return f.Funcs()
}

// Types implements server.TypedFunctionExporter
func (p *Metadata) Types() []string {
	if p.typedPlugins == nil {
		return nil
	}
	list := []string{}
	for k := range p.typedPlugins {
		list = append(list, k)
	}
	return list
}

// FuncsByType implements server.TypedFunctionExporter
func (p *Metadata) FuncsByType(t string) []template.Function {
	if p.typedPlugins == nil {
		return nil
	}
	fp, has := p.typedPlugins[t]
	if !has {
		return nil
	}
	exp, is := fp.(template.FunctionExporter)
	if !is {
		return nil
	}
	return exp.Funcs()
}

// ImplementedInterface returns the interface implemented by this RPC service.
func (p *Metadata) ImplementedInterface() spi.InterfaceSpec {
	return metadata.InterfaceSpec
}

func (p *Metadata) getPlugin(metadataType string) metadata.Plugin {
	if metadataType == "" {
		return p.plugin
	}
	if p, has := p.typedPlugins[metadataType]; has {
		return p
	}
	return nil
}

// List returns a list of child nodes given a path.
func (p *Metadata) List(_ *http.Request, req *ListRequest, resp *ListResponse) error {
	nodes := []string{}

	// the . case - list the typed plugins and the default's first level.
	if len(req.Path) == 0 || req.Path[0] == "" || req.Path[0] == "." {
		if p.plugin != nil {
			n, err := p.plugin.List(req.Path)
			if err != nil {
				return err
			}
			nodes = append(nodes, n...)
		}
		for k := range p.typedPlugins {
			nodes = append(nodes, k)
		}
		sort.Strings(nodes)
		resp.Nodes = nodes
		return nil
	}

	c, has := p.typedPlugins[req.Path[0]]
	if !has {

		if p.plugin == nil {
			return nil
		}

		nodes, err := p.plugin.List(req.Path)
		if err != nil {
			return err
		}
		sort.Strings(nodes)
		resp.Nodes = nodes
		return nil
	}

	nodes, err := c.List(req.Path[1:])
	if err != nil {
		return err
	}

	sort.Strings(nodes)
	resp.Nodes = nodes
	return nil
}

// Get retrieves the value at path given.
func (p *Metadata) Get(_ *http.Request, req *GetRequest, resp *GetResponse) error {
	if len(req.Path) == 0 {
		return nil
	}

	c, has := p.typedPlugins[req.Path[0]]
	if !has {

		if p.plugin == nil {
			return nil
		}

		value, err := p.plugin.Get(req.Path)
		if err != nil {
			return err
		}
		resp.Value = value
		return nil
	}

	value, err := c.Get(req.Path[1:])
	if err != nil {
		return err
	}
	resp.Value = value
	return nil
}
