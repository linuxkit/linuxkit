package group

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/docker/infrakit/plugin"
	"github.com/docker/infrakit/plugin/util"
	"github.com/docker/infrakit/plugin/util/server"
	"github.com/docker/infrakit/spi/group"
)

type client struct {
	c plugin.Callable
}

type groupServer struct {
	plugin group.Plugin
}

// PluginClient returns an instance of the Plugin
func PluginClient(c plugin.Callable) group.Plugin {
	return &client{c: c}
}

// PluginServer returns an instance of the Plugin
func PluginServer(p group.Plugin) http.Handler {

	g := &groupServer{plugin: p}
	return server.BuildHandler([]func() (plugin.Endpoint, plugin.Handler){
		g.watchGroup,
		g.unwatchGroup,
		g.inspectGroup,
		g.describeUpdate,
		g.updateGroup,
		g.stopUpdate,
		g.destroyGroup,
	})
}

func (c *client) WatchGroup(grp group.Spec) error {
	_, err := c.c.Call(&util.HTTPEndpoint{Method: "POST", Path: "/Group.Watch"}, grp, nil)
	return err
}

func (s *groupServer) watchGroup() (plugin.Endpoint, plugin.Handler) {
	return &util.HTTPEndpoint{Method: "POST", Path: "/Group.Watch"},

		func(vars map[string]string, body io.Reader) (result interface{}, err error) {
			config := group.Spec{}
			err = json.NewDecoder(body).Decode(&config)
			if err != nil {
				return nil, err
			}
			err = s.plugin.WatchGroup(config)
			return nil, err
		}
}

func (c *client) UnwatchGroup(id group.ID) error {
	_, err := c.c.Call(&util.HTTPEndpoint{Method: "POST", Path: fmt.Sprintf("/Group.Unwatch/%v", id)}, nil, nil)
	return err
}

func (s *groupServer) unwatchGroup() (plugin.Endpoint, plugin.Handler) {
	return &util.HTTPEndpoint{Method: "POST", Path: "/Group.Unwatch/{id}"},

		func(vars map[string]string, body io.Reader) (result interface{}, err error) {
			err = s.plugin.UnwatchGroup(group.ID(vars["id"]))
			return nil, err
		}
}

func (c *client) InspectGroup(id group.ID) (group.Description, error) {
	description := group.Description{}
	_, err := c.c.Call(&util.HTTPEndpoint{Method: "POST", Path: fmt.Sprintf("/Group.Inspect/%v", id)}, nil, &description)
	return description, err
}

func (s *groupServer) inspectGroup() (plugin.Endpoint, plugin.Handler) {
	return &util.HTTPEndpoint{Method: "POST", Path: "/Group.Inspect/{id}"},

		func(vars map[string]string, body io.Reader) (result interface{}, err error) {
			return s.plugin.InspectGroup(group.ID(vars["id"]))
		}
}

func (c *client) DescribeUpdate(updated group.Spec) (string, error) {
	envelope := map[string]string{}
	_, err := c.c.Call(&util.HTTPEndpoint{Method: "POST", Path: "/Group.DescribeUpdate"}, updated, &envelope)
	return envelope["message"], err
}

func (s *groupServer) describeUpdate() (plugin.Endpoint, plugin.Handler) {
	return &util.HTTPEndpoint{Method: "POST", Path: "/Group.DescribeUpdate"},

		func(vars map[string]string, body io.Reader) (result interface{}, err error) {
			updated := group.Spec{}
			err = json.NewDecoder(body).Decode(&updated)
			if err != nil {
				return nil, err
			}
			message, err := s.plugin.DescribeUpdate(updated)
			if err != nil {
				return nil, err
			}
			// Use a wrapper
			return map[string]string{
				"message": message,
			}, nil
		}
}

func (c *client) UpdateGroup(updated group.Spec) error {
	_, err := c.c.Call(&util.HTTPEndpoint{Method: "POST", Path: "/Group.Update"}, updated, nil)
	return err
}

func (s *groupServer) updateGroup() (plugin.Endpoint, plugin.Handler) {
	return &util.HTTPEndpoint{Method: "POST", Path: "/Group.Update"},

		func(vars map[string]string, body io.Reader) (result interface{}, err error) {
			updated := group.Spec{}
			err = json.NewDecoder(body).Decode(&updated)
			if err != nil {
				return nil, err
			}
			err = s.plugin.UpdateGroup(updated)
			return nil, err
		}
}

func (c *client) StopUpdate(id group.ID) error {
	_, err := c.c.Call(&util.HTTPEndpoint{Method: "POST", Path: fmt.Sprintf("/Group.StopUpdate/%v", id)}, nil, nil)
	return err
}

func (s *groupServer) stopUpdate() (plugin.Endpoint, plugin.Handler) {
	return &util.HTTPEndpoint{Method: "POST", Path: "/Group.StopUpdate/{id}"},

		func(vars map[string]string, body io.Reader) (result interface{}, err error) {
			err = s.plugin.StopUpdate(group.ID(vars["id"]))
			return nil, err
		}
}

func (c *client) DestroyGroup(id group.ID) error {
	_, err := c.c.Call(&util.HTTPEndpoint{Method: "POST", Path: fmt.Sprintf("/Group.Destroy/%v", id)}, nil, nil)
	return err
}

func (s *groupServer) destroyGroup() (plugin.Endpoint, plugin.Handler) {
	return &util.HTTPEndpoint{Method: "POST", Path: "/Group.Destroy/{id}"},

		func(vars map[string]string, body io.Reader) (result interface{}, err error) {
			err = s.plugin.DestroyGroup(group.ID(vars["id"]))
			return nil, err
		}
}
