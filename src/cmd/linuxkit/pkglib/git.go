package pkglib

// Thin wrappers around git CLI invocations

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const debugGitCommands = false

// 040000 tree 7804129bd06218b72c298139a25698a748d253c6\tpkg/init
var treeHashRe *regexp.Regexp

func init() {
	treeHashRe = regexp.MustCompile("^[0-7]{6} [^ ]+ ([0-9a-f]{40})\t.+\n$")
}

func gitCommandStdout(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Stderr = os.Stderr

	if debugGitCommands {
		fmt.Fprintf(os.Stderr, "+ %v\n", cmd.Args)
	}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func gitCommand(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if debugGitCommands {
		fmt.Fprintf(os.Stderr, "+ %v\n", cmd.Args)
	}
	return cmd.Run()
}

func gitTreeHash(pkg, commit string) (string, error) {
	out, err := gitCommandStdout("ls-tree", "--full-tree", commit, "--", pkg)
	if err != nil {
		return "", err
	}

	matches := treeHashRe.FindStringSubmatch(out)
	if len(matches) != 2 {
		return "", fmt.Errorf("Unable to parse ls-tree output: %q", out)
	}

	return matches[1], nil
}

func gitCommitHash(commit string) (string, error) {
	out, err := gitCommandStdout("rev-parse", commit)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out), nil
}

func gitCommitTag(commit string) (string, error) {
	out, err := gitCommandStdout("tag", "-l", "--points-at", commit)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(out), nil
}

func gitIsDirty(pkg, commit string) (bool, error) {
	// If it isn't HEAD it can't be dirty
	if commit != "HEAD" {
		return false, nil
	}

	// Update cache, otherwise files which have an updated
	// timestamp but no actual changes are marked as changes
	// because `git diff-index` only uses the `lstat` result and
	// not the actual file contents. Running `git update-index
	// --refresh` updates the cache.
	if err := gitCommand("update-index", "-q", "--refresh"); err != nil {
		return false, err
	}

	err := gitCommand("diff-index", "--quiet", commit, "--", pkg)
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
