// Package vsock provides bindings to the hyperkit based
// implementation on macOS hosts.  virtio Sockets are exposed as named
// pipes on macOS. Two modes are supported (to be set with
// SockerMode()):
// - Hyperkit mode: The package needs to be initialised with the path
//   to where the named pipe was created.
// - Docker for Mac mode: This is a shortcut which hard codes the
//   location of the named pipe.
package vsock

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
)

var (
	socketPath  string
	connectPath string
	socketFmt   string
)

// SocketMode initialises the bindings to either raw hyperkit mode
// ("hyperkit:/path") or Docker for Mac mode ("docker"). This function
// must be called before using the vsock bindings.
func SocketMode(socketMode string) {
	if strings.HasPrefix(socketMode, "hyperkit:") {
		socketPath = socketMode[len("hyperkit:"):]
		connectPath = filepath.Join(socketPath, "connect")
		socketFmt = "%08x.%08x"
	} else if socketMode == "docker" {
		socketPath = filepath.Join(os.Getenv("HOME"), "/Library/Containers/com.docker.docker/Data")
		connectPath = filepath.Join(socketPath, "@connect")
		socketFmt = "*%08x.%08x"
	} else {
		log.Fatalln("Unknown socket mode: ", socketMode)
	}
}

// Dial creates a connection to the VM with the given client ID and port
func Dial(cid, port uint32) (Conn, error) {
	c, err := net.DialUnix("unix", nil, &net.UnixAddr{connectPath, "unix"})
	if err != nil {
		return c, err
	}
	if _, err := fmt.Fprintf(c, "%08x.%08x\n", cid, port); err != nil {
		return c, fmt.Errorf("Failed to write dest (%08x.%08x) to %s", cid, port, connectPath)
	}
	return c, nil
}

// Listen creates a listener for a specifc vsock.
func Listen(cid, port uint32) (net.Listener, error) {
	sock := filepath.Join(socketPath, fmt.Sprintf(socketFmt, cid, port))
	if err := os.Remove(sock); err != nil && !os.IsNotExist(err) {
		log.Fatalln("Listen(): Remove:", err)
		return nil, err
	}

	return net.ListenUnix("unix", &net.UnixAddr{sock, "unix"})
}
