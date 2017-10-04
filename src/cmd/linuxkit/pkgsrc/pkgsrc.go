package pkgsrc

import (
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
	"strings"
)

// Containers fields settable in the build.yml
type pkgInfo struct {
	Image   string   `yaml:"image"`
	Org     string   `yaml:"org"`
	Arches  []string `yaml:"arches"`
	GitRepo string   `yaml:"gitrepo"` // ??
	Network bool     `yaml:"network"`
	Trust   bool     `yaml:"trust"`
	Cache   bool     `yaml:"cache"`
}

// PkgSrc encapsulates information about a package's source
type PkgSrc struct {
	// These correspond to pkgInfo fields
	image   string
	org     string
	arches  []string
	gitRepo string
	network bool
	trust   bool
	cache   bool

	// Internal state
	pkgPath    string
	hash       string
	dirty      bool
	commitHash string
}

// NewFromCLI creates a PkgSrc from a set of CLI arguments. Calls fs.Parse()
func NewFromCLI(fs *flag.FlagSet, args []string) (PkgSrc, error) {
	// Defaults
	pi := pkgInfo{
		Org:     "linuxkit",
		Arches:  []string{"amd64", "arm64"},
		GitRepo: "https://github.com/linuxkit/linuxkit",
		Network: false,
		Trust:   true,
		Cache:   true,
	}

	// TODO(ijc) look for "$(git rev-parse --show-toplevel)/.build-defaults.yml"?

	// Ideally want to look at every directory from root to `pkg`
	// for this file but might be tricky to arrange ordering-wise.

	// These override fields in pi below
	argCache := fs.Bool("cache", pi.Cache, "Disable build cache")
	argNetwork := fs.Bool("network", pi.Network, "Allow network use during build")
	argOrg := fs.String("org", pi.Org, "Override the hub org")
	argTrust := fs.Bool("trust", pi.Trust, "Disable content trust")

	// Other arguments
	var buildYML, hash, hashCommit, hashPath string
	fs.StringVar(&buildYML, "build-yml", "build.yml", "Override the name of the yml file")
	fs.StringVar(&hash, "hash", "", "Override the image hash (default is to query git for the package's tree-sh)")
	fs.StringVar(&hashCommit, "hash-commit", "HEAD", "Override the git commit to use for the hash")
	fs.StringVar(&hashPath, "hash-path", "", "Override the directory to use for the image hash, must be a parent of the package dir (default is to use the package dir)")

	fs.Parse(args)

	if fs.NArg() < 1 {
		return PkgSrc{}, fmt.Errorf("A pkg directory is required")
	}
	if fs.NArg() > 1 {
		return PkgSrc{}, fmt.Errorf("Unknown extra arguments given: %s", fs.Args()[1:])
	}

	pkg := fs.Arg(0)
	pkgPath, err := filepath.Abs(pkg)
	if err != nil {
		return PkgSrc{}, err
	}

	if hashPath == "" {
		hashPath = pkgPath
	} else {
		hashPath, err = filepath.Abs(hashPath)
		if err != nil {
			return PkgSrc{}, err
		}

		if !strings.HasPrefix(pkgPath, hashPath) {
			return PkgSrc{}, fmt.Errorf("Hash path is not a prefix of the package path")
		}

		// TODO(ijc) pkgPath and hashPath really ought to be in the same git tree too...
	}

	b, err := ioutil.ReadFile(filepath.Join(pkgPath, buildYML))
	if err != nil {
		return PkgSrc{}, err
	}
	if err := yaml.Unmarshal(b, &pi); err != nil {
		return PkgSrc{}, err
	}

	if pi.Image == "" {
		return PkgSrc{}, fmt.Errorf("Image field is required")
	}

	// Go's flag package provides no way to see if a flag was set
	// apart from Visit which iterates over only those which were
	// set.
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "cache":
			pi.Cache = *argCache
		case "network":
			pi.Network = *argNetwork
		case "org":
			pi.Org = *argOrg
		case "trust":
			pi.Trust = *argTrust
		}
	})

	if hash == "" {
		if hash, err = gitTreeHash(hashPath, hashCommit); err != nil {
			return PkgSrc{}, err
		}
	}

	dirty, err := gitIsDirty(hashPath, hashCommit)
	if err != nil {
		return PkgSrc{}, err
	}

	return PkgSrc{
		image:      pi.Image,
		org:        pi.Org,
		hash:       hash,
		commitHash: hashCommit,
		arches:     pi.Arches,
		gitRepo:    pi.GitRepo,
		network:    pi.Network,
		trust:      pi.Trust,
		cache:      pi.Cache,
		dirty:      dirty,
		pkgPath:    pkgPath,
	}, nil
}

// Hash returns the hash of the package
func (ps PkgSrc) Hash() string {
	return ps.hash
}

// ReleaseTag returns the tag to use for a particular release of the package
func (ps PkgSrc) ReleaseTag(release string) (string, error) {
	if release == "" {
		return "", fmt.Errorf("A release tag is required")
	}
	if ps.dirty {
		return "", fmt.Errorf("Cannot release a dirty package")
	}
	tag := ps.org + "/" + ps.image + ":" + release
	return tag, nil
}

// Tag returns the tag to use for the package
func (ps PkgSrc) Tag() string {
	tag := ps.org + "/" + ps.image + ":" + ps.hash
	if ps.dirty {
		tag += "-dirty"
	}
	return tag
}

func (ps PkgSrc) archSupported(want string) bool {
	for _, supp := range ps.arches {
		if supp == want {
			return true
		}
	}
	return false
}

func (ps PkgSrc) cleanForBuild() error {
	if ps.commitHash != "HEAD" {
		return fmt.Errorf("Cannot build from commit hash != HEAD")
	}
	return nil
}
