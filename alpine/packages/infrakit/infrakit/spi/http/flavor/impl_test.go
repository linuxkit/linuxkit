package flavor

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/docker/infrakit/plugin/group/types"
	plugin_client "github.com/docker/infrakit/plugin/util/client"
	"github.com/docker/infrakit/plugin/util/server"
	"github.com/docker/infrakit/spi/flavor"
	"github.com/docker/infrakit/spi/instance"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"path"
)

var allocation = types.AllocationMethod{}

func tempSocket() string {
	dir, err := ioutil.TempDir("", "infrakit-test-")
	if err != nil {
		panic(err)
	}

	return path.Join(dir, "flavor-impl-test")
}

type testPlugin struct {
	DoValidate func(flavorProperties json.RawMessage, allocation types.AllocationMethod) error
	DoPrepare  func(
		flavorProperties json.RawMessage,
		spec instance.Spec,
		allocation types.AllocationMethod) (instance.Spec, error)
	DoHealthy func(flavorProperties json.RawMessage, inst instance.Description) (flavor.Health, error)
}

func (t *testPlugin) Validate(flavorProperties json.RawMessage, allocation types.AllocationMethod) error {
	return t.DoValidate(flavorProperties, allocation)
}
func (t *testPlugin) Prepare(
	flavorProperties json.RawMessage,
	spec instance.Spec,
	allocation types.AllocationMethod) (instance.Spec, error) {

	return t.DoPrepare(flavorProperties, spec, allocation)
}
func (t *testPlugin) Healthy(flavorProperties json.RawMessage, inst instance.Description) (flavor.Health, error) {
	return t.DoHealthy(flavorProperties, inst)
}

func TestFlavorPluginValidate(t *testing.T) {
	socketPath := tempSocket()

	inputFlavorPropertiesActual := make(chan json.RawMessage, 1)
	inputFlavorProperties := json.RawMessage([]byte(`{"flavor":"zookeeper","role":"leader"}`))

	stop, _, err := server.StartPluginAtPath(socketPath, PluginServer(&testPlugin{
		DoValidate: func(flavorProperties json.RawMessage, allocation types.AllocationMethod) error {
			inputFlavorPropertiesActual <- flavorProperties
			return nil
		},
	}))
	require.NoError(t, err)

	require.NoError(t, PluginClient(plugin_client.New(socketPath)).Validate(inputFlavorProperties, allocation))

	close(stop)

	require.Equal(t, inputFlavorProperties, <-inputFlavorPropertiesActual)
}

func TestFlavorPluginValidateError(t *testing.T) {
	socketPath := tempSocket()

	inputFlavorPropertiesActual := make(chan json.RawMessage, 1)
	inputFlavorProperties := json.RawMessage([]byte(`{"flavor":"zookeeper","role":"leader"}`))

	stop, _, err := server.StartPluginAtPath(socketPath, PluginServer(&testPlugin{
		DoValidate: func(flavorProperties json.RawMessage, allocation types.AllocationMethod) error {
			inputFlavorPropertiesActual <- flavorProperties
			return errors.New("something-went-wrong")
		},
	}))
	require.NoError(t, err)

	err = PluginClient(plugin_client.New(socketPath)).Validate(inputFlavorProperties, allocation)
	require.Error(t, err)
	require.Equal(t, "something-went-wrong", err.Error())

	close(stop)
	require.Equal(t, inputFlavorProperties, <-inputFlavorPropertiesActual)
}

func TestFlavorPluginPrepare(t *testing.T) {
	socketPath := tempSocket()

	inputFlavorPropertiesActual := make(chan json.RawMessage, 1)
	inputFlavorProperties := json.RawMessage([]byte(`{"flavor":"zookeeper","role":"leader"}`))
	inputInstanceSpecActual := make(chan instance.Spec, 1)
	inputInstanceSpec := instance.Spec{
		Properties: &inputFlavorProperties,
		Tags:       map[string]string{"foo": "bar"},
	}

	stop, _, err := server.StartPluginAtPath(socketPath, PluginServer(&testPlugin{
		DoPrepare: func(
			flavorProperties json.RawMessage,
			instanceSpec instance.Spec,
			allocation types.AllocationMethod) (instance.Spec, error) {

			inputFlavorPropertiesActual <- flavorProperties
			inputInstanceSpecActual <- instanceSpec

			return instanceSpec, nil
		},
	}))
	require.NoError(t, err)

	spec, err := PluginClient(plugin_client.New(socketPath)).Prepare(
		inputFlavorProperties,
		inputInstanceSpec,
		allocation)
	require.NoError(t, err)
	require.Equal(t, inputInstanceSpec, spec)

	close(stop)

	require.Equal(t, inputFlavorProperties, <-inputFlavorPropertiesActual)
	require.Equal(t, inputInstanceSpec, <-inputInstanceSpecActual)
}

func TestFlavorPluginPrepareError(t *testing.T) {
	socketPath := tempSocket()

	inputFlavorPropertiesActual := make(chan json.RawMessage, 1)
	inputFlavorProperties := json.RawMessage([]byte(`{"flavor":"zookeeper","role":"leader"}`))
	inputInstanceSpecActual := make(chan instance.Spec, 1)
	inputInstanceSpec := instance.Spec{
		Properties: &inputFlavorProperties,
		Tags:       map[string]string{"foo": "bar"},
	}

	stop, _, err := server.StartPluginAtPath(socketPath, PluginServer(&testPlugin{
		DoPrepare: func(
			flavorProperties json.RawMessage,
			instanceSpec instance.Spec,
			allocation types.AllocationMethod) (instance.Spec, error) {

			inputFlavorPropertiesActual <- flavorProperties
			inputInstanceSpecActual <- instanceSpec

			return instanceSpec, errors.New("bad-thing-happened")
		},
	}))
	require.NoError(t, err)

	_, err = PluginClient(plugin_client.New(socketPath)).Prepare(
		inputFlavorProperties,
		inputInstanceSpec,
		allocation)
	require.Error(t, err)
	require.Equal(t, "bad-thing-happened", err.Error())

	close(stop)

	require.Equal(t, inputFlavorProperties, <-inputFlavorPropertiesActual)
	require.Equal(t, inputInstanceSpec, <-inputInstanceSpecActual)
}

func TestFlavorPluginHealthy(t *testing.T) {
	socketPath := tempSocket()

	inputPropertiesActual := make(chan json.RawMessage, 1)
	inputInstanceActual := make(chan instance.Description, 1)
	inputProperties := json.RawMessage("{}")
	inputInstance := instance.Description{
		ID:   instance.ID("foo"),
		Tags: map[string]string{"foo": "bar"},
	}
	stop, _, err := server.StartPluginAtPath(socketPath, PluginServer(&testPlugin{
		DoHealthy: func(properties json.RawMessage, inst instance.Description) (flavor.Health, error) {
			inputPropertiesActual <- properties
			inputInstanceActual <- inst
			return flavor.Healthy, nil
		},
	}))
	require.NoError(t, err)

	health, err := PluginClient(plugin_client.New(socketPath)).Healthy(inputProperties, inputInstance)
	require.NoError(t, err)
	require.Equal(t, flavor.Healthy, health)

	require.Equal(t, inputProperties, <-inputPropertiesActual)
	require.Equal(t, inputInstance, <-inputInstanceActual)
	close(stop)
}

func TestFlavorPluginHealthyError(t *testing.T) {
	socketPath := tempSocket()

	inputPropertiesActual := make(chan json.RawMessage, 1)
	inputInstanceActual := make(chan instance.Description, 1)
	inputProperties := json.RawMessage("{}")
	inputInstance := instance.Description{
		ID:   instance.ID("foo"),
		Tags: map[string]string{"foo": "bar"},
	}
	stop, _, err := server.StartPluginAtPath(socketPath, PluginServer(&testPlugin{
		DoHealthy: func(flavorProperties json.RawMessage, inst instance.Description) (flavor.Health, error) {
			inputPropertiesActual <- flavorProperties
			inputInstanceActual <- inst
			return flavor.Unknown, errors.New("oh-noes")
		},
	}))
	require.NoError(t, err)

	_, err = PluginClient(plugin_client.New(socketPath)).Healthy(inputProperties, inputInstance)
	require.Error(t, err)
	require.Equal(t, "oh-noes", err.Error())

	require.Equal(t, inputProperties, <-inputPropertiesActual)
	require.Equal(t, inputInstance, <-inputInstanceActual)
	close(stop)
}
