package build

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/containerd/containerd/reference"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/moby"
	"github.com/opencontainers/runtime-spec/specs-go"
	log "github.com/sirupsen/logrus"
)

type tarWriter interface {
	Close() error
	Flush() error
	Write(b []byte) (n int, err error)
	WriteHeader(hdr *tar.Header) error
}

// This uses Docker to convert a Docker image into a tarball. It would be an improvement if we
// used the containerd libraries to do this instead locally direct from a local image
// cache as it would be much simpler.

// Unfortunately there are some files that Docker always makes appear in a running image and
// export shows them. In particular we have no way for a user to specify their own resolv.conf.
// Even if we were not using docker export to get the image, users of docker build cannot override
// the resolv.conf either, as it is not writeable and bind mounted in.

var exclude = map[string]bool{
	".dockerenv":   true,
	"Dockerfile":   true,
	"dev/console":  true,
	"dev/pts":      true,
	"dev/shm":      true,
	"etc/hostname": true,
}

var replace = map[string]string{
	"etc/hosts": `127.0.0.1       localhost
::1     localhost ip6-localhost ip6-loopback
fe00::0 ip6-localnet
ff00::0 ip6-mcastprefix
ff02::1 ip6-allnodes
ff02::2 ip6-allrouters
`,
	"etc/resolv.conf": `
# no resolv.conf configured
`,
}

// Files which must exist. They may be created as part of 'docker export',
// in which case they need their timestamp fixed. If they do not exist, they must be created.
var touch = map[string]tar.Header{
	"dev/": {
		Size:     0,
		Mode:     0755,
		Uid:      0,
		Gid:      0,
		ModTime:  defaultModTime,
		Name:     "dev",
		Typeflag: tar.TypeDir,
	},
	"dev/pts/": {
		Size:     0,
		Mode:     0755,
		Uid:      0,
		Gid:      0,
		ModTime:  defaultModTime,
		Name:     "dev/pts",
		Typeflag: tar.TypeDir,
	},
	"dev/shm/": {
		Size:     0,
		Mode:     0755,
		Uid:      0,
		Gid:      0,
		ModTime:  defaultModTime,
		Name:     "dev/shm",
		Typeflag: tar.TypeDir,
	},
	"etc/": {
		Size:     0,
		Mode:     0755,
		Uid:      0,
		Gid:      0,
		ModTime:  defaultModTime,
		Name:     "etc",
		Typeflag: tar.TypeDir,
	},
	"etc/mtab": {
		Size:     0,
		Mode:     0755,
		Uid:      0,
		Gid:      0,
		ModTime:  defaultModTime,
		Name:     "etc/mtab",
		Typeflag: tar.TypeSymlink,
		Linkname: "/proc/mounts",
	},
	"etc/resolv.conf": {
		Size:     0,
		Mode:     0644,
		Uid:      0,
		Gid:      0,
		ModTime:  defaultModTime,
		Name:     "etc/resolv.conf",
		Typeflag: tar.TypeReg,
	},
	"etc/hosts": {
		Size:     0,
		Mode:     0644,
		Uid:      0,
		Gid:      0,
		ModTime:  defaultModTime,
		Name:     "etc/hosts",
		Typeflag: tar.TypeReg,
	},
	"proc/": {
		Size:     0,
		Mode:     0755,
		Uid:      0,
		Gid:      0,
		ModTime:  defaultModTime,
		Name:     "proc",
		Typeflag: tar.TypeDir,
	},
	"sys/": {
		Size:     0,
		Mode:     0755,
		Uid:      0,
		Gid:      0,
		ModTime:  defaultModTime,
		Name:     "sys",
		Typeflag: tar.TypeDir,
	},
}

// tarPrefix creates the leading directories for a path
// path is the path to prefix, location is where this appears in the linuxkit.yaml file
func tarPrefix(path, location string, ref *reference.Spec, tw tarWriter) error {
	if path == "" {
		return nil
	}
	if path[len(path)-1] != '/' {
		return fmt.Errorf("path does not end with /: %s", path)
	}
	path = path[:len(path)-1]
	if path[0] == '/' {
		return fmt.Errorf("path should be relative: %s", path)
	}
	mkdir := ""
	for _, dir := range strings.Split(path, "/") {
		mkdir = mkdir + dir
		hdr := &tar.Header{
			Name:     mkdir,
			Mode:     0755,
			ModTime:  defaultModTime,
			Typeflag: tar.TypeDir,
			Format:   tar.FormatPAX,
			PAXRecords: map[string]string{
				moby.PaxRecordLinuxkitSource:   ref.String(),
				moby.PaxRecordLinuxkitLocation: location,
			},
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		mkdir = mkdir + "/"
	}
	return nil
}

// ImageTar takes a Docker image and outputs it to a tar stream
// location is where it is in the linuxkit.yaml file
func ImageTar(location string, ref *reference.Spec, prefix string, tw tarWriter, resolv string, opts BuildOpts) (e error) {
	log.Debugf("image tar: %s %s", ref, prefix)
	if prefix != "" && prefix[len(prefix)-1] != '/' {
		return fmt.Errorf("prefix does not end with /: %s", prefix)
	}

	err := tarPrefix(prefix, location, ref, tw)
	if err != nil {
		return err
	}

	// pullImage first checks in the cache, then pulls the image.
	// If pull==true, then it always tries to pull from registry.
	src, err := imagePull(ref, opts.Pull, opts.CacheDir, opts.DockerCache, opts.Arch)
	if err != nil {
		return fmt.Errorf("could not pull image %s: %v", ref, err)
	}

	contents, err := src.TarReader()
	if err != nil {
		return fmt.Errorf("could not unpack image %s: %v", ref, err)
	}

	defer contents.Close()

	// all of the files in `touch` must exist in the output, so keep track if
	// we found them, and, if not, create them
	touchFound := map[string]bool{}

	// now we need to filter out some files from the resulting tar archive

	tr := tar.NewReader(contents)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// force PAX format, since it allows for unlimited Name/Linkname
		// and we move all files below prefix.
		hdr.Format = tar.FormatPAX
		// ensure we record the source of the file in the PAX header
		if hdr.PAXRecords == nil {
			hdr.PAXRecords = make(map[string]string)
		}
		hdr.PAXRecords[moby.PaxRecordLinuxkitSource] = ref.String()
		hdr.PAXRecords[moby.PaxRecordLinuxkitLocation] = location
		if exclude[hdr.Name] {
			log.Debugf("image tar: %s %s exclude %s", ref, prefix, hdr.Name)
			_, err = io.Copy(io.Discard, tr)
			if err != nil {
				return err
			}
		} else if replace[hdr.Name] != "" {
			if hdr.Name != "etc/resolv.conf" || resolv == "" {
				contents := replace[hdr.Name]
				hdr.Size = int64(len(contents))
				hdr.Name = prefix + hdr.Name
				hdr.ModTime = defaultModTime
				log.Debugf("image tar: %s %s add %s (replaced)", ref, prefix, hdr.Name)
				if err := tw.WriteHeader(hdr); err != nil {
					return err
				}
				_, err = tw.Write([]byte(contents))
				if err != nil {
					return err
				}
			} else {
				// replace resolv.conf with specified symlink
				hdr.Name = prefix + hdr.Name
				hdr.Size = 0
				hdr.Typeflag = tar.TypeSymlink
				hdr.Linkname = resolv
				hdr.ModTime = defaultModTime
				log.Debugf("image tar: %s %s add resolv symlink /etc/resolv.conf -> %s", ref, prefix, resolv)
				if err := tw.WriteHeader(hdr); err != nil {
					return err
				}
			}
			_, err = io.Copy(io.Discard, tr)
			if err != nil {
				return err
			}
		} else {
			if found, ok := touch[hdr.Name]; ok {
				log.Debugf("image tar: %s %s add %s (touch)", ref, prefix, hdr.Name)
				hdr.ModTime = found.ModTime
				// record that we saw this one
				touchFound[hdr.Name] = true
			} else {
				log.Debugf("image tar: %s %s add %s (original)", ref, prefix, hdr.Name)
			}
			hdr.Name = prefix + hdr.Name
			if hdr.Typeflag == tar.TypeLink {
				// hard links are referenced by full path so need to be adjusted
				hdr.Linkname = prefix + hdr.Linkname
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			_, err = io.Copy(tw, tr)
			if err != nil {
				return err
			}
		}
	}
	// now make sure that we had all of the touch files
	// be sure to do it in a consistent order
	var touchNames []string
	for name := range touch {
		touchNames = append(touchNames, name)
	}
	sort.Strings(touchNames)
	for _, name := range touchNames {
		if touchFound[name] {
			log.Debugf("image tar: %s already found in original image", name)
			continue
		}
		hdr := touch[name]
		// ensure that we record the source of the file
		if hdr.PAXRecords == nil {
			hdr.PAXRecords = make(map[string]string)
		}
		hdr.PAXRecords[moby.PaxRecordLinuxkitSource] = ref.String()
		hdr.PAXRecords[moby.PaxRecordLinuxkitLocation] = location
		origName := hdr.Name
		hdr.Name = prefix + origName
		hdr.Format = tar.FormatPAX
		contents, ok := replace[origName]
		switch {
		case ok && len(contents) > 0 && (origName != "etc/resolv.conf" || resolv == ""):
			hdr.Size = int64(len(contents))
		case origName == "etc/resolv.conf" && resolv != "":
			// replace resolv.conf with specified symlink
			hdr.Size = 0
			hdr.Typeflag = tar.TypeSymlink
			hdr.Linkname = resolv
			log.Debugf("image tar: %s %s add resolv symlink /etc/resolv.conf -> %s", ref, prefix, resolv)
		}
		log.Debugf("image tar: creating %s", name)
		if err := tw.WriteHeader(&hdr); err != nil {
			return err
		}
		if hdr.Size > 0 {
			if _, err = tw.Write([]byte(contents)); err != nil {
				return err
			}
		}
	}

	// save the sbom to the sbom writer
	if opts.SbomGenerator != nil {
		sboms, err := src.SBoMs()
		if err != nil {
			return err
		}
		for _, sbom := range sboms {
			// sbomWriter will escape out any problematic characters for us
			if err := opts.SbomGenerator.Add(prefix, sbom); err != nil {
				return err
			}
		}
	}

	return nil
}

// ImageBundle produces an OCI bundle at the given path in a tarball, given an image and a config.json
func ImageBundle(prefix, location string, ref *reference.Spec, config []byte, runtime moby.Runtime, tw tarWriter, readonly bool, dupMap map[string]string, opts BuildOpts) error { // nolint: lll
	// if read only, just unpack in rootfs/ but otherwise set up for overlay
	rootExtract := "rootfs"
	if !readonly {
		rootExtract = "lower"
	}

	// See if we have extracted this image previously
	root := path.Join(prefix, rootExtract)
	var foundElsewhere = dupMap[ref.String()] != ""
	if !foundElsewhere {
		if err := ImageTar(location, ref, root+"/", tw, "", opts); err != nil {
			return err
		}
		dupMap[ref.String()] = root
	} else {
		if err := tarPrefix(prefix+"/", location, ref, tw); err != nil {
			return err
		}
		root = dupMap[ref.String()]
	}

	hdr := &tar.Header{
		Name:    path.Join(prefix, "config.json"),
		Mode:    0644,
		Size:    int64(len(config)),
		ModTime: defaultModTime,
		Format:  tar.FormatPAX,
		PAXRecords: map[string]string{
			moby.PaxRecordLinuxkitSource:   ref.String(),
			moby.PaxRecordLinuxkitLocation: location,
		},
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write(config); err != nil {
		return err
	}

	var rootfsMounts []specs.Mount
	if !readonly {
		// add a tmp directory to be used as a mount point for tmpfs for upper, work
		tmp := path.Join(prefix, "tmp")
		hdr = &tar.Header{
			Name:     tmp,
			Mode:     0755,
			Typeflag: tar.TypeDir,
			ModTime:  defaultModTime,
			Format:   tar.FormatPAX,
			PAXRecords: map[string]string{
				moby.PaxRecordLinuxkitSource:   ref.String(),
				moby.PaxRecordLinuxkitLocation: location,
			},
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		// add rootfs as merged mount point
		hdr = &tar.Header{
			Name:     path.Join(prefix, "rootfs"),
			Mode:     0755,
			Typeflag: tar.TypeDir,
			ModTime:  defaultModTime,
			Format:   tar.FormatPAX,
			PAXRecords: map[string]string{
				moby.PaxRecordLinuxkitSource:   ref.String(),
				moby.PaxRecordLinuxkitLocation: location,
			},
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		overlayOptions := []string{"lowerdir=/" + root, "upperdir=/" + path.Join(tmp, "upper"), "workdir=/" + path.Join(tmp, "work")}
		rootfsMounts = []specs.Mount{
			{Source: "tmpfs", Type: "tmpfs", Destination: "/" + tmp},
			// remount private as nothing else should see the temporary layers
			{Destination: "/" + tmp, Options: []string{"remount", "private"}},
			{Source: "overlay", Type: "overlay", Destination: "/" + path.Join(prefix, "rootfs"), Options: overlayOptions},
		}
	} else {
		if foundElsewhere {
			// we need to make the mountpoint at rootfs
			hdr = &tar.Header{
				Name:     path.Join(prefix, "rootfs"),
				Mode:     0755,
				Typeflag: tar.TypeDir,
				ModTime:  defaultModTime,
				Format:   tar.FormatPAX,
				PAXRecords: map[string]string{
					moby.PaxRecordLinuxkitSource:   ref.String(),
					moby.PaxRecordLinuxkitLocation: location,
				},
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
		}
		// either bind from another location, or bind from self to make sure it is a mountpoint as runc prefers this
		rootfsMounts = []specs.Mount{
			{Source: "/" + root, Destination: "/" + path.Join(prefix, "rootfs"), Options: []string{"bind"}},
		}
	}

	// Prepend the rootfs onto the user specified mounts.
	runtimeMounts := append(rootfsMounts, *runtime.Mounts...)
	runtime.Mounts = &runtimeMounts

	// write the runtime config
	runtimeConfig, err := json.MarshalIndent(runtime, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to create runtime config for %s: %v", ref, err)
	}

	hdr = &tar.Header{
		Name:    path.Join(prefix, "runtime.json"),
		Mode:    0644,
		Size:    int64(len(runtimeConfig)),
		ModTime: defaultModTime,
		Format:  tar.FormatPAX,
		PAXRecords: map[string]string{
			moby.PaxRecordLinuxkitSource:   ref.String(),
			moby.PaxRecordLinuxkitLocation: location,
		},
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	if _, err := tw.Write(runtimeConfig); err != nil {
		return err
	}

	log.Debugf("image bundle: %s %s cfg: %s runtime: %s", prefix, ref, string(config), string(runtimeConfig))

	return nil
}
