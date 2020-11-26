package dhclient

import (
	"io/ioutil"
	"net"
	"testing"
	"time"

	"github.com/google/gopacket/layers"

	"github.com/stretchr/testify/assert"
)

func TestParseIPs(t *testing.T) {
	assert := assert.New(t)

	data := []byte{143, 209, 4, 1, 143, 209, 5, 1}
	ips := parseIPs(data)
	assert.Len(ips, 2)
	assert.Equal(net.IP{143, 209, 4, 1}, ips[0])
	assert.Equal(net.IP{143, 209, 5, 1}, ips[1])

	// not enough bytes
	assert.Len(parseIPs([]byte{143, 209, 4}), 0)
}

func TestParseResponse(t *testing.T) {
	assert := assert.New(t)

	data, err := ioutil.ReadFile("testdata/offer.packet")
	assert.NoError(err)

	packet := parsePacket(data)
	assert.NotNil(packet)

	msgType, lease := newLease(packet)
	assert.Equal(layers.DHCPMsgTypeOffer, msgType)
	assert.Equal(net.IP{192, 168, 9, 131}, lease.FixedAddress)
	assert.Len(lease.Router, 1)
	assert.Equal(net.IP{192, 168, 8, 1}, lease.Router[0])
	assert.Len(lease.DNS, 1)
	assert.Equal(net.IP{192, 168, 8, 1}, lease.DNS[0])
	assert.Equal(net.IPMask{255, 255, 252, 0}, lease.Netmask)
	assert.EqualValues(1406, lease.MTU)
	assert.Len(lease.OtherOptions, 0)

	// check timestamps
	assert.False(lease.Bound.IsZero())
	assert.Equal(1800, int(lease.Renew.Sub(lease.Bound)/time.Second))
	assert.Equal(3150, int(lease.Rebind.Sub(lease.Bound)/time.Second))
	assert.Equal(3600, int(lease.Expire.Sub(lease.Bound)/time.Second))
}
