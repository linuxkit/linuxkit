package spec

type RegistryAuth struct {
	Username      string
	Password      string
	RegistryToken string // base64 encoded auth token
}

type ImageBuildOptions struct {
	Labels        map[string]string
	BuildArgs     map[string]*string
	NetworkMode   string
	Dockerfile    string
	SSH           []string
	RegistryAuths map[string]RegistryAuth
}
