package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestWriteConfig(t *testing.T) {
	basePath, err := ioutil.TempDir("", "metadata")
	if err != nil {
		t.Fatalf("can't make a temp rootdir %v", err)
	}
	defer os.RemoveAll(basePath)

	json := `{
		"ssh": {
			"sshd_config": {
				"perm": "0600",
				"content": "PermitRootLogin yes\nPasswordAuthentication no"
			}
		},
		"foo": {
			"bar": "foobar",
			"baz": {
				"perm": "0600",
				"content": "bar"
			}
		}
	}`
	if err := processUserData(basePath, []byte(json)); err != nil {
		t.Fatalf("fail to process json %v", err)
	}

	sshd := path.Join(basePath, "ssh", "sshd_config")
	assertContent(t, sshd, "PermitRootLogin yes\nPasswordAuthentication no")
	assertFileMode(t, sshd, 0600)

	bar := path.Join(basePath, "foo", "bar")
	assertContent(t, bar, "foobar")
	assertFileMode(t, bar, 0644)

	assertContent(t, path.Join(basePath, "foo", "baz"), "bar")
}

func assertContent(t *testing.T, path, expected string) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("can't read %v: %v", path, err)
	}
	if !bytes.Equal(file, []byte(expected)) {
		t.Fatalf("%v: expected %v but has %v", path, string(expected), string(file))
	}
}

func assertFileMode(t *testing.T, path string, expected os.FileMode) {
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("can't get fileinfo %v: %v", path, err)
	}
	if expected != info.Mode() {
		t.Fatalf("%v: expected filemode %v but has %v", path, expected, info.Mode())
	}
}
