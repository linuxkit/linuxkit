package zookeeper

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/infrakit/plugin/group/types"
	"github.com/docker/infrakit/spi/flavor"
	"github.com/docker/infrakit/spi/instance"
	"strings"
	"text/template"
)

const (
	roleMember = "member"
)

// NewPlugin creates a ProvisionHelper that creates ZooKeeper nodes.
func NewPlugin() flavor.Plugin {
	return &zkFlavor{}
}

type zkFlavor struct {
}

type spec struct {
	Type      string
	UseDocker bool
}

func parseSpec(flavorProperties json.RawMessage) (spec, error) {
	s := spec{UseDocker: false}
	err := json.Unmarshal(flavorProperties, &s)
	return s, err
}

func (s zkFlavor) Validate(flavorProperties json.RawMessage, allocation types.AllocationMethod) error {
	properties, err := parseSpec(flavorProperties)
	if err != nil {
		return err
	}

	switch properties.Type {
	case roleMember:
		if len(allocation.LogicalIDs) == 0 {
			return errors.New("IP addresses are required as LogicalIDs for a ZooKeeper ensemble")
		}

		return nil
	default:
		return errors.New("Unrecognized node Type")
	}
}

const (
	// initScript is used to generate node boot scripts.
	initScript = `#!/bin/sh

apt-get update
apt-get install -y openjdk-8-jdk-headless
apt-get install -y zookeeperd

cat << EOF > /etc/zookeeper/conf/zoo.cfg
tickTime=2000
dataDir=/var/lib/zookeeper
clientPort=2181
initLimit=5
syncLimit=2
{{range $i, $ip := .Servers}}
server.{{inc $i}}={{$ip}}:2888:3888
{{end}}
EOF

echo '{{.MyID}}' > /var/lib/zookeeper/myid

systemctl reload-or-restart zookeeper
`

	// TODO(wfarner): Running via docker doesn't work yet - nodes can't connect to each other.
	// TODO(wfarner): Persist data directory.
	initScriptDocker = `#!/bin/sh

docker run \
  -p 2181:2181 \
  -p 2888:2888 \
  -p 3888:3888 \
  -e ZOO_MY_ID='{{.MyID}}' \
  -e ZOO_SERVERS='{{.ServersList}}' \
  --name zk \
  --restart always \
  -d zookeeper
`
)

func generateInitScript(useDocker bool, servers []instance.LogicalID, id instance.LogicalID) string {
	buffer := bytes.Buffer{}

	myID := -1
	for i, server := range servers {
		if server == id {
			myID = i + 1
			break
		}
	}
	if myID == -1 {
		panic(fmt.Sprintf("Logical ID %s is not in available IDs %s", id, servers))
	}

	var templateText string
	params := map[string]interface{}{
		"MyID": myID,
	}
	if useDocker {
		templateText = initScriptDocker
		serverStrings := []string{}
		for i, server := range servers {
			if server == id {
				// ZooKeeper uses the matching server entry to determine the bind address for leader
				// election.  We replace the 'self' entry with 0.0.0.0 to avoid trying to bind to the
				// host's public IP, which will fail.
				server = "0.0.0.0"
			}

			serverStrings = append(serverStrings, fmt.Sprintf("server.%d=%s:2888:3888", i+1, server))
		}

		params["ServersList"] = strings.Join(serverStrings, " ")
	} else {
		templateText = initScript
		params["Servers"] = servers
	}

	funcs := template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
	}

	templ := template.Must(template.New("").Funcs(funcs).Parse(templateText))

	if err := templ.Execute(&buffer, params); err != nil {
		panic(err)
	}
	return buffer.String()
}

// Healthy determines whether an instance is healthy.
func (s zkFlavor) Healthy(flavorProperties json.RawMessage, inst instance.Description) (flavor.Health, error) {
	// TODO(wfarner): Implement.
	return flavor.Healthy, nil
}

func (s zkFlavor) Prepare(
	flavorProperties json.RawMessage,
	spec instance.Spec,
	allocation types.AllocationMethod) (instance.Spec, error) {

	properties, err := parseSpec(flavorProperties)
	if err != nil {
		return spec, err
	}

	switch properties.Type {
	case roleMember:
		if spec.LogicalID == nil {
			return spec, errors.New("Manager nodes require an assigned logical ID")
		}

		spec.Init = generateInitScript(properties.UseDocker, allocation.LogicalIDs, *spec.LogicalID)

	default:
		return spec, errors.New("Unsupported role type")
	}

	return spec, nil
}
