package main

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
)

func TestOverrides(t *testing.T) {
	var yamlCaps = []string{"CAP_SYS_ADMIN"}

	var yaml MobyImage = MobyImage{
		Name:         "test",
		Image:        "testimage",
		Capabilities: &yamlCaps,
	}

	var labelCaps = []string{"CAP_SYS_CHROOT"}

	var label MobyImage = MobyImage{
		Capabilities: &labelCaps,
		Cwd:          "/label/directory",
	}

	var inspect types.ImageInspect
	var config container.Config

	labelJSON, err := json.Marshal(label)
	if err != nil {
		t.Error(err)
	}
	config.Labels = map[string]string{"org.mobyproject.config": string(labelJSON)}

	inspect.Config = &config

	oci, err := ConfigInspectToOCI(yaml, inspect)
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
