package dockercontext

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/docker/cli/cli/command"
	dockerendpoint "github.com/docker/cli/cli/context/docker"
	"github.com/docker/cli/cli/context/store"
	"github.com/moby/moby/client"
)

func environment(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, key := range []string{"DOCKER_CONTEXT", "DOCKER_HOST", "DOCKER_TLS", "DOCKER_TLS_VERIFY", "DOCKER_CERT_PATH", "DOCKER_API_VERSION"} {
		t.Setenv(key, "")
	}
	t.Setenv("DOCKER_CONFIG", dir)
	return dir
}

func writeConfig(t *testing.T, dir, current string) {
	t.Helper()
	b, _ := json.Marshal(map[string]string{"currentContext": current})
	if err := os.WriteFile(filepath.Join(dir, "config.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
}

func writeContext(t *testing.T, dir, name, host string) *store.ContextStore {
	t.Helper()
	st := store.New(filepath.Join(dir, "contexts"), command.DefaultContextStoreConfig())
	if err := st.CreateOrUpdate(store.Metadata{Name: name, Endpoints: map[string]any{"docker": dockerendpoint.EndpointMeta{Host: host}}}); err != nil {
		t.Fatal(err)
	}
	return st
}

func TestSelectionPrecedence(t *testing.T) {
	tests := []struct {
		name, mapped, flag, env, host, current, platform, want string
		discover, desktop                                      bool
	}{
		{name: "platform mapping", mapped: "native", flag: "flag", env: "env", host: "unix:///host", current: "current", platform: "linux/arm64", want: "native", discover: true},
		{name: "flag", flag: "flag", env: "env", host: "unix:///host", current: "current", want: "flag"},
		{name: "explicit default", flag: "default", env: "env", current: "current", want: "default"},
		{name: "context environment beats host", env: "env", host: "unix:///host", current: "current", want: "env"},
		{name: "host beats configured context", host: "unix:///host", current: "current", want: "default"},
		{name: "environment beats discovered builder", env: "env", platform: "linux/arm64", discover: true, want: "env"},
		{name: "flag beats discovered builder", flag: "flag", platform: "linux/arm64", discover: true, want: "flag"},
		{name: "platform discovery", current: "current", platform: "linux/arm64", discover: true, want: "linuxkit-linux-arm64"},
		{name: "missing platform context uses desktop", current: "remote", platform: "linux/arm64", desktop: true, want: "desktop-linux"},
		{name: "saved context ignored", current: "orbstack", want: "default"},
		{name: "saved desktop context absent", current: "desktop-linux", want: "default"},
		{name: "desktop before default", current: "remote", desktop: true, want: "desktop-linux"},
		{name: "builder before desktop", platform: "linux/arm64", discover: true, desktop: true, want: "linuxkit-linux-arm64"},
		{name: "explicit default skips desktop", flag: "default", desktop: true, want: "default"},
		{name: "flag skips desktop", flag: "colima", desktop: true, want: "colima"},
		{name: "environment skips desktop", env: "orbstack", desktop: true, want: "orbstack"},
		{name: "host skips desktop", host: "unix:///podman.sock", desktop: true, want: "default"},
		{name: "platform default", want: "default"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := environment(t)
			writeConfig(t, dir, tt.current)
			if tt.discover {
				writeContext(t, dir, "linuxkit-linux-arm64", "unix:///builder")
			}
			if tt.desktop {
				writeContext(t, dir, "desktop-linux", "unix:///desktop.sock")
			}
			t.Setenv("DOCKER_CONTEXT", tt.env)
			t.Setenv("DOCKER_HOST", tt.host)
			s, err := ResolveBuilder(tt.mapped, tt.flag, tt.platform)
			if err != nil {
				t.Fatal(err)
			}
			if s.Name != tt.want || s.ConfigDir != dir {
				t.Fatalf("selection %+v, want %q in %s", s, tt.want, dir)
			}
		})
	}
}

func TestInvalidConfigAndDiscoveredContext(t *testing.T) {
	dir := environment(t)
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if s, err := Resolve(""); err != nil || s.Name != "default" {
		t.Fatalf("context selection should not read currentContext: %+v, %v", s, err)
	}
	// Explicit overrides need not read currentContext, but client initialization
	// still validates config.json before connecting.
	s, err := Resolve("chosen")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Client(); err == nil {
		t.Fatal("client ignored malformed configuration")
	}
	writeConfig(t, dir, "current")
	writeContext(t, dir, "linuxkit-linux-arm64", "unix:///builder")
	// Corrupt discovery metadata must be reported, not treated as absent.
	files, err := filepath.Glob(filepath.Join(dir, "contexts", "meta", "*", "meta.json"))
	if err != nil || len(files) != 1 {
		t.Fatalf("metadata: %v %v", files, err)
	}
	if err := os.WriteFile(files[0], []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveBuilder("", "", "linux/arm64"); err == nil {
		t.Fatal("ignored corrupt discovered context")
	}
}

func TestClientUsesContextInsteadOfAmbientHost(t *testing.T) {
	dir := environment(t)
	writeConfig(t, dir, "colima")
	writeContext(t, dir, "colima", "unix:///colima.sock")
	writeContext(t, dir, "orbstack", "unix:///orbstack.sock")
	for _, tt := range []struct{ flag, env, host, want string }{
		{want: client.DefaultDockerHost},
		{env: "orbstack", host: "unix:///wrong.sock", want: "unix:///orbstack.sock"},
		{flag: "colima", env: "orbstack", host: "unix:///wrong.sock", want: "unix:///colima.sock"},
		{host: "unix:///podman.sock", want: "unix:///podman.sock"},
		{flag: "default", env: "orbstack", host: "unix:///podman.sock", want: "unix:///podman.sock"},
	} {
		t.Setenv("DOCKER_CONTEXT", tt.env)
		t.Setenv("DOCKER_HOST", tt.host)
		s, err := Resolve(tt.flag)
		if err != nil {
			t.Fatal(err)
		}
		cli, err := s.Client()
		if err != nil {
			t.Fatal(err)
		}
		if cli.DaemonHost() != tt.want {
			t.Errorf("host %q, want %q", cli.DaemonHost(), tt.want)
		}
		_ = cli.Close()
	}
	s, err := Resolve("missing")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Client(); err == nil || !strings.Contains(err.Error(), `"missing"`) {
		t.Fatalf("missing context: %v", err)
	}
}

func TestContextTLS(t *testing.T) {
	dir := environment(t)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API negotiation and actual request must both use context-provided TLS.
		w.Header().Set("API-Version", "1.47")
		if strings.HasSuffix(r.URL.Path, "/_ping") {
			_, _ = io.WriteString(w, "OK")
			return
		}
		_, _ = io.WriteString(w, `{"Id":"local","Architecture":"arm64"}`)
	}))
	defer server.Close()
	cert, err := x509.ParseCertificate(server.TLS.Certificates[0].Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	st := writeContext(t, dir, "remote", strings.Replace(server.URL, "https://", "tcp://", 1))
	if err := st.ResetTLSMaterial("remote", &store.ContextTLSData{Endpoints: map[string]store.EndpointTLSData{"docker": {Files: map[string][]byte{"ca.pem": pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})}}}}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DOCKER_HOST", "unix:///wrong.sock")
	s, err := Resolve("remote")
	if err != nil {
		t.Fatal(err)
	}
	cli, err := s.Client()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cli.Close() }()
	info, err := cli.ImageInspect(context.Background(), "local")
	if err != nil {
		t.Fatal(err)
	}
	if info.Architecture != "arm64" {
		t.Fatalf("inspect result %+v", info)
	}
}

func TestCommandKeepsStderrAndExitStatus(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses a shell script")
	}
	dir := environment(t)
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "docker"), []byte("#!/bin/sh\nprintf '%s\\n' \"$@\"\necho 'Cannot connect to unix:///var/run/docker.sock: no such file or directory' >&2\nexit 17\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	s := Selection{Name: "desktop-linux", ConfigDir: dir}
	var out strings.Builder
	err := s.Run(context.Background(), nil, &out, io.Discard, "container", "ls")
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 17 {
		t.Fatalf("lost exit status: %v", err)
	}
	for _, text := range []string{"desktop-linux", "container ls", "/var/run/docker.sock", "no such file"} {
		if !strings.Contains(err.Error(), text) {
			t.Errorf("missing %q in %v", text, err)
		}
	}
	got := strings.Split(strings.TrimSpace(out.String()), "\n")
	if !reflect.DeepEqual(got, s.Args("container", "ls")) {
		t.Fatalf("argv %q", got)
	}
}

func TestBuilderURL(t *testing.T) {
	s := Selection{Name: "remote&other=ignored"}
	u, err := url.Parse(s.BuilderURL("builder"))
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("context") != s.Name || u.Host != "builder" {
		t.Fatalf("builder URL %s", u)
	}
}

func TestBuildKitExecPinsContextAndConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fixture uses a shell script")
	}
	dir := environment(t)
	calls := filepath.Join(dir, "buildkit-calls")
	t.Setenv("DOCKER_TEST_CALLS", calls)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	script := "#!/bin/sh\nprintf '%s\\n' \"$*\" > \"$DOCKER_TEST_CALLS\"\necho 'buildctl connection diagnostic' >&2\nexit 1\n"
	if err := os.WriteFile(filepath.Join(dir, "docker"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	s := Selection{Name: "remote", ConfigDir: dir}
	conn, err := s.dialBuilder(context.Background(), "builder")
	if err != nil {
		t.Fatal(err)
	}
	_, readErr := io.ReadAll(conn)
	closeErr := conn.Close()
	if readErr == nil || !strings.Contains(readErr.Error(), "buildctl connection diagnostic") {
		t.Fatalf("lost exec diagnostic: read=%v, close=%v", readErr, closeErr)
	}
	b, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join(s.Args("exec", "-i", "builder", "buildctl", "dial-stdio"), " ")
	if strings.TrimSpace(string(b)) != want {
		t.Fatalf("BuildKit exec arguments %q, want %q", b, want)
	}
}
