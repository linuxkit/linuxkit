package dockercontext

import (
	"context"
	"net"

	"github.com/docker/cli/cli/connhelper/commandconn"
	buildkit "github.com/moby/buildkit/client"
)

// BuildKitClient pins Docker's context and config directory for the lifetime of
// the connection, including reconnects. The upstream docker-container helper
// only pins the context name and otherwise inherits the config directory.
func (s Selection) BuildKitClient(ctx context.Context, container string) (*buildkit.Client, error) {
	return buildkit.New(ctx, s.BuilderURL(container), buildkit.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
		return s.dialBuilder(ctx, container)
	}))
}

func (s Selection) dialBuilder(ctx context.Context, container string) (net.Conn, error) {
	// commandconn manages the subprocess lifetime through net.Conn.Close, rather
	// than killing it when the initial dial context finishes. It retains stderr in
	// connection errors, including failures of docker exec or buildctl dial-stdio.
	return commandconn.New(ctx, "docker", s.Args("exec", "-i", container, "buildctl", "dial-stdio")...)
}
