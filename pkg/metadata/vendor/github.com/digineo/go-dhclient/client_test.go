package dhclient

import (
	"testing"

	"github.com/google/gopacket/layers"
	"github.com/stretchr/testify/assert"
)

func TestAddParamRequest(t *testing.T) {
	assert := assert.New(t)
	client := Client{}
	assert.Len(client.DHCPOptions, 0)

	// Add one option
	client.AddOption(layers.DHCPOptHostname, []byte("example.com"))
	assert.Len(client.DHCPOptions, 1)

	// Add first param request
	client.AddParamRequest(layers.DHCPOptSubnetMask)
	assert.Len(client.DHCPOptions, 2)
	assert.Len(client.DHCPOptions[1].Data, 1)

	// Add second param request
	client.AddParamRequest(layers.DHCPOptRouter)
	assert.Len(client.DHCPOptions[1].Data, 2)

	// Add existing param request
	client.AddParamRequest(layers.DHCPOptRouter)
	assert.Len(client.DHCPOptions[1].Data, 2)

}
