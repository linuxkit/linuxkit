package pkglib

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/cli/cli/command"
	dockerendpoint "github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
)

func fakeDocker(t *testing.T, script string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses a shell script")
	}
	dir := t.TempDir()
	log := filepath.Join(dir, "calls")
	t.Setenv("DOCKER_CONFIG", dir)
	t.Setenv("DOCKER_CONTEXT", "")
	t.Setenv("DOCKER_HOST", "")
	t.Setenv("DOCKER_TEST_CALLS", log)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(dir, "docker"), []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$DOCKER_TEST_CALLS\"\n"+script), 0700); err != nil {
		t.Fatal(err)
	}
	return log
}

func TestDiscoveredBuilderFailureDoesNotChangeContext(t *testing.T) {
	calls := fakeDocker(t, "echo 'Cannot connect to unix:///native.sock: no such file or directory' >&2\nexit 1\n")
	st := store.New(filepath.Join(os.Getenv("DOCKER_CONFIG"), "contexts"), command.DefaultContextStoreConfig())
	if err := st.CreateOrUpdate(store.Metadata{Name: "linuxkit-linux-arm64", Endpoints: map[string]any{"docker": dockerendpoint.EndpointMeta{Host: "unix:///native.sock"}}}); err != nil {
		t.Fatal(err)
	}
	dr := NewDockerRunner(false, BuilderConfig{Image: "buildkit"})
	_, err := dr.Builder(context.Background(), "", "linux/arm64")
	if err == nil {
		t.Fatal("expected unavailable daemon error")
	}
	for _, text := range []string{"linuxkit-linux-arm64", "native.sock", "no such file"} {
		if !strings.Contains(err.Error(), text) {
			t.Errorf("missing %q in %v", text, err)
		}
	}
	b, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 1 || !strings.Contains(lines[0], "--context linuxkit-linux-arm64 container ls") {
		t.Fatalf("unexpected daemon fallback: %s", b)
	}
}

func TestBuilderConfigCopyKeepsMappedContext(t *testing.T) {
	calls := fakeDocker(t, `# Skip explicit global --config and --context flags.
shift 4
case "$1 $2" in
 "container ls") exit 0 ;;
 "container create") echo builder-id ;;
 "container cp") cat >/dev/null ;;
 "container start") echo 'builder startup diagnostic' >&2; exit 1 ;;
 *) echo "unexpected command $*" >&2; exit 2 ;;
esac
`)
	t.Setenv("DOCKER_CONTEXT", "desktop-linux")
	configPath := filepath.Join(t.TempDir(), "buildkitd.toml")
	if err := os.WriteFile(configPath, []byte("debug = true\n"), 0600); err != nil {
		t.Fatal(err)
	}
	dr := NewDockerRunner(false, BuilderConfig{Context: "orbstack", Image: "buildkit", ConfigPath: configPath})
	_, err := dr.Builder(context.Background(), "remote", "linux/arm64")
	if err == nil || !strings.Contains(err.Error(), "builder startup diagnostic") {
		t.Fatalf("lost startup diagnostic: %v", err)
	}
	b, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 4 {
		t.Fatalf("unexpected calls: %s", b)
	}
	for _, line := range lines {
		if !strings.Contains(line, "--context remote container") {
			t.Errorf("context lost: %s", line)
		}
	}
	if !strings.Contains(lines[2], "container cp - linuxkit-builder:/etc/buildkit") {
		t.Fatalf("copy command: %s", lines[2])
	}
}

func TestGeneralOutputContextIsSeparateFromBuilder(t *testing.T) {
	calls := fakeDocker(t, "exit 0\n")
	dr := NewDockerRunner(false, BuilderConfig{Context: "orbstack"}).(*dockerRunnerImpl)
	if err := dr.copyFilesToContainer("remote", "builder", map[string][]byte{"buildkitd.toml": []byte("debug=true")}); err != nil {
		t.Fatal(err)
	}
	if err := dr.Load(strings.NewReader("image tar")); err != nil {
		t.Fatal(err)
	}
	if err := dr.Tag("built", "tagged"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 3 || !strings.Contains(lines[0], "--context remote container cp") || !strings.Contains(lines[1], "--context orbstack image load") || !strings.Contains(lines[2], "--context orbstack image tag") {
		t.Fatalf("unexpected endpoint routing: %s", b)
	}
}

func TestDockerCommandErrorAndPullError(t *testing.T) {
	fakeDocker(t, "echo 'permission denied on unix:///colima.sock' >&2\nexit 1\n")
	dr := NewDockerRunner(false, BuilderConfig{Context: "colima"}).(*dockerRunnerImpl)
	if err := dr.command(nil, io.Discard, io.Discard, "container", "ls"); err == nil || !strings.Contains(err.Error(), "permission denied on unix:///colima.sock") {
		t.Fatalf("lost command error: %v", err)
	}
	ok, err := dr.Pull("missing")
	if ok || err == nil || !strings.Contains(err.Error(), "colima.sock") {
		t.Fatalf("pull lost error: %v %v", ok, err)
	}
}

func TestDesktopBuilderFailureDoesNotFallBackToDefault(t *testing.T) {
	calls := fakeDocker(t, "echo 'Cannot connect to unix:///desktop.sock: no such file or directory' >&2\nexit 1\n")
	st := store.New(filepath.Join(os.Getenv("DOCKER_CONFIG"), "contexts"), command.DefaultContextStoreConfig())
	if err := st.CreateOrUpdate(store.Metadata{Name: "desktop-linux", Endpoints: map[string]any{"docker": dockerendpoint.EndpointMeta{Host: "unix:///desktop.sock"}}}); err != nil {
		t.Fatal(err)
	}
	dr := NewDockerRunner(false, BuilderConfig{Image: "buildkit"})
	_, err := dr.Builder(context.Background(), "", "linux/arm64")
	if err == nil || !strings.Contains(err.Error(), "desktop.sock") {
		t.Fatalf("lost desktop daemon error: %v", err)
	}
	b, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 1 || !strings.Contains(lines[0], "--context desktop-linux container ls") {
		t.Fatalf("unexpected default fallback: %s", b)
	}
}
