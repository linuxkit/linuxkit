package client

import (
	"net/url"
	"strconv"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"golang.org/x/net/context"
)

// ServiceUpdate updates a Service.
func (cli *Client) ServiceUpdate(ctx context.Context, serviceID string, version swarm.Version, service swarm.ServiceSpec, options types.ServiceUpdateOptions) error {
	var (
		headers map[string][]string
		query   = url.Values{}
	)

	if options.EncodedRegistryAuth != "" {
		headers = map[string][]string{
			"X-Registry-Auth": {options.EncodedRegistryAuth},
		}
	}

	if options.RegistryAuthFrom != "" {
		query.Set("registryAuthFrom", options.RegistryAuthFrom)
	}

	query.Set("version", strconv.FormatUint(version.Index, 10))

	resp, err := cli.post(ctx, "/services/"+serviceID+"/update", query, service, headers)
	ensureReaderClosed(resp)
	return err
}
