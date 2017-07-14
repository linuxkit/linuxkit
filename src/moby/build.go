package moby

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
)

const defaultNameForStdin = "moby"

var streamable = map[string]bool{
	"docker": true,
	"tar":    true,
}

// Streamable returns true if an output can be streamed
func Streamable(t string) bool {
	return streamable[t]
}

type addFun func(*tar.Writer) error

const dockerfile = `
FROM scratch

COPY . ./
RUN rm -f Dockerfile

ENTRYPOINT ["/sbin/tini", "--", "/bin/rc.init"]
`

var additions = map[string]addFun{
	"docker": func(tw *tar.Writer) error {
		log.Infof("  Adding Dockerfile")
		hdr := &tar.Header{
			Name: "Dockerfile",
			Mode: 0644,
			Size: int64(len(dockerfile)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write([]byte(dockerfile)); err != nil {
			return err
		}
		return nil
	},
}

// OutputTypes returns a list of the valid output types
func OutputTypes() []string {
	ts := []string{}
	for k := range streamable {
		ts = append(ts, k)
	}
	for k := range outFuns {
		ts = append(ts, k)
	}
	sort.Strings(ts)

	return ts
}

func enforceContentTrust(fullImageName string, config *TrustConfig) bool {
	for _, img := range config.Image {
		// First check for an exact name match
		if img == fullImageName {
			return true
		}
		// Also check for an image name only match
		// by removing a possible tag (with possibly added digest):
		imgAndTag := strings.Split(fullImageName, ":")
		if len(imgAndTag) >= 2 && img == imgAndTag[0] {
			return true
		}
		// and by removing a possible digest:
		imgAndDigest := strings.Split(fullImageName, "@sha256:")
		if len(imgAndDigest) >= 2 && img == imgAndDigest[0] {
			return true
		}
	}

	for _, org := range config.Org {
		var imgOrg string
		splitName := strings.Split(fullImageName, "/")
		switch len(splitName) {
		case 0:
			// if the image is empty, return false
			return false
		case 1:
			// for single names like nginx, use library
			imgOrg = "library"
		case 2:
			// for names that assume docker hub, like linxukit/alpine, take the first split
			imgOrg = splitName[0]
		default:
			// for names that include the registry, the second piece is the org, ex: docker.io/library/alpine
			imgOrg = splitName[1]
		}
		if imgOrg == org {
			return true
		}
	}
	return false
}

// Build performs the actual build process
func Build(m Moby, w io.Writer, pull bool, tp string) error {
	if MobyDir == "" {
		return fmt.Errorf("MobyDir for temporary storage not set")
	}

	iw := tar.NewWriter(w)

	// add additions
	addition := additions[tp]

	// allocate each container a uid, gid that can be referenced by name
	idMap := map[string]uint32{}
	id := uint32(100)
	for _, image := range m.Onboot {
		idMap[image.Name] = id
		id++
	}
	for _, image := range m.Services {
		idMap[image.Name] = id
		id++
	}

	if m.Kernel.Image != "" {
		// get kernel and initrd tarball from container
		log.Infof("Extract kernel image: %s", m.Kernel.Image)
		kf := newKernelFilter(iw, m.Kernel.Cmdline, m.Kernel.Binary, m.Kernel.Tar)
		err := ImageTar(m.Kernel.Image, "", kf, enforceContentTrust(m.Kernel.Image, &m.Trust), pull)
		if err != nil {
			return fmt.Errorf("Failed to extract kernel image and tarball: %v", err)
		}
		err = kf.Close()
		if err != nil {
			return fmt.Errorf("Close error: %v", err)
		}
	}

	// convert init images to tarballs
	if len(m.Init) != 0 {
		log.Infof("Add init containers:")
	}
	for _, ii := range m.Init {
		log.Infof("Process init image: %s", ii)
		err := ImageTar(ii, "", iw, enforceContentTrust(ii, &m.Trust), pull)
		if err != nil {
			return fmt.Errorf("Failed to build init tarball from %s: %v", ii, err)
		}
	}

	if len(m.Onboot) != 0 {
		log.Infof("Add onboot containers:")
	}
	for i, image := range m.Onboot {
		log.Infof("  Create OCI config for %s", image.Image)
		useTrust := enforceContentTrust(image.Image, &m.Trust)
		config, err := ConfigToOCI(image, useTrust, idMap)
		if err != nil {
			return fmt.Errorf("Failed to create config.json for %s: %v", image.Image, err)
		}
		so := fmt.Sprintf("%03d", i)
		path := "containers/onboot/" + so + "-" + image.Name
		err = ImageBundle(path, image.Image, config, iw, useTrust, pull)
		if err != nil {
			return fmt.Errorf("Failed to extract root filesystem for %s: %v", image.Image, err)
		}
	}

	if len(m.Services) != 0 {
		log.Infof("Add service containers:")
	}
	for _, image := range m.Services {
		log.Infof("  Create OCI config for %s", image.Image)
		useTrust := enforceContentTrust(image.Image, &m.Trust)
		config, err := ConfigToOCI(image, useTrust, idMap)
		if err != nil {
			return fmt.Errorf("Failed to create config.json for %s: %v", image.Image, err)
		}
		path := "containers/services/" + image.Name
		err = ImageBundle(path, image.Image, config, iw, useTrust, pull)
		if err != nil {
			return fmt.Errorf("Failed to extract root filesystem for %s: %v", image.Image, err)
		}
	}

	// add files
	err := filesystem(m, iw, idMap)
	if err != nil {
		return fmt.Errorf("failed to add filesystem parts: %v", err)
	}

	// add anything additional for this output type
	if addition != nil {
		err = addition(iw)
		if err != nil {
			return fmt.Errorf("Failed to add additional files: %v", err)
		}
	}

	err = iw.Close()
	if err != nil {
		return fmt.Errorf("initrd close error: %v", err)
	}

	return nil
}

// kernelFilter is a tar.Writer that transforms a kernel image into the output we want on underlying tar writer
type kernelFilter struct {
	tw          *tar.Writer
	buffer      *bytes.Buffer
	cmdline     string
	kernel      string
	tar         string
	discard     bool
	foundKernel bool
	foundKTar   bool
}

func newKernelFilter(tw *tar.Writer, cmdline string, kernel string, tar *string) *kernelFilter {
	tarName, kernelName := "kernel.tar", "kernel"
	if tar != nil {
		tarName = *tar
		if tarName == "none" {
			tarName = ""
		}
	}
	if kernel != "" {
		kernelName = kernel
	}
	return &kernelFilter{tw: tw, cmdline: cmdline, kernel: kernelName, tar: tarName}
}

func (k *kernelFilter) finishTar() error {
	if k.buffer == nil {
		return nil
	}
	tr := tar.NewReader(k.buffer)
	err := tarAppend(k.tw, tr)
	k.buffer = nil
	return err
}

func (k *kernelFilter) Close() error {
	if !k.foundKernel {
		return errors.New("did not find kernel in kernel image")
	}
	if !k.foundKTar && k.tar != "" {
		return errors.New("did not find kernel tar in kernel image")
	}
	return k.finishTar()
}

func (k *kernelFilter) Flush() error {
	err := k.finishTar()
	if err != nil {
		return err
	}
	return k.tw.Flush()
}

func (k *kernelFilter) Write(b []byte) (n int, err error) {
	if k.discard {
		return len(b), nil
	}
	if k.buffer != nil {
		return k.buffer.Write(b)
	}
	return k.tw.Write(b)
}

func (k *kernelFilter) WriteHeader(hdr *tar.Header) error {
	err := k.finishTar()
	if err != nil {
		return err
	}
	tw := k.tw
	switch hdr.Name {
	case k.kernel:
		if k.foundKernel {
			return errors.New("found more than one possible kernel image")
		}
		k.foundKernel = true
		k.discard = false
		whdr := &tar.Header{
			Name:     "boot",
			Mode:     0755,
			Typeflag: tar.TypeDir,
		}
		if err := tw.WriteHeader(whdr); err != nil {
			return err
		}
		// add the cmdline in /boot/cmdline
		whdr = &tar.Header{
			Name: "boot/cmdline",
			Mode: 0644,
			Size: int64(len(k.cmdline)),
		}
		if err := tw.WriteHeader(whdr); err != nil {
			return err
		}
		buf := bytes.NewBufferString(k.cmdline)
		_, err = io.Copy(tw, buf)
		if err != nil {
			return err
		}
		whdr = &tar.Header{
			Name: "boot/kernel",
			Mode: hdr.Mode,
			Size: hdr.Size,
		}
		if err := tw.WriteHeader(whdr); err != nil {
			return err
		}
	case k.tar:
		k.foundKTar = true
		k.discard = false
		k.buffer = new(bytes.Buffer)
	default:
		k.discard = true
	}

	return nil
}

func tarAppend(iw *tar.Writer, tr *tar.Reader) error {
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		err = iw.WriteHeader(hdr)
		if err != nil {
			return err
		}
		_, err = io.Copy(iw, tr)
		if err != nil {
			return err
		}
	}
	return nil
}

func filesystem(m Moby, tw *tar.Writer, idMap map[string]uint32) error {
	// TODO also include the files added in other parts of the build
	var addedFiles = map[string]bool{}

	if len(m.Files) != 0 {
		log.Infof("Add files:")
	}
	for _, f := range m.Files {
		log.Infof("  %s", f.Path)
		if f.Path == "" {
			return errors.New("Did not specify path for file")
		}
		// tar archives should not have absolute paths
		if f.Path[0] == os.PathSeparator {
			f.Path = f.Path[1:]
		}
		mode := int64(0600)
		if f.Directory {
			mode = 0700
		}
		if f.Mode != "" {
			var err error
			mode, err = strconv.ParseInt(f.Mode, 8, 32)
			if err != nil {
				return fmt.Errorf("Cannot parse file mode as octal value: %v", err)
			}
		}
		dirMode := mode
		if dirMode&0700 != 0 {
			dirMode |= 0100
		}
		if dirMode&0070 != 0 {
			dirMode |= 0010
		}
		if dirMode&0007 != 0 {
			dirMode |= 0001
		}

		uid, err := idNumeric(f.UID, idMap)
		if err != nil {
			return err
		}
		gid, err := idNumeric(f.GID, idMap)
		if err != nil {
			return err
		}

		var contents []byte
		if f.Contents != nil {
			contents = []byte(*f.Contents)
		}
		if !f.Directory && f.Contents == nil && f.Symlink == "" {
			if f.Source == "" {
				return errors.New("Contents of file not specified")
			}
			if len(f.Source) > 2 && f.Source[:2] == "~/" {
				f.Source = homeDir() + f.Source[1:]
			}
			if f.Optional {
				_, err := os.Stat(f.Source)
				if err != nil {
					// skip if not found or readable
					log.Debugf("Skipping file [%s] as not readable and marked optional", f.Source)
					continue
				}
			}
			var err error
			contents, err = ioutil.ReadFile(f.Source)
			if err != nil {
				return err
			}
		}
		// we need all the leading directories
		parts := strings.Split(path.Dir(f.Path), "/")
		root := ""
		for _, p := range parts {
			if p == "." || p == "/" {
				continue
			}
			if root == "" {
				root = p
			} else {
				root = root + "/" + p
			}
			if !addedFiles[root] {
				hdr := &tar.Header{
					Name:     root,
					Typeflag: tar.TypeDir,
					Mode:     dirMode,
					Uid:      int(uid),
					Gid:      int(gid),
				}
				err := tw.WriteHeader(hdr)
				if err != nil {
					return err
				}
				addedFiles[root] = true
			}
		}
		addedFiles[f.Path] = true
		hdr := &tar.Header{
			Name: f.Path,
			Mode: mode,
			Uid:  int(uid),
			Gid:  int(gid),
		}
		if f.Directory {
			if f.Contents != nil {
				return errors.New("Directory with contents not allowed")
			}
			hdr.Typeflag = tar.TypeDir
			err := tw.WriteHeader(hdr)
			if err != nil {
				return err
			}
		} else if f.Symlink != "" {
			hdr.Typeflag = tar.TypeSymlink
			hdr.Linkname = f.Symlink
			err := tw.WriteHeader(hdr)
			if err != nil {
				return err
			}
		} else {
			hdr.Size = int64(len(contents))
			err := tw.WriteHeader(hdr)
			if err != nil {
				return err
			}
			_, err = tw.Write(contents)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
