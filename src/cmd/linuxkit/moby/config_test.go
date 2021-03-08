package moby

import (
	"encoding/json"
	"reflect"
	"testing"

	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
)

func setupInspect(t *testing.T, label ImageConfig) imagespec.ImageConfig {
	var config imagespec.ImageConfig

	labelJSON, err := json.Marshal(label)
	if err != nil {
		t.Error(err)
	}
	config.Labels = map[string]string{"org.mobyproject.config": string(labelJSON)}

	return config
}

func TestOverrides(t *testing.T) {
	idMap := map[string]uint32{}

	var yamlCaps = []string{"CAP_SYS_ADMIN"}

	var yaml = Image{
		Name:  "test",
		Image: "testimage",
		ImageConfig: ImageConfig{
			Capabilities: &yamlCaps,
		},
	}

	var labelCaps = []string{"CAP_SYS_CHROOT"}

	var label = ImageConfig{
		Capabilities: &labelCaps,
		Cwd:          "/label/directory",
	}

	inspect := setupInspect(t, label)

	oci, _, err := ConfigToOCI(&yaml, inspect, idMap)
	if err != nil {
		t.Error(err)
	}

	if !reflect.DeepEqual(oci.Process.Capabilities.Bounding, yamlCaps) {
		t.Error("Expected yaml capabilities to override but got", oci.Process.Capabilities.Bounding)
	}
	if oci.Process.Cwd != label.Cwd {
		t.Error("Expected label Cwd to be applied, got", oci.Process.Cwd)
	}
}

func TestInvalidCap(t *testing.T) {
	idMap := map[string]uint32{}

	yaml := Image{
		Name:  "test",
		Image: "testimage",
	}

	labelCaps := []string{"NOT_A_CAP"}
	var label = ImageConfig{
		Capabilities: &labelCaps,
	}

	inspect := setupInspect(t, label)

	_, _, err := ConfigToOCI(&yaml, inspect, idMap)
	if err == nil {
		t.Error("expected error, got valid OCI config")
	}
}

func TestIdMap(t *testing.T) {
	idMap := map[string]uint32{"test": 199}

	var uid interface{} = "test"
	var gid interface{} = 76

	yaml := Image{
		Name:  "test",
		Image: "testimage",
		ImageConfig: ImageConfig{
			UID: &uid,
			GID: &gid,
		},
	}

	var label = ImageConfig{}

	inspect := setupInspect(t, label)

	oci, _, err := ConfigToOCI(&yaml, inspect, idMap)
	if err != nil {
		t.Error(err)
	}

	if oci.Process.User.UID != 199 {
		t.Error("Expected named uid to work")
	}
	if oci.Process.User.GID != 76 {
		t.Error("Expected numerical gid to work")
	}
}
