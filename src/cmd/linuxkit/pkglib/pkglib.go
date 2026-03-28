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

	"gopkg.in/yaml.v3"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/moby"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
)

// Contains fields settable in the build.yml
type pkgInfo struct {
	Image        string            `yaml:"image"`
	Org          string            `yaml:"org"`
	Tag          string            `yaml:"tag,omitempty"` // default to {{.Hash}}
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
	// HashDir, when non-empty, is the directory containing per-package .hash
	// YAML manifest files (written by `linuxkit pkg show-tag --hash-dir`).
	// During dep tag resolution and combined-hash computation, hash files are
	// read instead of recursively calling NewFromConfig, which both avoids
	// dependency cycles and ensures version-specific build variants (e.g.
	// build-2.4.yml for ZFS) are correctly reflected in downstream hashes.
	HashDir string
	// StrictDeps, when true, causes an error if a dep's hash file is absent
	// in HashDir during @lkt: build arg resolution. When false (default), the
	// resolver falls back to NewFromConfig with the default build.yml.
	StrictDeps bool
}

// NewPkgInfo returns a new pkgInfo with default values
func NewPkgInfo() pkgInfo {
	return pkgInfo{
		Org:          "linuxkit",
		Arches:       []string{"amd64", "arm64", "riscv64"},
		Tag:          "{{.Hash}}",
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
	buildYML   string // full path to the build.yml file, not just relative to path
	dockerfile string
	hash       string
	tag        string
	dirty      bool
	commitHash string
	git        *git
	// hashDir is the directory for per-package .hash manifest files used to
	// resolve @lkt: dep tags without recursion.
	hashDir    string
	strictDeps bool
}

// NewFromConfig creates a range of Pkg from a PkglibConfig and paths to packages.
func NewFromConfig(cfg PkglibConfig, args ...string) ([]Pkg, error) {
	// Defaults
	piBase := NewPkgInfo()

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
		}

		// make our own copy of piBase via marshal/unmarshal
		pib, err := yaml.Marshal(&piBase)
		if err != nil {
			return nil, err
		}
		var pi pkgInfo
		if err := yaml.Unmarshal(pib, &pi); err != nil {
			return nil, err
		}

		buildYML := cfg.BuildYML
		buildYmlFile := filepath.Join(pkgPath, buildYML)
		b, err := os.ReadFile(buildYmlFile)
		if err != nil && os.IsNotExist(err) && cfg.HashDir != "" {
			// The requested build yml doesn't exist.  If a hash file is
			// available, read the build-yml field from it — this handles
			// versioned packages (e.g. pkg/zfs with only build-2.3.yml)
			// that were previously processed by update-hashes.
			if m, mErr := readHashManifest(cfg.HashDir, pkgPath); mErr == nil && m != nil && m.BuildYML != "" {
				buildYML = m.BuildYML
				buildYmlFile = filepath.Join(pkgPath, buildYML)
				b, err = os.ReadFile(buildYmlFile)
			}
		}
		if err != nil {
			return nil, err
		}

		dec := yaml.NewDecoder(bytes.NewReader(b))
		dec.KnownFields(true)
		if err := dec.Decode(&pi); err != nil {
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
			pi.Org = os.Getenv("USER")
			if pkgHash == "" {
				pkgHash = "dev"
			}
		}

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
				return nil, fmt.Errorf("bad source format in %s", source)
			}
			srcPath := filepath.Clean(tmp[0])
			dstPath := path.Clean(tmp[1])

			if !filepath.IsAbs(srcPath) {
				srcPath = filepath.Join(pkgPath, srcPath)
			}

			g, err := newGit(srcPath)
			if err != nil {
				return nil, err
			}
			if g == nil {
				return nil, fmt.Errorf("source %s not in a git repository", srcPath)
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

				// Combine the git tree hash with extra source hashes and, when a
				// hash-dir is available, with the resolved tags of all @lkt: build
				// arg deps. Reading dep tags from hash files (written by `linuxkit
				// pkg show-tag --hash-dir`) avoids recursive NewFromConfig calls and
				// the dependency cycles they would create.
				extraHashes := srcHashes
				if pi.BuildArgs != nil && cfg.HashDir != "" {
					for _, arg := range *pi.BuildArgs {
						resolved, err := transformBuildArgValue(arg, buildYmlFile, cfg.HashDir, cfg.StrictDeps)
						if err != nil {
							return nil, fmt.Errorf("resolving build arg %q for hash: %w", arg, err)
						}
						for _, r := range resolved {
							extraHashes += r
						}
					}
				}

				if extraHashes != "" {
					pkgHash += extraHashes
					pkgHash = fmt.Sprintf("%x", sha1.Sum([]byte(pkgHash)))
				}

				if dirty {
					contentHash, err := git.contentHash(pkgHashPath)
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
		tagTmpl := pi.Tag
		if cfg.Tag != "" {
			tagTmpl = cfg.Tag
		}
		if tagTmpl == "" {
			tagTmpl = "{{.Hash}}"
		}

		tmpl, err := template.New("tag").Parse(tagTmpl)
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
			buildYML:      buildYmlFile,
			dockerfile:    pi.Dockerfile,
			git:           git,
			tag:           tag,
			hashDir:       cfg.HashDir,
			strictDeps:    cfg.StrictDeps,
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
		return "", fmt.Errorf("a release tag is required")
	}
	if p.dirty {
		return "", fmt.Errorf("cannot release a dirty package")
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

// Path returns the absolute path to the package source directory.
func (p Pkg) Path() string {
	return p.path
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
		return fmt.Errorf("cannot build from commit hash != HEAD")
	}
	return nil
}

func (p *Pkg) ProcessBuildArgs() error {
	if p.buildArgs == nil {
		return nil
	}
	var buildArgs []string
	for _, arg := range *p.buildArgs {
		// Use hash-dir-aware resolution when available so that @lkt: dep tags
		// come from pre-computed hash files rather than triggering recursive
		// NewFromConfig calls. This ensures version-specific variants (e.g.
		// build-2.4.yml for ZFS) are reflected in the Docker build args.
		transformedLine, err := transformBuildArgValue(arg, p.buildYML, p.hashDir, p.strictDeps)
		if err != nil {
			return fmt.Errorf("error processing build arg %q: %v", arg, err)
		}
		buildArgs = append(buildArgs, transformedLine...)
	}
	if len(buildArgs) > 0 {
		p.buildArgs = &buildArgs
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
