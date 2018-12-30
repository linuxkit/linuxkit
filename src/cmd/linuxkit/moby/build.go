package moby

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

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

ENTRYPOINT ["/bin/rc.init"]
`

// For now this is a constant that we use in init section only to make
// resolv.conf point at somewhere writeable. In future whe we are not using
// Docker to extract images we can read this directly from image, but now Docker
// will overwrite anything we put in the image.
const resolvconfSymlink = "/run/resolvconf/resolv.conf"

var additions = map[string]addFun{
	"docker": func(tw *tar.Writer) error {
		log.Infof("  Adding Dockerfile")
		hdr := &tar.Header{
			Name:    "Dockerfile",
			Mode:    0644,
			Size:    int64(len(dockerfile)),
			ModTime: defaultModTime,
			Format:  tar.FormatPAX,
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

func outputImage(image *Image, section string, prefix string, m Moby, idMap map[string]uint32, dupMap map[string]string, pull bool, iw *tar.Writer) error {
	log.Infof("  Create OCI config for %s", image.Image)
	useTrust := enforceContentTrust(image.Image, &m.Trust)
	oci, runtime, err := ConfigToOCI(image, useTrust, idMap)
	if err != nil {
		return fmt.Errorf("Failed to create OCI spec for %s: %v", image.Image, err)
	}
	config, err := json.MarshalIndent(oci, "", "    ")
	if err != nil {
		return fmt.Errorf("Failed to create config for %s: %v", image.Image, err)
	}
	path := path.Join("containers", section, prefix+image.Name)
	readonly := oci.Root.Readonly
	err = ImageBundle(path, image.ref, config, runtime, iw, useTrust, pull, readonly, dupMap)
	if err != nil {
		return fmt.Errorf("Failed to extract root filesystem for %s: %v", image.Image, err)
	}
	return nil
}

// Build performs the actual build process
func Build(m Moby, w io.Writer, pull bool, tp string, decompressKernel bool) error {
	if MobyDir == "" {
		MobyDir = defaultMobyConfigDir()
	}

	// create tmp dir in case needed
	if err := os.MkdirAll(filepath.Join(MobyDir, "tmp"), 0755); err != nil {
		return err
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
	for _, image := range m.Onshutdown {
		idMap[image.Name] = id
		id++
	}
	for _, image := range m.Services {
		idMap[image.Name] = id
		id++
	}

	// deduplicate containers with the same image
	dupMap := map[string]string{}

	if m.Kernel.ref != nil {
		// get kernel and initrd tarball and ucode cpio archive from container
		log.Infof("Extract kernel image: %s", m.Kernel.ref)
		kf := newKernelFilter(iw, m.Kernel.Cmdline, m.Kernel.Binary, m.Kernel.Tar, m.Kernel.UCode, decompressKernel)
		err := ImageTar(m.Kernel.ref, "", kf, enforceContentTrust(m.Kernel.ref.String(), &m.Trust), pull, "")
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
	for _, ii := range m.initRefs {
		log.Infof("Process init image: %s", ii)
		err := ImageTar(ii, "", iw, enforceContentTrust(ii.String(), &m.Trust), pull, resolvconfSymlink)
		if err != nil {
			return fmt.Errorf("Failed to build init tarball from %s: %v", ii, err)
		}
	}

	if len(m.Onboot) != 0 {
		log.Infof("Add onboot containers:")
	}
	for i, image := range m.Onboot {
		so := fmt.Sprintf("%03d", i)
		if err := outputImage(image, "onboot", so+"-", m, idMap, dupMap, pull, iw); err != nil {
			return err
		}
	}

	if len(m.Onshutdown) != 0 {
		log.Infof("Add onshutdown containers:")
	}
	for i, image := range m.Onshutdown {
		so := fmt.Sprintf("%03d", i)
		if err := outputImage(image, "onshutdown", so+"-", m, idMap, dupMap, pull, iw); err != nil {
			return err
		}
	}

	if len(m.Services) != 0 {
		log.Infof("Add service containers:")
	}
	for _, image := range m.Services {
		if err := outputImage(image, "services", "", m, idMap, dupMap, pull, iw); err != nil {
			return err
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
	tw               *tar.Writer
	buffer           *bytes.Buffer
	hdr              *tar.Header
	cmdline          string
	kernel           string
	tar              string
	ucode            string
	decompressKernel bool
	discard          bool
	foundKernel      bool
	foundKTar        bool
	foundUCode       bool
}

func newKernelFilter(tw *tar.Writer, cmdline string, kernel string, tar, ucode *string, decompressKernel bool) *kernelFilter {
	tarName, kernelName, ucodeName := "kernel.tar", "kernel", ""
	if tar != nil {
		tarName = *tar
		if tarName == "none" {
			tarName = ""
		}
	}
	if kernel != "" {
		kernelName = kernel
	}
	if ucode != nil {
		ucodeName = *ucode
	}
	return &kernelFilter{tw: tw, cmdline: cmdline, kernel: kernelName, tar: tarName, ucode: ucodeName, decompressKernel: decompressKernel}
}

func (k *kernelFilter) finishTar() error {
	if k.buffer == nil {
		return nil
	}

	if k.hdr != nil {
		if k.decompressKernel {
			log.Debugf("Decompressing kernel")
			b, err := decompressKernel(k.buffer)
			if err != nil {
				return err
			}
			k.buffer = b
			k.hdr.Size = int64(k.buffer.Len())
		}

		if err := k.tw.WriteHeader(k.hdr); err != nil {
			return err
		}
		if _, err := k.tw.Write(k.buffer.Bytes()); err != nil {
			return err
		}
		k.hdr = nil
		k.buffer = nil
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
		// If we handled the ucode, /boot already exist.
		if !k.foundUCode {
			whdr := &tar.Header{
				Name:     "boot",
				Mode:     0755,
				Typeflag: tar.TypeDir,
				ModTime:  defaultModTime,
				Format:   tar.FormatPAX,
			}
			if err := tw.WriteHeader(whdr); err != nil {
				return err
			}
		}
		// add the cmdline in /boot/cmdline
		whdr := &tar.Header{
			Name:    "boot/cmdline",
			Mode:    0644,
			Size:    int64(len(k.cmdline)),
			ModTime: defaultModTime,
			Format:  tar.FormatPAX,
		}
		if err := tw.WriteHeader(whdr); err != nil {
			return err
		}
		buf := bytes.NewBufferString(k.cmdline)
		_, err = io.Copy(tw, buf)
		if err != nil {
			return err
		}
		// Stash the kernel header and prime the buffer for the kernel
		k.hdr = &tar.Header{
			Name:    "boot/kernel",
			Mode:    hdr.Mode,
			Size:    hdr.Size,
			ModTime: defaultModTime,
			Format:  tar.FormatPAX,
		}
		k.buffer = new(bytes.Buffer)
	case k.tar:
		k.foundKTar = true
		k.discard = false
		k.buffer = new(bytes.Buffer)
	case k.ucode:
		k.foundUCode = true
		k.discard = false
		// If we handled the kernel, /boot already exist.
		if !k.foundKernel {
			whdr := &tar.Header{
				Name:     "boot",
				Mode:     0755,
				Typeflag: tar.TypeDir,
				ModTime:  defaultModTime,
				Format:   tar.FormatPAX,
			}
			if err := tw.WriteHeader(whdr); err != nil {
				return err
			}
		}
		whdr := &tar.Header{
			Name:    "boot/ucode.cpio",
			Mode:    hdr.Mode,
			Size:    hdr.Size,
			ModTime: defaultModTime,
			Format:  tar.FormatPAX,
		}
		if err := tw.WriteHeader(whdr); err != nil {
			return err
		}
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

// Attempt to decompress a Linux kernel image
// The kernel image can be a plain gzip'ed image (e.g., the LinuxKit arm64 kernel) or a bzImage (x86)
// or not compressed at all (e.g., s390x). This function tries to detect the image type and decompress
// the kernel. If no supported compressed kernel is found it returns an error.
// For bzImages it performs some sanity checks on the header and currently only supports gzip'ed bzImages.
func decompressKernel(src *bytes.Buffer) (*bytes.Buffer, error) {
	const gzipMagic = "\037\213"

	s := src.Bytes()

	if bytes.HasPrefix(s, []byte(gzipMagic)) {
		log.Debugf("Found gzip signature at offset: 0")
		return gunzip(src)
	}

	// Check if it is a bzImage
	// See: https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/Documentation/x86/boot.txt
	const bzMagicIdx = 0x1fe
	const bzMagic = uint16(0xaa55)
	const bzHeaderIdx = 0x202
	const bzHeader = "HdrS"
	const bzMinLen = 0x250 // Minimum length for the required 2.0.8+ header
	if len(s) > bzMinLen &&
		binary.LittleEndian.Uint16(s[bzMagicIdx:bzMagicIdx+2]) == bzMagic &&
		bytes.HasPrefix(s[bzHeaderIdx:], []byte(bzHeader)) {

		log.Debugf("Found bzImage Magic and Header")

		const versionIdx = 0x206
		const setupSectorsIdx = 0x1f1
		const sectorSize = 512
		const payloadIdx = 0x248
		const payloadLengthIdx = 0x24c

		// Check that the version is 2.08+
		versionMajor := int(s[versionIdx])
		versionMinor := int(s[versionIdx+1])
		if versionMajor < 2 && versionMinor < 8 {
			return nil, fmt.Errorf("Unsupported bzImage version: %d.%d", versionMajor, versionMinor)
		}

		setupSectors := uint32(s[setupSectorsIdx])
		payloadOff := binary.LittleEndian.Uint32(s[payloadIdx : payloadIdx+4])
		payloadLen := binary.LittleEndian.Uint32(s[payloadLengthIdx : payloadLengthIdx+4])
		payloadOff += (setupSectors + 1) * sectorSize
		log.Debugf("bzImage: Payload at Offset: %d Length: %d", payloadOff, payloadLen)

		if len(s) < int(payloadOff+payloadLen) {
			return nil, fmt.Errorf("Compressed bzImage payload exceeds size of image")
		}

		if bytes.HasPrefix(s[payloadOff:], []byte(gzipMagic)) {
			log.Debugf("bzImage: gzip signature at offset: %d", payloadOff)
			return gunzip(bytes.NewBuffer(s[payloadOff : payloadOff+payloadLen]))
		}
		// TODO(rn): Add more supported formats
		return nil, fmt.Errorf("Unsupported bzImage payload format at offset %d", payloadOff)
	}

	return nil, fmt.Errorf("No compressed kernel or no supported format found")
}

func gunzip(src *bytes.Buffer) (*bytes.Buffer, error) {
	dst := new(bytes.Buffer)

	zr, err := gzip.NewReader(src)
	if err != nil {
		return nil, err
	}

	n, err := io.Copy(dst, zr)
	if err != nil {
		return nil, err
	}

	log.Debugf("gunzip'ed %d bytes", n)
	return dst, nil
}

// this allows inserting metadata into a file in the image
func metadata(m Moby, md string) ([]byte, error) {
	// Make sure the Image strings are update to date with the refs
	updateImages(&m)
	switch md {
	case "json":
		return json.MarshalIndent(m, "", "    ")
	case "yaml":
		return yaml.Marshal(m)
	default:
		return []byte{}, fmt.Errorf("Unsupported metadata type: %s", md)
	}
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
		if f.Path[0] == '/' {
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
		if !f.Directory && f.Symlink == "" && f.Contents == nil {
			if f.Source == "" && f.Metadata == "" {
				return fmt.Errorf("Contents of file (%s) not specified", f.Path)
			}
			if f.Source != "" && f.Metadata != "" {
				return fmt.Errorf("Specified Source and Metadata for file: %s", f.Path)
			}
			if f.Source != "" {
				source := f.Source
				if len(source) > 2 && source[:2] == "~/" {
					source = homeDir() + source[1:]
				}
				if f.Optional {
					_, err := os.Stat(source)
					if err != nil {
						// skip if not found or readable
						log.Debugf("Skipping file [%s] as not readable and marked optional", source)
						continue
					}
				}
				var err error
				contents, err = ioutil.ReadFile(source)
				if err != nil {
					return err
				}
			} else {
				contents, err = metadata(m, f.Metadata)
				if err != nil {
					return err
				}
			}
		} else {
			if f.Metadata != "" {
				return fmt.Errorf("Specified Contents and Metadata for file: %s", f.Path)
			}
			if f.Source != "" {
				return fmt.Errorf("Specified Contents and Source for file: %s", f.Path)
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
					ModTime:  defaultModTime,
					Uid:      int(uid),
					Gid:      int(gid),
					Format:   tar.FormatPAX,
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
			Name:    f.Path,
			Mode:    mode,
			ModTime: defaultModTime,
			Uid:     int(uid),
			Gid:     int(gid),
			Format:  tar.FormatPAX,
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
