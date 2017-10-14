package pkglib

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
	Image               string   `yaml:"image"`
	Org                 string   `yaml:"org"`
	Arches              []string `yaml:"arches"`
	GitRepo             string   `yaml:"gitrepo"` // ??
	Network             bool     `yaml:"network"`
	DisableContentTrust bool     `yaml:"disable-content-trust"`
	DisableCache        bool     `yaml:"disable-cache"`
}

// Pkg encapsulates information about a package's source
type Pkg struct {
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

// NewFromCLI creates a Pkg from a set of CLI arguments. Calls fs.Parse()
func NewFromCLI(fs *flag.FlagSet, args ...string) (Pkg, error) {
	// Defaults
	pi := pkgInfo{
		Org:                 "linuxkit",
		Arches:              []string{"amd64", "arm64"},
		GitRepo:             "https://github.com/linuxkit/linuxkit",
		Network:             false,
		DisableContentTrust: false,
		DisableCache:        false,
	}

	// TODO(ijc) look for "$(git rev-parse --show-toplevel)/.build-defaults.yml"?

	// Ideally want to look at every directory from root to `pkg`
	// for this file but might be tricky to arrange ordering-wise.

	// These override fields in pi below, bools are in both forms to allow user overrides in either direction
	argDisableCache := fs.Bool("disable-cache", pi.DisableCache, "Disable build cache")
	argEnableCache := fs.Bool("enable-cache", !pi.DisableCache, "Enable build cache")
	argDisableContentTrust := fs.Bool("disable-content-trust", pi.DisableContentTrust, "Enable content trust")
	argEnableContentTrust := fs.Bool("enable-content-trust", !pi.DisableContentTrust, "Enable content trust")
	argNoNetwork := fs.Bool("nonetwork", !pi.Network, "Disallow network use during build")
	argNetwork := fs.Bool("network", pi.Network, "Allow network use during build")

	argOrg := fs.String("org", pi.Org, "Override the hub org")

	// Other arguments
	var buildYML, hash, hashCommit, hashPath string
	var dirty bool
	fs.StringVar(&buildYML, "build-yml", "build.yml", "Override the name of the yml file")
	fs.StringVar(&hash, "hash", "", "Override the image hash (default is to query git for the package's tree-sh)")
	fs.StringVar(&hashCommit, "hash-commit", "HEAD", "Override the git commit to use for the hash")
	fs.StringVar(&hashPath, "hash-path", "", "Override the directory to use for the image hash, must be a parent of the package dir (default is to use the package dir)")
	fs.BoolVar(&dirty, "force-dirty", false, "Force the pkg to be considered dirty")

	fs.Parse(args)

	if fs.NArg() < 1 {
		return Pkg{}, fmt.Errorf("A pkg directory is required")
	}
	if fs.NArg() > 1 {
		return Pkg{}, fmt.Errorf("Unknown extra arguments given: %s", fs.Args()[1:])
	}

	pkg := fs.Arg(0)
	pkgPath, err := filepath.Abs(pkg)
	if err != nil {
		return Pkg{}, err
	}

	if hashPath == "" {
		hashPath = pkgPath
	} else {
		hashPath, err = filepath.Abs(hashPath)
		if err != nil {
			return Pkg{}, err
		}

		if !strings.HasPrefix(pkgPath, hashPath) {
			return Pkg{}, fmt.Errorf("Hash path is not a prefix of the package path")
		}

		// TODO(ijc) pkgPath and hashPath really ought to be in the same git tree too...
	}

	b, err := ioutil.ReadFile(filepath.Join(pkgPath, buildYML))
	if err != nil {
		return Pkg{}, err
	}
	if err := yaml.Unmarshal(b, &pi); err != nil {
		return Pkg{}, err
	}

	if pi.Image == "" {
		return Pkg{}, fmt.Errorf("Image field is required")
	}

	// Go's flag package provides no way to see if a flag was set
	// apart from Visit which iterates over only those which were
	// set.
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "disable-cache":
			pi.DisableCache = *argDisableCache
		case "enable-cache":
			pi.DisableCache = !*argEnableCache
		case "disable-content-trust":
			pi.DisableContentTrust = *argDisableContentTrust
		case "enable-content-trust":
			pi.DisableContentTrust = !*argEnableContentTrust
		case "network":
			pi.Network = *argNetwork
		case "nonetwork":
			pi.Network = !*argNoNetwork
		case "org":
			pi.Org = *argOrg
		}
	})

	gitDirty, err := gitIsDirty(hashPath, hashCommit)
	if err != nil {
		return Pkg{}, err
	}

	dirty = dirty || gitDirty

	if hash == "" {
		if hash, err = gitTreeHash(hashPath, hashCommit); err != nil {
			return Pkg{}, err
		}

		if dirty {
			hash += "-dirty"
		}
	}

	return Pkg{
		image:      pi.Image,
		org:        pi.Org,
		hash:       hash,
		commitHash: hashCommit,
		arches:     pi.Arches,
		gitRepo:    pi.GitRepo,
		network:    pi.Network,
		trust:      !pi.DisableContentTrust,
		cache:      !pi.DisableCache,
		dirty:      dirty,
		pkgPath:    pkgPath,
	}, nil
}

// Hash returns the hash of the package
func (p Pkg) Hash() string {
	return p.hash
}

// ReleaseTag returns the tag to use for a particular release of the package
func (p Pkg) ReleaseTag(release string) (string, error) {
	if release == "" {
		return "", fmt.Errorf("A release tag is required")
	}
	if p.dirty {
		return "", fmt.Errorf("Cannot release a dirty package")
	}
	tag := p.org + "/" + p.image + ":" + release
	return tag, nil
}

// Tag returns the tag to use for the package
func (p Pkg) Tag() string {
	return p.org + "/" + p.image + ":" + p.hash
}

// TrustEnabled returns true if trust is enabled
func (p Pkg) TrustEnabled() bool {
	return p.trust
}

func (p Pkg) archSupported(want string) bool {
	for _, supp := range p.arches {
		if supp == want {
			return true
		}
	}
	return false
}

func (p Pkg) cleanForBuild() error {
	if p.commitHash != "HEAD" {
		return fmt.Errorf("Cannot build from commit hash != HEAD")
	}
	return nil
}
