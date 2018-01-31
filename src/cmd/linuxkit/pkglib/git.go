package pkglib

// Thin wrappers around git CLI invocations

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

// 040000 tree 7804129bd06218b72c298139a25698a748d253c6\tpkg/init
var treeHashRe *regexp.Regexp

func init() {
	treeHashRe = regexp.MustCompile("^[0-7]{6} [^ ]+ ([0-9a-f]{40})\t.+\n$")
}

type git struct {
	dir string
}

// Returns git==nil and no error if the path is not within a git repository
func newGit(dir string) (*git, error) {
	g := &git{dir}

	// Check if dir really is within a git directory
	ok, err := g.isWorkTree(dir)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, nil
	}
	return g, nil
}

func (g git) mkCmd(args ...string) *exec.Cmd {
	return exec.Command("git", append([]string{"-C", g.dir}, args...)...)
}

func (g git) commandStdout(stderr io.Writer, args ...string) (string, error) {
	cmd := g.mkCmd(args...)
	cmd.Stderr = stderr
	log.Debugf("Executing: %v", cmd.Args)

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (g git) command(args ...string) error {
	cmd := g.mkCmd(args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Debugf("Executing: %v", cmd.Args)

	return cmd.Run()
}

func (g git) isWorkTree(pkg string) (bool, error) {
	tf, err := g.commandStdout(nil, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		// If we executed git ok but it errored then that's because this isn't a git repo
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, err
	}

	tf = strings.TrimSpace(tf)

	if tf == "true" {
		return true, nil
	}

	return false, fmt.Errorf("unexpected output from git rev-parse --is-inside-work-tree: %s", tf)
}

func (g git) treeHash(pkg, commit string) (string, error) {
	// we have to check if pkg is at the top level of the git tree,
	// if that's the case we need to use tree hash from the commit itself
	out, err := g.commandStdout(nil, "rev-parse", "--prefix", pkg, "--show-toplevel")
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(out) == pkg {
		out, err = g.commandStdout(nil, "show", "--format=%T", "-s", commit)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(out), nil
	}

	out, err = g.commandStdout(os.Stderr, "ls-tree", "--full-tree", commit, "--", pkg)
	if err != nil {
		return "", err
	}

	if out == "" {
		return "", fmt.Errorf("Package %s is not in git", pkg)
	}

	matches := treeHashRe.FindStringSubmatch(out)
	if len(matches) != 2 {
		return "", fmt.Errorf("Unable to parse ls-tree output: %q", out)
	}

	return matches[1], nil
}

func (g git) commitHash(commit string) (string, error) {
	out, err := g.commandStdout(os.Stderr, "rev-parse", commit)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out), nil
}

func (g git) commitTag(commit string) (string, error) {
	out, err := g.commandStdout(os.Stderr, "tag", "-l", "--points-at", commit)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out), nil
}

func (g git) isDirty(pkg, commit string) (bool, error) {
	// If it isn't HEAD it can't be dirty
	if commit != "HEAD" {
		return false, nil
	}

	// Update cache, otherwise files which have an updated
	// timestamp but no actual changes are marked as changes
	// because `git diff-index` only uses the `lstat` result and
	// not the actual file contents. Running `git update-index
	// --refresh` updates the cache.
	if err := g.command("update-index", "-q", "--refresh"); err != nil {
		return false, err
	}

	err := g.command("diff-index", "--quiet", commit, "--", pkg)
	if err == nil {
		return false, nil
	}
	switch err.(type) {
	case *exec.ExitError:
		// diff-index exits with an error if there are differences
		return true, nil
	default:
		return false, err
	}
}
