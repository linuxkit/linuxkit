package vpnkit

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
)

func (d *Dialer) connectTransport() (io.ReadWriteCloser, error) {
	path := d.HyperkitConnectPath
	if path == "" {
		// On Mac assume Docker Desktop
		path = filepath.Join(os.Getenv("HOME"), "Library", "Containers", "com.docker.docker", "Data", "vms", "0", "connect")
	}
	port := d.Port
	if port == 0 {
		port = DefaultVsockPort
	}
	conn, err := net.Dial("unix", path)
	if err != nil {
		return nil, err
	}
	if _, err := io.WriteString(conn, fmt.Sprintf("00000003.%08x\n", port)); err != nil {
		return nil, err
	}
	return conn, nil
}
