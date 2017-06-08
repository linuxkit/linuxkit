package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// Moby is the type of a Moby config file
// stolen from github.com/moby/tool/cmd/moby, perhaps we should make handling
// this into a lib?
type Moby struct {
	Kernel struct {
		Image   string
		Cmdline string
	}
	Init []string
	//Onboot   []MobyImage
	//Services []MobyImage
	//Trust    TrustConfig
	Files []struct {
		Path      string
		Directory bool
		Symlink   string
		Contents  string
		Source    string
	}
}

const linuxkitPath = "../linuxkit.yml"
const kernelSeries = "4.11.x"

func main() {
	if err := run(); err != nil {
		fmt.Printf("error: %s", err)
		os.Exit(1)
	}
}

func run() error {
	var projectsFile string

	flag.StringVar(&projectsFile, "input", "probational/projects.yml", "The input list of projects")
	flag.Parse()

	content, err := ioutil.ReadFile(projectsFile)
	if err != nil {
		return err
	}

	var projects []string
	if err := yaml.Unmarshal(content, &projects); err != nil {
		return err
	}

	content, err = ioutil.ReadFile(linuxkitPath)
	if err != nil {
		return err
	}

	os.Remove("kernel-config/kernel_config.probational")

	cfg := Moby{}
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return err
	}

	for _, p := range projects {
		if err := addProject(&cfg, p); err != nil {
			return err
		}
	}

	// We need users to build the resulting kernel, so use a dummy in the
	// yaml output.
	cfg.Kernel.Image = "your-probational-image"

	raw, err := yaml.Marshal(&cfg)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile("probational/probational.yml", raw, 0644); err != nil {
		return err
	}

	fmt.Println(`Probational configuration complete. Please do:

    cd kernel-config && make IMAGE=probational

The resulting kernel can be used in conjunction with the configuration in
probational/probational.yml to generate a linuxkit image.
`)

	return nil
}

func addProject(cfg *Moby, project string) error {
	fmt.Printf("adding project %s\n", project)

	content, err := ioutil.ReadFile(filepath.Join(project, fmt.Sprintf("%s.yml", project)))
	if err != nil {
		return err
	}

	projectCfg := Moby{}
	if err := yaml.Unmarshal(content, &projectCfg); err != nil {
		return err
	}

	// merge config
	args := strings.Split(cfg.Kernel.Cmdline, " ")
	for _, arg := range strings.Split(projectCfg.Kernel.Cmdline, " ") {
		args = appendIfNotExist(args, arg)
	}
	cfg.Kernel.Cmdline = strings.Join(args, " ")

	for _, init := range projectCfg.Init {
		cfg.Init = appendIfNotExist(cfg.Init, init)
	}

	// TODO: merge other stuff like Onboot, Files, Services, etc.

	// add any kernel patches
	entries, err := ioutil.ReadDir(filepath.Join(project, "kernel"))
	if err != nil {
		return err
	}

	patchesDir := ""
	for _, fi := range entries {
		if !fi.IsDir() {
			continue
		}

		// Exact version match
		if fi.Name() == fmt.Sprintf("patches-%s", kernelSeries) {
			patchesDir = filepath.Join(project, "kernel", fi.Name())
			break
		}

		if strings.HasPrefix(fi.Name(), "patches-") {
			patchesDir = filepath.Join(project, "kernel", fi.Name())
		}
	}

	if patchesDir != "" {
		if !strings.HasSuffix(patchesDir, kernelSeries) {
			fmt.Printf(
				"WARNING: %s doesn't have patches for %s, using %s instead\n",
				project,
				kernelSeries,
				patchesDir[len(patchesDir)-5:],
			)
		}

		probationalPatches := fmt.Sprintf("kernel-config/patches-%s", kernelSeries)
		if err := os.MkdirAll(probationalPatches, 0755); err != nil {
			return err
		}

		patches, err := ioutil.ReadDir(patchesDir)
		if err != nil {
			return err

		}

		for _, patch := range patches {
			if patch.IsDir() {
				continue
			}

			source := filepath.Join(patchesDir, patch.Name())
			dest := filepath.Join(probationalPatches, patch.Name())
			if err := CopyFile(dest, source, patch.Mode()); err != nil {
				return err
			}
		}
	}

	// add any kernel config
	config, err := ioutil.ReadFile(filepath.Join(project, "kernel", "kernel_config.probational"))
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		f, err := os.OpenFile("kernel-config/kernel_config.probational", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = f.Write(config)
		if err != nil {
			fmt.Println("hello2")
			return err
		}
	}

	return nil
}

func appendIfNotExist(arr []string, s string) []string {
	for _, m := range arr {
		if m == s {
			return arr
		}
	}

	return append(arr, s)
}

// CopyFile copies the contents from src to dst atomically.
// If dst does not exist, CopyFile creates it with permissions perm.
// If the copy fails, CopyFile aborts and dst is preserved.
// Lifted from: https://go-review.googlesource.com/c/1591/9/src/io/ioutil/ioutil.go
func CopyFile(dst, src string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := ioutil.TempFile(filepath.Dir(dst), "")
	if err != nil {
		return err
	}
	_, err = io.Copy(tmp, in)
	if err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err = tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	if err = os.Chmod(tmp.Name(), perm); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), dst)
}
