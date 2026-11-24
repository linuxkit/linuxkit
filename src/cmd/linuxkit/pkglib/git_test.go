package pkglib

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", dir}, args...)...).CombinedOutput()
	require.NoError(t, err, string(out))
}

func TestContentHashFileNames(t *testing.T) {
	for _, tt := range []struct {
		name    string
		file    string
		tracked bool
	}{
		{"tracked", "file.txt", true},
		{"tracked with space", "a file.txt", true},
		{"tracked non-ASCII", "caf\u00e9.txt", true},
		{"untracked", "file.txt", false},
		{"untracked non-ASCII", "caf\u00e9.txt", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			runGit(t, dir, "init", "-q")
			runGit(t, dir, "config", "core.quotePath", "true")

			path := filepath.Join(dir, tt.file)
			require.NoError(t, os.WriteFile(path, []byte("one"), 0o644))
			if tt.tracked {
				runGit(t, dir, "add", "-A")
			}

			g, err := newGit(dir)
			require.NoError(t, err)
			require.NotNil(t, g)

			before, err := g.contentHash()
			require.NoError(t, err)

			require.NoError(t, os.WriteFile(path, []byte("two"), 0o644))

			after, err := g.contentHash()
			require.NoError(t, err)
			require.NotEqual(t, before, after, "a changed file must change the hash")
		})
	}
}

// the tracked and untracked lists must not run together
func TestContentHashTrackedAndUntracked(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")

	tracked := filepath.Join(dir, "tracked.txt")
	untracked := filepath.Join(dir, "untracked.txt")
	require.NoError(t, os.WriteFile(tracked, []byte("one"), 0o644))
	runGit(t, dir, "add", "-A")
	require.NoError(t, os.WriteFile(untracked, []byte("one"), 0o644))

	g, err := newGit(dir)
	require.NoError(t, err)
	require.NotNil(t, g)

	before, err := g.contentHash()
	require.NoError(t, err)

	for _, path := range []string{tracked, untracked} {
		require.NoError(t, os.WriteFile(path, []byte("two"), 0o644))
		after, err := g.contentHash()
		require.NoError(t, err)
		require.NotEqual(t, before, after, "a change in %s must change the hash", path)
		before = after
	}
}
