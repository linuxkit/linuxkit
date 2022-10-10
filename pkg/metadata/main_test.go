package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path"
	"testing"
)

func TestSampleConfig(t *testing.T) {
	basePath, err := os.MkdirTemp("", "metadata")
	if err != nil {
		t.Fatalf("can't make a temp rootdir %v", err)
	}
	defer os.RemoveAll(basePath)

	process(t, basePath, `{
	  "ssh": {
		"entries": {
		  "sshd_config": {
			"perm": "0600",
			"content": "PermitRootLogin yes\nPasswordAuthentication no"
		  }
		}
	  },
	  "foo": {
		"entries": {
		  "bar": {
			"content": "foobar"
		  },
		  "baz": {
			"perm": "0600",
			"content": "bar"
		  }
		}
	  }
	}`)

	sshd := path.Join(basePath, "ssh", "sshd_config")
	assertContent(t, sshd, "PermitRootLogin yes\nPasswordAuthentication no")
	assertPermission(t, sshd, 0600)

	bar := path.Join(basePath, "foo", "bar")
	assertContent(t, bar, "foobar")
	assertPermission(t, bar, 0644)

	assertContent(t, path.Join(basePath, "foo", "baz"), "bar")
}

func TestSerialization(t *testing.T) {
	bin, err := json.Marshal(ConfigFile{
		"ssh": Entry{
			Entries: map[string]Entry{
				"sshd_config": {
					Content: str("PermitRootLogin yes\nPasswordAuthentication no"),
					Perm:    "0600",
				},
			},
		},
		"foo": Entry{
			Entries: map[string]Entry{
				"bar": {
					Content: str("foobar"),
				},
				"baz": {
					Content: str("bar"),
					Perm:    "0600",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Cannot convert to json: %v", err)
	}

	expected := `{"foo":{"entries":{"bar":{"content":"foobar"},"baz":{"perm":"0600","content":"bar"}}},"ssh":{"entries":{"sshd_config":{"perm":"0600","content":"PermitRootLogin yes\nPasswordAuthentication no"}}}}`
	if expected != string(bin) {
		t.Fatalf("Expected %v but has %v", expected, string(bin))
	}
}

func TestWriteSingleFile(t *testing.T) {
	basePath, err := os.MkdirTemp(os.TempDir(), "metadata")
	if err != nil {
		t.Fatalf("can't make a temp rootdir %v", err)
	}
	defer os.RemoveAll(basePath)

	process(t, basePath, `{
	  "hostname": {
		"content": "foobar"
	  }
	}`)

	assertContent(t, path.Join(basePath, "hostname"), "foobar")
}

func TestWriteEmptyFile(t *testing.T) {
	basePath, err := os.MkdirTemp(os.TempDir(), "metadata")
	if err != nil {
		t.Fatalf("can't make a temp rootdir %v", err)
	}
	defer os.RemoveAll(basePath)

	process(t, basePath, `{
	  "empty": {
		"content": ""
	  }
	}`)

	assertContent(t, path.Join(basePath, "empty"), "")
}

func TestWriteEmptyDirectory(t *testing.T) {
	basePath, err := os.MkdirTemp(os.TempDir(), "metadata")
	if err != nil {
		t.Fatalf("can't make a temp rootdir %v", err)
	}
	defer os.RemoveAll(basePath)

	process(t, basePath, `{
	  "empty": {
		"entries": {}
	  }
	}`)

	if _, err := os.Stat(path.Join(basePath, "empty")); err != nil {
		t.Fatalf("empty folder doesn't exist: %v", err)
	}
}

func TestSetPermission(t *testing.T) {
	basePath, err := os.MkdirTemp(os.TempDir(), "metadata")
	if err != nil {
		t.Fatalf("can't make a temp rootdir %v", err)
	}
	defer os.RemoveAll(basePath)

	process(t, basePath, `{
	  "restricted": {
		"perm": "0600",
		"entries": {
		  "password": {
			"perm": "0600",
			"content": "secret"
		  }
		}
	  }
	}`)

	assertPermission(t, path.Join(basePath, "restricted"), 0600|os.ModeDir)
	assertPermission(t, path.Join(basePath, "restricted", "password"), 0600)
}

func TestDeepTree(t *testing.T) {
	basePath, err := os.MkdirTemp("", "metadata")
	if err != nil {
		t.Fatalf("can't make a temp rootdir %v", err)
	}
	defer os.RemoveAll(basePath)

	process(t, basePath, `{
	  "level1": {
		"entries": {
		  "level2": {
			"entries": {
			  "file2": {
				"content": "depth2"
			  },
			  "level3": {
				"entries": {
				  "file3": {
					"content": "depth3"
				  }
				}
			  }
			}
		  }
		}
	  }
	}`)

	assertContent(t, path.Join(basePath, "level1", "level2", "level3", "file3"), "depth3")
	assertContent(t, path.Join(basePath, "level1", "level2", "file2"), "depth2")
}

func str(input string) *string {
	return &input
}

func process(t *testing.T, basePath string, json string) {
	if err := processUserData(basePath, []byte(json)); err != nil {
		t.Fatalf("fail to process json %v", err)
	}
}

func assertPermission(t *testing.T, path string, expected os.FileMode) {
	fileinfo, err := os.Stat(path)
	if err != nil {
		t.Fatalf("%v doesn't exist: %v", path, err)
	}
	if fileinfo.Mode() != expected {
		t.Fatalf("%v: expected %v but has %v", path, expected, fileinfo.Mode())
	}
}

func assertContent(t *testing.T, path, expected string) {
	file, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("can't read %v: %v", path, err)
	}
	if !bytes.Equal(file, []byte(expected)) {
		t.Fatalf("%v: expected %v but has %v", path, expected, string(file))
	}
}
