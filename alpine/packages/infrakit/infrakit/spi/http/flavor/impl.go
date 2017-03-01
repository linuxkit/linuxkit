package flavor

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/docker/infrakit/plugin"
	"github.com/docker/infrakit/plugin/group/types"
	"github.com/docker/infrakit/plugin/util"
	"github.com/docker/infrakit/plugin/util/server"
	"github.com/docker/infrakit/spi/flavor"
	"github.com/docker/infrakit/spi/instance"
)

type client struct {
	c plugin.Callable
}

type flavorServer struct {
	plugin flavor.Plugin
}

// PluginClient returns an instance of the Plugin
func PluginClient(c plugin.Callable) flavor.Plugin {
	return &client{c: c}
}

// PluginServer returns an instance of the Plugin
func PluginServer(p flavor.Plugin) http.Handler {

	f := &flavorServer{plugin: p}
	return server.BuildHandler([]func() (plugin.Endpoint, plugin.Handler){
		f.validate,
		f.prepare,
		f.healthy,
	})
}

type validateRequest struct {
	Properties *json.RawMessage
	Allocation types.AllocationMethod
}

func (c *client) Validate(flavorProperties json.RawMessage, allocation types.AllocationMethod) error {
	request := validateRequest{Properties: &flavorProperties, Allocation: allocation}
	_, err := c.c.Call(&util.HTTPEndpoint{Method: "POST", Path: "/Flavor.Validate"}, request, nil)
	return err
}

func (s *flavorServer) validate() (plugin.Endpoint, plugin.Handler) {
	return &util.HTTPEndpoint{Method: "POST", Path: "/Flavor.Validate"},

		func(vars map[string]string, body io.Reader) (result interface{}, err error) {
			request := validateRequest{}
			if err := json.NewDecoder(body).Decode(&request); err != nil {
				return nil, err
			}

			var properties json.RawMessage
			if request.Properties != nil {
				properties = *request.Properties
			}

			return nil, s.plugin.Validate(properties, request.Allocation)
		}
}

type prepareRequest struct {
	Properties *json.RawMessage
	Instance   instance.Spec
	Allocation types.AllocationMethod
}

func (c *client) Prepare(
	flavorProperties json.RawMessage,
	spec instance.Spec,
	allocation types.AllocationMethod) (instance.Spec, error) {

	request := prepareRequest{Properties: &flavorProperties, Instance: spec, Allocation: allocation}
	instanceSpec := instance.Spec{}
	_, err := c.c.Call(&util.HTTPEndpoint{Method: "POST", Path: "/Flavor.PreProvision"}, request, &instanceSpec)
	return instanceSpec, err
}

func (s *flavorServer) prepare() (plugin.Endpoint, plugin.Handler) {
	return &util.HTTPEndpoint{Method: "POST", Path: "/Flavor.PreProvision"},

		func(vars map[string]string, body io.Reader) (result interface{}, err error) {
			request := prepareRequest{}

			if err := json.NewDecoder(body).Decode(&request); err != nil {
				return nil, err
			}

			var arg1 json.RawMessage
			if request.Properties != nil {
				arg1 = *request.Properties
			}

			return s.plugin.Prepare(arg1, request.Instance, request.Allocation)
		}
}

type healthRequest struct {
	Properties *json.RawMessage
	Instance   instance.Description
}

type healthResponse struct {
	Health flavor.Health
}

func (c *client) Healthy(flavorProperties json.RawMessage, inst instance.Description) (flavor.Health, error) {
	request := healthRequest{Properties: &flavorProperties, Instance: inst}
	response := healthResponse{}
	_, err := c.c.Call(&util.HTTPEndpoint{Method: "POST", Path: "/Flavor.Healthy"}, request, &response)
	return response.Health, err
}

func (s *flavorServer) healthy() (plugin.Endpoint, plugin.Handler) {
	return &util.HTTPEndpoint{Method: "POST", Path: "/Flavor.Healthy"},

		func(vars map[string]string, body io.Reader) (result interface{}, err error) {
			request := healthRequest{}
			err = json.NewDecoder(body).Decode(&request)
			if err != nil {
				return nil, err
			}
			health, err := s.plugin.Healthy(types.RawMessage(request.Properties), request.Instance)
			return healthResponse{Health: health}, err
		}
}
