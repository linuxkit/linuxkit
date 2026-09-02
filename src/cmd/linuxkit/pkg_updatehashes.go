package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/pkglib"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// pkgSpec holds a package path and its associated build yml.
type pkgSpec struct {
	Path    string // absolute path to package directory
	BuildYML string // build yml filename (relative to Path), e.g. "build.yml" or "build-2.3.yml"
	OrigArg string // original argument string (for error messages)
}

// parsePkgSpec parses a "path[:build-yml]" argument into a pkgSpec.
func parsePkgSpec(arg string) (pkgSpec, error) {
	var spec pkgSpec
	spec.OrigArg = arg
	if idx := strings.Index(arg, ":"); idx >= 0 {
		spec.Path = arg[:idx]
		spec.BuildYML = arg[idx+1:]
	} else {
		spec.Path = arg
		spec.BuildYML = pkglib.DefaultPkgBuildYML
	}
	absPath, err := filepath.Abs(spec.Path)
	if err != nil {
		return pkgSpec{}, fmt.Errorf("resolving path %q: %w", spec.Path, err)
	}
	spec.Path = absPath
	return spec, nil
}

// imageKey converts an image name (e.g. "lfedge/eve-zfs") to the uppercased
// key form used in build arg names (e.g. "LFEDGE_EVE_ZFS").
func imageKey(imageName string) string {
	s := strings.ReplaceAll(imageName, "/", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return strings.ToUpper(s)
}

// buildYMLBuildArgs is the minimal build.yml structure needed to extract buildArgs.
type buildYMLBuildArgs struct {
	BuildArgs *[]string `yaml:"buildArgs,omitempty"`
}

// depEdges computes the set of spec indices that specs[i] directly depends on.
// Dependency detection: for each @lkt:pkg: or @lkt:pkgs: entry in the build.yml,
// find which other specs they resolve to, applying Dockerfile ARG filtering.
func depEdges(i int, specs []pkgSpec) ([]int, error) {
	spec := specs[i]

	// Read build.yml buildArgs.
	b, err := os.ReadFile(filepath.Join(spec.Path, spec.BuildYML))
	if err != nil {
		return nil, nil // no build yml (should not happen in normal usage)
	}
	var stub buildYMLBuildArgs
	if err := yaml.Unmarshal(b, &stub); err != nil || stub.BuildArgs == nil {
		return nil, nil // no buildArgs
	}

	// Read Dockerfile ARGs for filtering @lkt:pkgs: wildcards.
	usedARGs := pkglib.DockerfileARGNames(filepath.Join(spec.Path, "Dockerfile"))

	// Build a map from absolute path to spec index for O(1) lookup.
	pathToIdx := make(map[string]int, len(specs))
	for j, s := range specs {
		pathToIdx[s.Path] = j
	}

	var deps []int
	seen := make(map[int]bool)

	for _, arg := range *stub.BuildArgs {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, val := parts[0], parts[1]
		if !strings.HasPrefix(val, "@lkt:") {
			continue
		}
		stripped := val[len("@lkt:"):]

		switch {
		case strings.HasPrefix(stripped, "pkg:"):
			relPath := stripped[len("pkg:"):]
			var absPath string
			if filepath.IsAbs(relPath) {
				absPath = relPath
			} else {
				absPath = filepath.Clean(filepath.Join(spec.Path, relPath))
			}
			if j, ok := pathToIdx[absPath]; ok && !seen[j] {
				// For @lkt:pkg:, check if the key is declared in the Dockerfile.
				if len(usedARGs) == 0 || usedARGs[key] {
					deps = append(deps, j)
					seen[j] = true
				}
			}

		case strings.HasPrefix(stripped, "pkgs:"):
			if !strings.Contains(key, "%") {
				continue
			}
			pkgGlob := stripped[len("pkgs:"):]
			if !filepath.IsAbs(pkgGlob) {
				pkgGlob = filepath.Clean(filepath.Join(spec.Path, pkgGlob))
			}
			matches, err := filepath.Glob(pkgGlob)
			if err != nil {
				continue
			}
			for _, match := range matches {
				matchAbs, err := filepath.Abs(match)
				if err != nil {
					continue
				}
				j, ok := pathToIdx[matchAbs]
				if !ok || seen[j] {
					continue
				}
				info, err := os.Stat(matchAbs)
				if err != nil || !info.IsDir() || strings.HasPrefix(info.Name(), ".") {
					continue
				}
				// Compute the ARG key for this dep using its image name.
				imageName, err := pkglib.PkgImageName(specs[j].Path, specs[j].BuildYML)
				if err != nil {
					continue
				}
				updatedKey := strings.ReplaceAll(key, "%", imageKey(imageName))
				// Apply Dockerfile ARG filter.
				if len(usedARGs) > 0 && !usedARGs[updatedKey] {
					continue
				}
				deps = append(deps, j)
				seen[j] = true
			}
		}
	}
	return deps, nil
}

// topoSort performs Kahn's topological sort on n nodes where edges[i] is the
// list of direct dependencies (nodes that i depends on). Returns nodes in
// dependency-first order (deps before consumers).
func topoSort(n int, edges [][]int) ([]int, error) {
	// Build reverse edges: rdeps[j] = list of nodes that depend on j.
	rdeps := make([][]int, n)
	inDegree := make([]int, n)
	for i, deps := range edges {
		inDegree[i] = len(deps)
		for _, dep := range deps {
			rdeps[dep] = append(rdeps[dep], i)
		}
	}

	// Seed the queue with all nodes that have no dependencies.
	var queue []int
	for i := 0; i < n; i++ {
		if inDegree[i] == 0 {
			queue = append(queue, i)
		}
	}
	sort.Ints(queue) // deterministic ordering within the same level

	var result []int
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)
		var newReady []int
		for _, dependent := range rdeps[node] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				newReady = append(newReady, dependent)
			}
		}
		sort.Ints(newReady)
		queue = append(queue, newReady...)
	}

	if len(result) != n {
		return nil, fmt.Errorf("cycle detected in package dependency graph")
	}
	return result, nil
}

func pkgUpdateHashesCmd() *cobra.Command {
	var (
		arch       string
		hashDir    string
		strictDeps bool
	)

	cmd := &cobra.Command{
		Use:   "update-hashes [path[:build-yml]]...",
		Short: "compute and write hash manifests for all packages in dependency order",
		Long: `Compute content hashes for all specified packages and write them to
<hash-dir>/<pkgname>.hash. Packages are processed in topological order (deps
before consumers) so that combined hashes correctly reflect dep versions.

Each argument is a package path optionally suffixed with ':build-yml', e.g.:
  pkg/alpine:build.yml
  pkg/zfs:build-2.3.yml
  pkg/dom0-ztools

Use --strict-deps to error when a package references a dep not in the list.
`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if hashDir == "" {
				return fmt.Errorf("--hash-dir is required")
			}
			if err := os.MkdirAll(hashDir, 0755); err != nil {
				return fmt.Errorf("creating hash dir %s: %w", hashDir, err)
			}

			// Step 1: Parse all package specs.
			specs := make([]pkgSpec, 0, len(args))
			for _, arg := range args {
				spec, err := parsePkgSpec(arg)
				if err != nil {
					return fmt.Errorf("parsing %q: %w", arg, err)
				}
				specs = append(specs, spec)
			}

			// Step 2: Build dependency graph.
			edges := make([][]int, len(specs))
			for i := range specs {
				deps, err := depEdges(i, specs)
				if err != nil {
					return fmt.Errorf("building dep graph for %s: %w", specs[i].OrigArg, err)
				}
				edges[i] = deps
			}

			// Step 3: Topological sort.
			order, err := topoSort(len(specs), edges)
			if err != nil {
				return fmt.Errorf("topological sort: %w", err)
			}

			// Step 4: Process packages in topo order.
			// Each package's deps are already written when we reach it.
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}

			for _, idx := range order {
				spec := specs[idx]

				// Compute the tag using NewFromConfig with hashDir so that
				// @lkt: build arg resolution reads dep tags from hash files.
				cfg := pkglib.PkglibConfig{
					BuildYML:   spec.BuildYML,
					HashDir:    hashDir,
					StrictDeps: strictDeps,
					HashCommit: pkglib.DefaultPkgCommit,
				}
				pkgs, err := pkglib.NewFromConfig(cfg, spec.Path)
				if err != nil {
					return fmt.Errorf("computing hash for %s: %w", spec.OrigArg, err)
				}
				if len(pkgs) == 0 {
					return fmt.Errorf("no package found at %s", spec.OrigArg)
				}
				tag := pkgs[0].Tag()

				// Collect dep entries from the dep graph (all deps already have hash files).
				var depEntries []pkglib.DepEntry
				for _, depIdx := range edges[idx] {
					depSpec := specs[depIdx]
					m, err := pkglib.ReadHashManifest(hashDir, depSpec.Path)
					if err != nil || m == nil {
						continue
					}
					relPath, err := filepath.Rel(cwd, depSpec.Path)
					if err != nil {
						relPath = depSpec.Path
					}
					depEntries = append(depEntries, pkglib.DepEntry{
						Path: filepath.ToSlash(relPath),
						Tag:  m.Tag,
					})
				}

				manifest := pkglib.HashManifest{
					Tag:      tag,
					BuildYML: spec.BuildYML,
					Arch:     arch,
					Deps:     depEntries,
				}
				if _, err := pkglib.WriteHashManifest(hashDir, spec.Path, manifest); err != nil {
					return fmt.Errorf("writing hash manifest for %s: %w", spec.OrigArg, err)
				}

				if v, _ := cmd.Root().PersistentFlags().GetInt("verbose"); v >= 2 {
					fmt.Printf("  %s → %s\n", spec.OrigArg, tag)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&hashDir, "hash-dir", "", "Directory to write .hash manifest files into (required)")
	cmd.Flags().BoolVar(&strictDeps, "strict-deps", false, "Error if a referenced dep is not in the package list or has no hash file")
	cmd.Flags().StringVar(&arch, "arch", "", "Target architecture (e.g. amd64, arm64); stored in hash manifests for invalidation on arch switch")
	_ = cmd.MarkFlagRequired("hash-dir")

	return cmd
}
