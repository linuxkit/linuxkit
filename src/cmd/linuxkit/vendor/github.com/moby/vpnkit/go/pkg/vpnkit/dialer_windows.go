package vpnkit

import (
	"errors"
	"fmt"
	"io"

	"github.com/linuxkit/virtsock/pkg/hvsock"
)

func (d *Dialer) connectTransport() (io.ReadWriteCloser, error) {
	if d.HyperVVMID == "" {
		return nil, errors.New("Hyper-V VMID must be provided")
	}
	port := d.Port
	if port == 0 {
		port = DefaultVsockPort
	}
	guid := fmt.Sprintf("%08x-FACB-11E6-BD58-64006A7986D3", port)
	vmid, err := hvsock.GUIDFromString(d.HyperVVMID)
	if err != nil {
		return nil, err
	}
	svc, err := hvsock.GUIDFromString(guid)
	if err != nil {
		return nil, err
	}
	addr := hvsock.Addr{
		VMID:      vmid,
		ServiceID: svc,
	}
	return hvsock.Dial(addr)
}
