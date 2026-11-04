package build

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDockerHelperPinsPullAndRunContext(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses a shell script")
	}
	dir := t.TempDir()
	calls := filepath.Join(dir, "calls")
	t.Setenv("DOCKER_CONFIG", dir)
	t.Setenv("DOCKER_CONTEXT", "ambient")
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("DOCKER_TEST_CALLS", calls)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$DOCKER_TEST_CALLS"
shift 4
if [ "$1" = run ]; then
 echo 'helper runtime diagnostic' >&2
 exit 1
fi
`
	if err := os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	err := dockerRun("selected", strings.NewReader("input"), io.Discard, "helper", []string{"TARGETARCH=arm64"}, "arg")
	if err == nil || !strings.Contains(err.Error(), "helper runtime diagnostic") {
		t.Fatalf("lost helper diagnostic: %v", err)
	}
	b, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], "--context selected pull helper") || !strings.Contains(lines[1], "--context selected run") {
		t.Fatalf("context not pinned: %s", b)
	}
}
