package main

import (
	"encoding/json"
	"errors"
	mock_flavor "github.com/docker/infrakit/mock/spi/flavor"
	"github.com/docker/infrakit/plugin/group"
	"github.com/docker/infrakit/plugin/group/types"
	"github.com/docker/infrakit/spi/flavor"
	"github.com/docker/infrakit/spi/instance"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"testing"
)

func jsonPtr(v string) *json.RawMessage {
	j := json.RawMessage(v)
	return &j
}

func logicalID(v string) *instance.LogicalID {
	id := instance.LogicalID(v)
	return &id
}

var inst = instance.Spec{
	Properties:  jsonPtr("{}"),
	Tags:        map[string]string{},
	Init:        "",
	LogicalID:   logicalID("id"),
	Attachments: []instance.Attachment{"att1"},
}

func pluginLookup(plugins map[string]flavor.Plugin) group.FlavorPluginLookup {
	return func(key string) (flavor.Plugin, error) {
		plugin, has := plugins[key]
		if has {
			return plugin, nil
		}
		return nil, errors.New("Plugin doesn't exist")
	}
}

func TestMergeBehavior(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	a := mock_flavor.NewMockPlugin(ctrl)
	b := mock_flavor.NewMockPlugin(ctrl)

	plugins := map[string]flavor.Plugin{"a": a, "b": b}

	combo := NewPlugin(pluginLookup(plugins))

	flavorProperties := json.RawMessage(`{
	  "Flavors": [
	    {
	      "Plugin": "a",
	      "Properties": {"a": "1"}
	    },
	    {
	      "Plugin": "b",
	      "Properties": {"b": "2"}
	    }
	  ]
	}`)

	allocation := types.AllocationMethod{Size: 1}

	a.EXPECT().Prepare(json.RawMessage(`{"a": "1"}`), inst, allocation).Return(instance.Spec{
		Properties:  inst.Properties,
		Tags:        map[string]string{"a": "1", "c": "4"},
		Init:        "init data a",
		LogicalID:   inst.LogicalID,
		Attachments: []instance.Attachment{"a"},
	}, nil)

	b.EXPECT().Prepare(json.RawMessage(`{"b": "2"}`), inst, allocation).Return(instance.Spec{
		Properties:  inst.Properties,
		Tags:        map[string]string{"b": "2", "c": "5"},
		Init:        "init data b",
		LogicalID:   inst.LogicalID,
		Attachments: []instance.Attachment{"b"},
	}, nil)

	result, err := combo.Prepare(flavorProperties, inst, types.AllocationMethod{Size: 1})
	require.NoError(t, err)

	expected := instance.Spec{
		Properties:  inst.Properties,
		Tags:        map[string]string{"a": "1", "b": "2", "c": "5"},
		Init:        "init data a\ninit data b",
		LogicalID:   inst.LogicalID,
		Attachments: []instance.Attachment{"att1", "a", "b"},
	}
	require.Equal(t, expected, result)
}
