package build

// BuildOpts options that control the linuxkit build process
type BuildOpts struct {
	Pull             bool
	BuilderType      string
	DecompressKernel bool
	CacheDir         string
	DockerCache      bool
	// DockerContext overrides the Docker context for cache reads and helpers.
	DockerContext string
	Arch          string
	SbomGenerator *SbomGenerator
	InputTar      string
}
