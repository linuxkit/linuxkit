package pkglib

import (
	"flag"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dummyPackage(t *testing.T, tmpDir, yml string) string {
	d, err := ioutil.TempDir(tmpDir, "")
	require.NoError(t, err)

	err = ioutil.WriteFile(filepath.Join(d, "build.yml"), []byte(yml), 0644)
	require.NoError(t, err)

	return d
}

func testBool(t *testing.T, key string, inv bool, forceOn, forceOff string, get func(p Pkg) bool) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	tmpDir := filepath.Join(cwd, t.Name())
	err = os.Mkdir(tmpDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	check := func(pkgDir, override string, f func(t *testing.T, p Pkg)) func(t *testing.T) {
		return func(t *testing.T) {
			flags := flag.NewFlagSet(t.Name(), flag.ExitOnError)
			args := []string{"-hash-path=" + cwd}
			if override != "" {
				args = append(args, override)
			}
			args = append(args, pkgDir)
			pkg, err := NewFromCLI(flags, args...)
			require.NoError(t, err)
			t.Logf("override %q produced %t", override, get(pkg))
			f(t, pkg)
		}
	}

	setting := func(name, cfg string, def bool) {
		var value string
		if cfg != "" {
			value = key + ": " + cfg + "\n"
		}
		pkgDir := dummyPackage(t, tmpDir, `
image: dummy
`+value)

		t.Run(name, func(t *testing.T) {
			t.Run("None", check(pkgDir, "", func(t *testing.T, p Pkg) {
				assert.Equal(t, def, get(p))
			}))
			t.Run("ForceOn", check(pkgDir, forceOn, func(t *testing.T, p Pkg) {
				assert.True(t, get(p))
			}))
			t.Run("ForceOff", check(pkgDir, forceOff, func(t *testing.T, p Pkg) {
				assert.False(t, get(p))
			}))
		})
	}

	// `inv` indicates that the sense of the boolean in
	// `build.yml` is inverted, booleans default to false.
	setting("Default", "", inv)
	setting("SetTrue", "true", !inv)
	setting("SetFalse", "false", inv)
}

func TestNetwork(t *testing.T) {
	testBool(t, "network", false, "-network", "-nonetwork", func(p Pkg) bool { return p.network })
}

func TestCache(t *testing.T) {
	testBool(t, "disable-cache", true, "-enable-cache", "-disable-cache", func(p Pkg) bool { return p.cache })
}

func TestContentTrust(t *testing.T) {
	testBool(t, "disable-content-trust", true, "-enable-content-trust", "-disable-content-trust", func(p Pkg) bool { return p.trust })
}

func testBadBuildYML(t *testing.T, build, expect string) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	tmpDir := filepath.Join(cwd, t.Name())
	err = os.Mkdir(tmpDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pkgDir := dummyPackage(t, tmpDir, build)
	flags := flag.NewFlagSet(t.Name(), flag.ExitOnError)
	args := []string{"-hash-path=" + cwd, pkgDir}
	_, err = NewFromCLI(flags, args...)
	require.Error(t, err)
	assert.Regexp(t, regexp.MustCompile(expect), err.Error())
}

func TestDependsImageNoDigest(t *testing.T) {
	testBadBuildYML(t, `
image: dummy
depends:
  docker-images:
    target-dir: dl
    list:
      - docker.io/library/nginx:latest
`, `image ".*" lacks a digest`)
}

func TestDependsImageBadDigest(t *testing.T) {
	testBadBuildYML(t, `
image: dummy
depends:
  docker-images:
    target-dir: dl
    list:
      - docker.io/library/nginx:latest@sha256:invalid
`, `unable to validate digest in ".*"`)
}

func TestDependsImageBothTargets(t *testing.T) {
	testBadBuildYML(t, `
image: dummy
depends:
  docker-images:
    target: foo.tar
    target-dir: dl
`, `"depends.images.target" and "depends.images.target-dir" are mutually exclusive`)
}

func TestDependsImageBothLists(t *testing.T) {
	testBadBuildYML(t, `
image: dummy
depends:
  docker-images:
    from-file: images.lst
    list:
      - one
`, `"depends.images.list" and "depends.images.from-file" are mutually exclusive`)
}
