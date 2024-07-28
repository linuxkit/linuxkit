package spec

type ImageBuildOptions struct {
	Labels      map[string]string
	BuildArgs   map[string]*string
	NetworkMode string
	Dockerfile  string
	SSH         []string
}
