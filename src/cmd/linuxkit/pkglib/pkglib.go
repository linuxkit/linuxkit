package pkglib

import (
	"crypto/sha1"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/moby"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
)

// Contains fields settable in the build.yml
type pkgInfo struct {
	Image        string            `yaml:"image"`
	Org          string            `yaml:"org"`
	Arches       []string          `yaml:"arches"`
	ExtraSources []string          `yaml:"extra-sources"`
	GitRepo      string            `yaml:"gitrepo"` // ??
	Network      bool              `yaml:"network"`
	DisableCache bool              `yaml:"disable-cache"`
	Config       *moby.ImageConfig `yaml:"config"`
	Depends      struct {
		DockerImages struct {
			TargetDir string   `yaml:"target-dir"`
			Target    string   `yaml:"target"`
			FromFile  string   `yaml:"from-file"`
			List      []string `yaml:"list"`
		} `yaml:"docker-images"`
	} `yaml:"depends"`
}

// Specifies the source directory for a package and their destination in the build context.
type pkgSource struct {
	src string
	dst string
}

// Pkg encapsulates information about a package's source
type Pkg struct {
	// These correspond to pkgInfo fields
	image         string
	org           string
	arches        []string
	sources       []pkgSource
	gitRepo       string
	network       bool
	trust         bool
	cache         bool
	config        *moby.ImageConfig
	dockerDepends dockerDepends

	// Internal state
	path       string
	hash       string
	dirty      bool
	commitHash string
	git        *git
}

// NewFromCLI creates a range of Pkg from a set of CLI arguments. Calls fs.Parse()
func NewFromCLI(fs *flag.FlagSet, args ...string) ([]Pkg, error) {
	// Defaults
	piBase := pkgInfo{
		Org:          "linuxkit",
		Arches:       []string{"amd64", "arm64", "s390x"},
		GitRepo:      "https://github.com/linuxkit/linuxkit",
		Network:      false,
		DisableCache: false,
	}

	// TODO(ijc) look for "$(git rev-parse --show-toplevel)/.build-defaults.yml"?

	// Ideally want to look at every directory from root to `pkg`
	// for this file but might be tricky to arrange ordering-wise.

	// These override fields in pi below, bools are in both forms to allow user overrides in either direction.
	// These will apply to all packages built.
	argDisableCache := fs.Bool("disable-cache", piBase.DisableCache, "Disable build cache")
	argEnableCache := fs.Bool("enable-cache", !piBase.DisableCache, "Enable build cache")
	argNoNetwork := fs.Bool("nonetwork", !piBase.Network, "Disallow network use during build")
	argNetwork := fs.Bool("network", piBase.Network, "Allow network use during build")

	argOrg := fs.String("org", piBase.Org, "Override the hub org")

	// Other arguments
	var buildYML, hash, hashCommit, hashPath string
	var dirty, devMode bool

	fs.StringVar(&buildYML, "build-yml", "build.yml", "Override the name of the yml file")
	fs.StringVar(&hash, "hash", "", "Override the image hash (default is to query git for the package's tree-sh)")
	fs.StringVar(&hashCommit, "hash-commit", "HEAD", "Override the git commit to use for the hash")
	fs.StringVar(&hashPath, "hash-path", "", "Override the directory to use for the image hash, must be a parent of the package dir (default is to use the package dir)")
	fs.BoolVar(&dirty, "force-dirty", false, "Force the pkg(s) to be considered dirty")
	fs.BoolVar(&devMode, "dev", false, "Force org and hash to $USER and \"dev\" respectively")

	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		return nil, fmt.Errorf("At least one pkg directory is required")
	}

	var pkgs []Pkg
	for _, pkg := range fs.Args() {
		var (
			pkgHashPath string
			pkgHash     = hash
		)
		pkgPath, err := filepath.Abs(pkg)
		if err != nil {
			return nil, err
		}

		if hashPath == "" {
			pkgHashPath = pkgPath
		} else {
			pkgHashPath, err = filepath.Abs(hashPath)
			if err != nil {
				return nil, err
			}

			if !strings.HasPrefix(pkgPath, pkgHashPath) {
				return nil, fmt.Errorf("Hash path is not a prefix of the package path")
			}

			// TODO(ijc) pkgPath and hashPath really ought to be in the same git tree too...
		}

		// make our own copy of piBase. We could use some deepcopy library, but it is just as easy to marshal/unmarshal
		pib, err := yaml.Marshal(&piBase)
		if err != nil {
			return nil, err
		}
		var pi pkgInfo
		if err := yaml.Unmarshal(pib, &pi); err != nil {
			return nil, err
		}

		b, err := ioutil.ReadFile(filepath.Join(pkgPath, buildYML))
		if err != nil {
			return nil, err
		}

		if err := yaml.Unmarshal(b, &pi); err != nil {
			return nil, err
		}

		if pi.Image == "" {
			return nil, fmt.Errorf("Image field is required")
		}

		dockerDepends, err := newDockerDepends(pkgPath, &pi)
		if err != nil {
			return nil, err
		}

		if devMode {
			// If --org is also used then this will be overwritten
			// by argOrg when we iterate over the provided options
			// in the fs.Visit block below.
			pi.Org = os.Getenv("USER")
			if pkgHash == "" {
				pkgHash = "dev"
			}
		}

		// Go's flag package provides no way to see if a flag was set
		// apart from Visit which iterates over only those which were
		// set. This must be run here, rather than earlier, because we need to
		// have read it from the build.yml file first, then override based on CLI.
		fs.Visit(func(f *flag.Flag) {
			switch f.Name {
			case "disable-cache":
				pi.DisableCache = *argDisableCache
			case "enable-cache":
				pi.DisableCache = !*argEnableCache
			case "network":
				pi.Network = *argNetwork
			case "nonetwork":
				pi.Network = !*argNoNetwork
			case "org":
				pi.Org = *argOrg
			}
		})

		var srcHashes string
		sources := []pkgSource{{src: pkgPath, dst: "/"}}

		for _, source := range pi.ExtraSources {
			tmp := strings.Split(source, ":")
			if len(tmp) != 2 {
				return nil, fmt.Errorf("Bad source format in %s", source)
			}
			srcPath := filepath.Clean(tmp[0]) // Should work with windows paths
			dstPath := path.Clean(tmp[1])     // 'path' here because this should be a Unix path

			if !filepath.IsAbs(srcPath) {
				srcPath = filepath.Join(pkgPath, srcPath)
			}

			g, err := newGit(srcPath)
			if err != nil {
				return nil, err
			}
			if g == nil {
				return nil, fmt.Errorf("Source %s not in a git repository", srcPath)
			}
			h, err := g.treeHash(srcPath, hashCommit)
			if err != nil {
				return nil, err
			}

			srcHashes += h
			sources = append(sources, pkgSource{src: srcPath, dst: dstPath})
		}

		git, err := newGit(pkgPath)
		if err != nil {
			return nil, err
		}

		if git != nil {
			gitDirty, err := git.isDirty(pkgHashPath, hashCommit)
			if err != nil {
				return nil, err
			}

			dirty = dirty || gitDirty

			if pkgHash == "" {
				if pkgHash, err = git.treeHash(pkgHashPath, hashCommit); err != nil {
					return nil, err
				}

				if srcHashes != "" {
					pkgHash += srcHashes
					pkgHash = fmt.Sprintf("%x", sha1.Sum([]byte(pkgHash)))
				}

				if dirty {
					pkgHash += "-dirty"
				}
			}
		}

		pkgs = append(pkgs, Pkg{
			image:         pi.Image,
			org:           pi.Org,
			hash:          pkgHash,
			commitHash:    hashCommit,
			arches:        pi.Arches,
			sources:       sources,
			gitRepo:       pi.GitRepo,
			network:       pi.Network,
			cache:         !pi.DisableCache,
			config:        pi.Config,
			dockerDepends: dockerDepends,
			dirty:         dirty,
			path:          pkgPath,
			git:           git,
		})
	}
	return pkgs, nil
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
	t := p.hash
	if t == "" {
		t = "latest"
	}
	return p.org + "/" + p.image + ":" + t
}

// FullTag returns a reference expanded tag
func (p Pkg) FullTag() string {
	return util.ReferenceExpand(p.Tag())
}

// TrustEnabled returns true if trust is enabled
func (p Pkg) TrustEnabled() bool {
	return p.trust
}

// Arches which arches this can be built for
func (p Pkg) Arches() []string {
	return p.arches
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

// Expands path from relative to abs against base, ensuring the result is within base, but is not base itself. Field is the fieldname, to be used for constructing the error.
func makeAbsSubpath(field, base, path string) (string, error) {
	if path == "" {
		return "", nil
	}

	if filepath.IsAbs(path) {
		return "", fmt.Errorf("%s must be relative to package directory", field)
	}

	p, err := filepath.Abs(filepath.Join(base, path))
	if err != nil {
		return "", err
	}

	if p == base {
		return "", fmt.Errorf("%s must not be exactly the package directory", field)
	}

	if !filepath.HasPrefix(p, base) {
		return "", fmt.Errorf("%s must be within package directory", field)
	}

	return p, nil
}
