package build

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/containerd/containerd/v2/pkg/reference"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	log "github.com/sirupsen/logrus"
)

func TestDockerCacheConnectionFailureIsReportedBeforeFallback(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DOCKER_CONFIG", dir)
	t.Setenv("DOCKER_CONTEXT", "")
	t.Setenv("DOCKER_TLS", "")
	t.Setenv("DOCKER_TLS_VERIFY", "")
	t.Setenv("DOCKER_CERT_PATH", "")
	t.Setenv("DOCKER_HOST", "unix://"+filepath.ToSlash(filepath.Join(dir, "missing.sock")))
	// Stop the cache fallback locally, without contacting any registry.
	cacheFile := filepath.Join(dir, "not-a-directory")
	if err := os.WriteFile(cacheFile, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	logger := log.StandardLogger()
	output, level := logger.Out, logger.Level
	var diagnostics bytes.Buffer
	logger.SetOutput(&diagnostics)
	logger.SetLevel(log.WarnLevel)
	defer func() { logger.SetOutput(output); logger.SetLevel(level) }()
	ref, err := reference.Parse("example.invalid/local-only:latest")
	if err != nil {
		t.Fatal(err)
	}
	_, err = imageSource(&ref, false, cacheFile, true, "default", imagespec.Platform{OS: "linux", Architecture: "arm64"})
	if err == nil {
		t.Fatal("expected local cache fallback to fail")
	}
	for _, text := range []string{"context", "default", "missing.sock", "falling back"} {
		if !strings.Contains(diagnostics.String(), text) {
			t.Errorf("missing %q in warning %s", text, &diagnostics)
		}
	}
}
