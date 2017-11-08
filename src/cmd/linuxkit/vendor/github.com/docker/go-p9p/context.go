package p9p

import (
	"context"
)

type contextKey string

const (
	versionKey contextKey = "9p.version"
)

func withVersion(ctx context.Context, version string) context.Context {
	return context.WithValue(ctx, versionKey, version)
}

// GetVersion returns the protocol version from the context. If the version is
// not known, an empty string is returned. This is typically set on the
// context passed into function calls in a server implementation.
func GetVersion(ctx context.Context) string {
	v, ok := ctx.Value(versionKey).(string)
	if !ok {
		return ""
	}
	return v
}
