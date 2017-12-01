package pkglib

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containerd/containerd/reference"
)

type dockerDepends struct {
	images []reference.Spec
	path   string
	dir    bool
}

func newDockerDepends(pkgPath string, pi *pkgInfo) (dockerDepends, error) {
	var err error

	if (pi.Depends.DockerImages.TargetDir != "") && (pi.Depends.DockerImages.Target != "") {
		return dockerDepends{}, fmt.Errorf("\"depends.images.target\" and \"depends.images.target-dir\" are mutually exclusive")
	}
	if (pi.Depends.DockerImages.FromFile != "") && (len(pi.Depends.DockerImages.List) > 0) {
		return dockerDepends{}, fmt.Errorf("\"depends.images.list\" and \"depends.images.from-file\" are mutually exclusive")
	}

	if pi.Depends.DockerImages.Target, err = makeAbsSubpath("depends.image.target", pkgPath, pi.Depends.DockerImages.Target); err != nil {
		return dockerDepends{}, err
	}
	if pi.Depends.DockerImages.TargetDir, err = makeAbsSubpath("depends.image.target-dir", pkgPath, pi.Depends.DockerImages.TargetDir); err != nil {
		return dockerDepends{}, err
	}
	if pi.Depends.DockerImages.FromFile != "" {
		p, err := makeAbsSubpath("depends.image.from-file", pkgPath, pi.Depends.DockerImages.FromFile)
		if err != nil {
			return dockerDepends{}, err
		}
		f, err := os.Open(p)
		if err != nil {
			return dockerDepends{}, err
		}
		defer f.Close()

		s := bufio.NewScanner(f)
		for s.Scan() {
			t := s.Text()
			if len(t) > 0 && t[0] != '#' {
				pi.Depends.DockerImages.List = append(pi.Depends.DockerImages.List, s.Text())
			}
		}

		if err := s.Err(); err != nil {
			return dockerDepends{}, err
		}
	}

	var specs []reference.Spec
	for _, i := range pi.Depends.DockerImages.List {
		s, err := reference.Parse(i)
		if err != nil {
			return dockerDepends{}, err
		}
		dgst := s.Digest()
		if dgst == "" {
			return dockerDepends{}, fmt.Errorf("image %q lacks a digest", i)
		}
		if err := dgst.Validate(); err != nil {
			return dockerDepends{}, fmt.Errorf("unable to validate digest in %q: %v", i, err)
		}
		specs = append(specs, s)
	}

	var dir bool
	path := pi.Depends.DockerImages.Target
	if pi.Depends.DockerImages.TargetDir != "" {
		path = pi.Depends.DockerImages.TargetDir
		dir = true
	}
	return dockerDepends{
		images: specs,
		path:   path,
		dir:    dir,
	}, nil
}

// Do ensures that any dependencies the package has declared are met.
func (dd dockerDepends) Do(d dockerRunner) error {
	if len(dd.images) == 0 {
		return nil
	}

	if dd.dir {
		dir := dd.path

		// Delete and recreate so it is empty
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("failed to remove %q: %v", dir, err)
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create %q: %v", dir, err)
		}
	}

	var refs []string

	for _, s := range dd.images {
		if ok, err := d.pull(s.String()); !ok || err != nil {
			if err != nil {
				return err
			}
			return fmt.Errorf("failed to pull %q", s.String())
		}

		refs = append(refs, s.Locator)
		if dd.dir {
			bn := filepath.Base(s.Locator) + "@" + s.Digest().String()
			path := filepath.Join(dd.path, bn+".tar")
			fmt.Printf("Adding %q as dependency\n", bn)
			if err := d.save(path, s.String()); err != nil {
				return err
			}
		}
	}

	if !dd.dir {
		if err := d.save(dd.path, refs...); err != nil {
			return err
		}
	}

	return nil
}
