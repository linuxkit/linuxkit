package build

// BuildOpts options that control the linuxkit build process
type BuildOpts struct {
	Pull             bool
	BuilderType      string
	DecompressKernel bool
	CacheDir         string
	DockerCache      bool
	Arch             string
	SbomGenerator    *SbomGenerator
	InputTar         string
}
