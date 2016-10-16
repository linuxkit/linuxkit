package instance

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/docker/infrakit/plugin"
	"github.com/docker/infrakit/plugin/util"
	"github.com/docker/infrakit/plugin/util/server"
	"github.com/docker/infrakit/spi/instance"
)

type client struct {
	c plugin.Callable
}

type instanceServer struct {
	plugin instance.Plugin
}

// PluginServer returns an instance of the Plugin
func PluginServer(p instance.Plugin) http.Handler {

	i := &instanceServer{plugin: p}
	return server.BuildHandler([]func() (plugin.Endpoint, plugin.Handler){
		i.validate,
		i.provision,
		i.destroy,
		i.describeInstances,
	})
}

// PluginClient returns an instance of the Plugin
func PluginClient(c plugin.Callable) instance.Plugin {
	return &client{c: c}
}

func (c *client) Validate(req json.RawMessage) error {
	_, err := c.c.Call(&util.HTTPEndpoint{Method: "POST", Path: "/Instance.Validate"}, &req, nil)
	return err
}

func (s *instanceServer) validate() (plugin.Endpoint, plugin.Handler) {
	return &util.HTTPEndpoint{Method: "POST", Path: "/Instance.Validate"},

		func(vars map[string]string, body io.Reader) (result interface{}, err error) {
			buff, err := ioutil.ReadAll(body)
			if err != nil {
				return nil, err
			}
			// TODO -- change validate to return bool, error so we can tell if it's network vs semantic
			err = s.plugin.Validate(json.RawMessage(buff))
			return nil, err
		}
}

func (c *client) Provision(spec instance.Spec) (*instance.ID, error) {
	envelope := struct {
		ID *instance.ID
	}{}
	_, err := c.c.Call(&util.HTTPEndpoint{Method: "POST", Path: "/Instance.Provision"}, spec, &envelope)
	return envelope.ID, err
}

func (s *instanceServer) provision() (plugin.Endpoint, plugin.Handler) {
	return &util.HTTPEndpoint{Method: "POST", Path: "/Instance.Provision"},

		func(vars map[string]string, body io.Reader) (result interface{}, err error) {
			spec := instance.Spec{}
			err = json.NewDecoder(body).Decode(&spec)
			if err != nil {
				return nil, err
			}
			id, err := s.plugin.Provision(spec)
			return struct{ ID *instance.ID }{ID: id}, err
		}
}

func (c *client) Destroy(instance instance.ID) error {
	_, err := c.c.Call(&util.HTTPEndpoint{Method: "POST", Path: fmt.Sprintf("/Instance.Destroy/%v", instance)}, nil, nil)
	return err
}

func (s *instanceServer) destroy() (plugin.Endpoint, plugin.Handler) {
	return &util.HTTPEndpoint{Method: "POST", Path: "/Instance.Destroy/{id}"},

		func(vars map[string]string, body io.Reader) (result interface{}, err error) {
			err = s.plugin.Destroy(instance.ID(vars["id"]))
			return nil, err
		}
}

func (c *client) DescribeInstances(tags map[string]string) ([]instance.Description, error) {
	result := []instance.Description{}
	_, err := c.c.Call(&util.HTTPEndpoint{Method: "POST", Path: "/Instance.DescribeInstances"}, tags, &result)
	return result, err

}

func (s *instanceServer) describeInstances() (plugin.Endpoint, plugin.Handler) {
	return &util.HTTPEndpoint{Method: "POST", Path: "/Instance.DescribeInstances"},

		func(vars map[string]string, body io.Reader) (result interface{}, err error) {
			tags := map[string]string{}
			err = json.NewDecoder(body).Decode(&tags)
			if err != nil {
				return nil, err
			}
			return s.plugin.DescribeInstances(tags)
		}
}
