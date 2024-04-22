package pkglib

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v2"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/moby"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
)

// Contains fields settable in the build.yml
type pkgInfo struct {
	Image        string            `yaml:"image"`
	Org          string            `yaml:"org"`
	Dockerfile   string            `yaml:"dockerfile"`
	Arches       []string          `yaml:"arches"`
	ExtraSources []string          `yaml:"extra-sources"`
	GitRepo      string            `yaml:"gitrepo"` // ??
	Network      bool              `yaml:"network"`
	DisableCache bool              `yaml:"disable-cache"`
	Config       *moby.ImageConfig `yaml:"config"`
	BuildArgs    *[]string         `yaml:"buildArgs,omitempty"`
	Depends      struct {
		DockerImages struct {
			TargetDir string   `yaml:"target-dir"`
			Target    string   `yaml:"target"`
			FromFile  string   `yaml:"from-file"`
			List      []string `yaml:"list"`
		} `yaml:"docker-images"`
	} `yaml:"depends"`
}

// PkglibConfig contains the configuration for the pkglib package.
// It is used to override the default behaviour of the package.
// Fields that are pointers are so that the caller can leave it as nil
// for "use whatever default pkglib has", while non-nil means "explicitly override".
type PkglibConfig struct {
	DisableCache *bool
	Network      *bool
	Org          *string
	BuildYML     string
	Hash         string
	HashCommit   string
	HashPath     string
	Dirty        bool
	Dev          bool
	Tag          string // Tag is a text/template string, defaults to {{.Hash}}
}

// NewPkgInfo returns a new pkgInfo with default values
func NewPkgInfo() pkgInfo {
	return pkgInfo{
		Org:          "linuxkit",
		Arches:       []string{"amd64", "arm64"},
		GitRepo:      "https://github.com/linuxkit/linuxkit",
		Network:      false,
		DisableCache: false,
		Dockerfile:   "Dockerfile",
	}
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
	buildArgs     *[]string
	dockerDepends dockerDepends

	// Internal state
	path       string
	dockerfile string
	hash       string
	tag        string
	dirty      bool
	commitHash string
	git        *git
}

// NewFromConfig creates a range of Pkg from a PkglibConfig and paths to packages.
func NewFromConfig(cfg PkglibConfig, args ...string) ([]Pkg, error) {
	// Defaults
	piBase := NewPkgInfo()

	// TODO(ijc) look for "$(git rev-parse --show-toplevel)/.build-defaults.yml"?

	// Ideally want to look at every directory from root to `pkg`
	// for this file but might be tricky to arrange ordering-wise.

	// These override fields in pi below, bools are in both forms to allow user overrides in either direction.
	// These will apply to all packages built.
	// Other arguments

	var pkgs []Pkg
	for _, pkg := range args {
		var (
			pkgHashPath string
			pkgHash     = cfg.Hash
		)
		pkgPath, err := filepath.Abs(pkg)
		if err != nil {
			return nil, err
		}

		if cfg.HashPath == "" {
			pkgHashPath = pkgPath
		} else {
			pkgHashPath, err = filepath.Abs(cfg.HashPath)
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

		b, err := os.ReadFile(filepath.Join(pkgPath, cfg.BuildYML))
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

		if cfg.Dev {
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
		if cfg.DisableCache != nil {
			pi.DisableCache = *cfg.DisableCache
		}
		if cfg.Network != nil {
			pi.Network = *cfg.Network
		}
		if cfg.Org != nil {
			pi.Org = *cfg.Org
		}
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
			h, err := g.treeHash(srcPath, cfg.HashCommit)
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

		var dirty bool
		if git != nil {
			gitDirty, err := git.isDirty(pkgHashPath, cfg.HashCommit)
			if err != nil {
				return nil, err
			}

			dirty = cfg.Dirty || gitDirty

			if pkgHash == "" {
				if pkgHash, err = git.treeHash(pkgHashPath, cfg.HashCommit); err != nil {
					return nil, err
				}

				if srcHashes != "" {
					pkgHash += srcHashes
					pkgHash = fmt.Sprintf("%x", sha1.Sum([]byte(pkgHash)))
				}

				if dirty {
					contentHash, err := git.contentHash()
					if err != nil {
						return nil, err
					}
					if len(contentHash) < 7 {
						return nil, fmt.Errorf("unexpected hash len: %d", len(contentHash))
					}
					// construct <ls-tree>-dirty-<content hash> tag
					pkgHash += fmt.Sprintf("-dirty-%s", contentHash[0:7])
				}
			}
		}

		// calculate the tag to use based on the template and the pkgHash
		tmpl, err := template.New("tag").Parse(cfg.Tag)
		if err != nil {
			return nil, fmt.Errorf("invalid tag template: %v", err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, map[string]string{"Hash": pkgHash}); err != nil {
			return nil, fmt.Errorf("failed to execute tag template: %v", err)
		}
		tag := buf.String()
		pkgs = append(pkgs, Pkg{
			image:         pi.Image,
			org:           pi.Org,
			hash:          pkgHash,
			commitHash:    cfg.HashCommit,
			arches:        pi.Arches,
			sources:       sources,
			gitRepo:       pi.GitRepo,
			network:       pi.Network,
			cache:         !pi.DisableCache,
			config:        pi.Config,
			buildArgs:     pi.BuildArgs,
			dockerDepends: dockerDepends,
			dirty:         dirty,
			path:          pkgPath,
			dockerfile:    pi.Dockerfile,
			git:           git,
			tag:           tag,
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
	t := p.tag
	if t == "" {
		t = "latest"
	}
	return p.org + "/" + p.image + ":" + t
}

// Image returns the image name without the tag
func (p Pkg) Image() string {
	return p.org + "/" + p.image
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

//nolint:unused // will be used when linuxkit cache is eliminated and we return to docker image cache
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

	if !strings.HasPrefix(p, base) {
		return "", fmt.Errorf("%s must be within package directory", field)
	}

	return p, nil
}
