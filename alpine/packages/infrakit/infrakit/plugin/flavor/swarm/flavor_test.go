package swarm

import (
	"encoding/json"
	"fmt"
	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	mock_client "github.com/docker/infrakit/mock/docker/docker/client"
	"github.com/docker/infrakit/plugin/group/types"
	"github.com/docker/infrakit/spi/flavor"
	"github.com/docker/infrakit/spi/instance"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestValidate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	swarmFlavor := NewSwarmFlavor(mock_client.NewMockAPIClient(ctrl))

	require.NoError(t, swarmFlavor.Validate(json.RawMessage(`{"type": "worker"}`), types.AllocationMethod{Size: 5}))
	require.NoError(t, swarmFlavor.Validate(
		json.RawMessage(`{"type": "manager"}`),
		types.AllocationMethod{LogicalIDs: []instance.LogicalID{"127.0.0.1"}}))
	require.Error(t, swarmFlavor.Validate(json.RawMessage(`{"type": "other"}`), types.AllocationMethod{Size: 5}))
}

func TestAssociation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	client := mock_client.NewMockAPIClient(ctrl)

	helper := NewSwarmFlavor(client)

	swarmInfo := swarm.Swarm{
		ClusterInfo: swarm.ClusterInfo{ID: "ClusterUUID"},
		JoinTokens: swarm.JoinTokens{
			Manager: "ManagerToken",
			Worker:  "WorkerToken",
		},
	}
	client.EXPECT().SwarmInspect(gomock.Any()).Return(swarmInfo, nil)

	client.EXPECT().Info(gomock.Any()).Return(docker_types.Info{Swarm: swarm.Info{NodeID: "my-node-id"}}, nil)

	nodeInfo := swarm.Node{ManagerStatus: &swarm.ManagerStatus{Addr: "1.2.3.4"}}
	client.EXPECT().NodeInspectWithRaw(gomock.Any(), "my-node-id").Return(nodeInfo, nil, nil)

	details, err := helper.Prepare(
		json.RawMessage(`{"type": "worker"}`),
		instance.Spec{Tags: map[string]string{"a": "b"}},
		types.AllocationMethod{Size: 5})
	require.NoError(t, err)
	require.Equal(t, "b", details.Tags["a"])
	associationID := details.Tags[associationTag]
	require.NotEqual(t, "", associationID)

	// Perform a rudimentary check to ensure that the expected fields are in the InitScript, without having any
	// other knowledge about the script structure.
	require.Contains(t, details.Init, associationID)
	require.Contains(t, details.Init, swarmInfo.JoinTokens.Worker)
	require.Contains(t, details.Init, nodeInfo.ManagerStatus.Addr)

	// An instance with no association information is considered unhealthy.
	health, err := helper.Healthy(json.RawMessage("{}"), instance.Description{})
	require.NoError(t, err)
	require.Equal(t, flavor.Unhealthy, health)

	filter, err := filters.FromParam(fmt.Sprintf(`{"label": {"%s=%s": true}}`, associationTag, associationID))
	require.NoError(t, err)
	client.EXPECT().NodeList(gomock.Any(), docker_types.NodeListOptions{Filter: filter}).Return(
		[]swarm.Node{
			{},
		}, nil)
	health, err = helper.Healthy(
		json.RawMessage("{}"),
		instance.Description{Tags: map[string]string{associationTag: associationID}})
	require.NoError(t, err)
	require.Equal(t, flavor.Healthy, health)
}
