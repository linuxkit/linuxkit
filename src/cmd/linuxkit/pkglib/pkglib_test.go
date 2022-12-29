package pkglib

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func dummyPackage(t *testing.T, tmpDir, yml string) string {
	d, err := os.MkdirTemp(tmpDir, "")
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(d, "build.yml"), []byte(yml), 0644)
	require.NoError(t, err)

	return d
}

// testGetBoolPkg given a combination of field, fileKey, fileSetting and override setting,
// create a Pkg that reflects it.
func testGetBoolPkg(t *testing.T, fileKey, cfgKey string, fileSetting, cfgSetting *bool) Pkg {
	// create a git-enabled temporary working directory
	cwd, err := os.Getwd()
	require.NoError(t, err)
	tmpdirBase := path.Join(cwd, "testdatadir")
	err = os.MkdirAll(tmpdirBase, 0755)
	require.NoError(t, err)
	tmpDir, err := os.MkdirTemp(tmpdirBase, "pkglib_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpdirBase)
	var value string
	if fileSetting != nil {
		value = fmt.Sprintf("%s: %v\n", fileKey, *fileSetting)
	}
	pkgDir := dummyPackage(t, tmpDir, `
image: dummy
`+value)

	cfg := PkglibConfig{
		HashPath:   cwd,
		BuildYML:   "build.yml",
		HashCommit: "HEAD",
	}
	if cfgSetting != nil {
		cfgField := reflect.ValueOf(&cfg).Elem().FieldByName(cfgKey)
		cfgField.Set(reflect.ValueOf(cfgSetting))
	}
	pkgs, err := NewFromConfig(cfg, pkgDir)
	require.NoError(t, err)
	return pkgs[0]
}

func TestBoolSettings(t *testing.T) {
	// this is false, because the default is to disable network. The option "Network"
	// is aligned with the value p.network
	var (
		trueVal  = true
		falseVal = false
	)
	tests := []struct {
		testName    string
		cfgKey      string
		fileKey     string
		fileSetting *bool
		cfgSetting  *bool
		pkgField    string
		expectation bool
	}{
		{"Network/Default/None", "Network", "network", nil, nil, "network", false},
		{"Network/Default/ForceOn", "Network", "network", nil, &trueVal, "network", true},
		{"Network/Default/ForceOff", "Network", "network", nil, &falseVal, "network", false},
		{"Network/SetTrue/None", "Network", "network", &trueVal, nil, "network", true},
		{"Network/SetTrue/ForceOn", "Network", "network", &trueVal, &trueVal, "network", true},
		{"Network/SetTrue/ForceOff", "Network", "network", &trueVal, &falseVal, "network", false},
		{"Network/SetFalse/None", "Network", "network", &falseVal, nil, "network", false},
		{"Network/SetFalse/ForceOn", "Network", "network", &falseVal, &trueVal, "network", true},
		{"Network/SetFalse/ForceOff", "Network", "network", &falseVal, &falseVal, "network", false},
		{"Cache/Default/None", "DisableCache", "disable-cache", nil, nil, "cache", true},
		{"Cache/Default/ForceOn", "DisableCache", "disable-cache", nil, &trueVal, "cache", false},
		{"Cache/Default/ForceOff", "DisableCache", "disable-cache", nil, &falseVal, "cache", true},
		{"Cache/SetTrue/None", "DisableCache", "disable-cache", &trueVal, nil, "cache", false},
		{"Cache/SetTrue/ForceOn", "DisableCache", "disable-cache", &trueVal, &trueVal, "cache", false},
		{"Cache/SetTrue/ForceOff", "DisableCache", "disable-cache", &trueVal, &falseVal, "cache", true},
		{"Cache/SetFalse/None", "DisableCache", "disable-cache", &falseVal, nil, "cache", true},
		{"Cache/SetFalse/ForceOn", "DisableCache", "disable-cache", &falseVal, &trueVal, "cache", false},
		{"Cache/SetFalse/ForceOff", "DisableCache", "disable-cache", &falseVal, &falseVal, "cache", true},
	}
	for _, tt := range tests {
		t.Run(tt.testName, func(t *testing.T) {
			pkg := testGetBoolPkg(t, tt.fileKey, tt.cfgKey, tt.fileSetting, tt.cfgSetting)
			returned := reflect.ValueOf(&pkg).Elem().FieldByName(tt.pkgField)

			t.Logf("override field %s value %v produced %t", tt.cfgKey, tt.cfgSetting, returned.Bool())
			assert.Equal(t, tt.expectation, returned.Bool())
		})
	}
}

func testBadBuildYML(t *testing.T, build, expect string) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	tmpDir := filepath.Join(cwd, t.Name())
	err = os.Mkdir(tmpDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pkgDir := dummyPackage(t, tmpDir, build)
	_, err = NewFromConfig(PkglibConfig{
		HashPath:   cwd,
		BuildYML:   "build.yml",
		HashCommit: "HEAD",
	}, pkgDir)
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
