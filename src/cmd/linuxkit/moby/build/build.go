package build

import (
	"archive/tar"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/containerd/containerd/v2/pkg/reference"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	imagespec "github.com/opencontainers/image-spec/specs-go/v1"

	// drop-in 100% compatible replacement and 17% faster than compress/gzip.
	gzip "github.com/klauspost/pgzip"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/moby"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
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
	var ts []string
	for k := range streamable {
		ts = append(ts, k)
	}
	for k := range outFuns {
		ts = append(ts, k)
	}
	sort.Strings(ts)

	return ts
}

// outputImage given an image and a section, such as onboot, onshutdown or services, lay it out with correct location
// config, etc. in the filesystem, so runc can use it.
func outputImage(image *moby.Image, section string, index int, prefix string, m moby.Moby, idMap map[string]uint32, dupMap map[string]string, iw *tar.Writer, opts BuildOpts) error {
	log.Infof("  Create OCI config for %s", image.Image)
	imageName := util.ReferenceExpand(image.Image)
	ref, err := reference.Parse(imageName)
	if err != nil {
		return fmt.Errorf("could not resolve references for image %s: %v", image.Image, err)
	}
	src, err := imageSource(&ref, opts.Pull, opts.CacheDir, opts.DockerCache, imagespec.Platform{OS: "linux", Architecture: opts.Arch})
	if err != nil {
		return fmt.Errorf("could not pull image %s: %v", image.Image, err)
	}
	configRaw, err := src.Config()
	if err != nil {
		return fmt.Errorf("failed to retrieve config for %s: %v", image.Image, err)
	}
	// use a modified version of onboot which replaces volume names with paths
	imageWithVolPaths, err := updateMountsAndBindsFromVolumes(image, m)
	if err != nil {
		return fmt.Errorf("failed update image %s from volumes: %w", image.Image, err)
	}

	oci, runtime, err := moby.ConfigToOCI(imageWithVolPaths, configRaw, idMap)
	if err != nil {
		return fmt.Errorf("failed to create OCI spec for %s: %v", image.Image, err)
	}
	config, err := json.MarshalIndent(oci, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to create config for %s: %v", image.Image, err)
	}
	path := path.Join("containers", section, prefix+image.Name)
	readonly := oci.Root.Readonly
	err = ImageBundle(path, fmt.Sprintf("%s[%d]", section, index), image.Ref(), config, runtime, iw, readonly, dupMap, opts)
	if err != nil {
		return fmt.Errorf("failed to extract root filesystem for %s: %v", image.Image, err)
	}
	return nil
}

// Build performs the actual build process. The output is the filesystem
// in a tar stream written to w.
func Build(m moby.Moby, w io.Writer, opts BuildOpts) error {
	if MobyDir == "" {
		MobyDir = defaultMobyConfigDir()
	}

	// create tmp dir in case needed
	if err := os.MkdirAll(filepath.Join(MobyDir, "tmp"), 0755); err != nil {
		return err
	}

	// find the Moby config file from the existing tar
	var metadataLocation string
	if m.Files != nil {
		for _, f := range m.Files {
			if f.Metadata == "" {
				continue
			}
			metadataLocation = strings.TrimPrefix(f.Path, "/")
		}
	}
	var (
		oldConfig *moby.Moby
		in        *os.File
		err       error
	)
	if metadataLocation != "" && opts.InputTar != "" {
		// copy the file over, in case it ends up being the same output
		in, err = os.Open(opts.InputTar)
		if err != nil {
			return fmt.Errorf("failed to open input tar: %w", err)
		}
		defer in.Close()
		if _, err := in.Seek(0, 0); err != nil {
			return fmt.Errorf("failed to seek to beginning of tmpfile: %w", err)
		}
		// read the tar until we find the metadata file
		inputTarReader := tar.NewReader(in)
		for {
			hdr, err := inputTarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("failed to read input tar: %w", err)
			}
			if strings.TrimPrefix(hdr.Name, "/") == metadataLocation {
				buf := new(bytes.Buffer)
				if _, err := buf.ReadFrom(inputTarReader); err != nil {
					return fmt.Errorf("failed to read metadata file from input tar: %w", err)
				}
				config, err := moby.NewConfig(buf.Bytes(), nil)
				if err != nil {
					return fmt.Errorf("invalid config in existing tar file: %v", err)
				}
				oldConfig = &config
				break
			}
		}
	}

	// do we have an inTar
	iw := tar.NewWriter(w)

	// add additions
	addition := additions[opts.BuilderType]

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

	kernelRef := m.Kernel.Ref()
	var oldKernelRef *reference.Spec
	if oldConfig != nil {
		oldKernelRef = oldConfig.Kernel.Ref()
	}
	if kernelRef != nil {
		// first check if the existing one had it
		if oldKernelRef != nil && oldKernelRef.String() == kernelRef.String() {
			if err := extractPackageFilesFromTar(in, iw, kernelRef.String(), "kernel"); err != nil {
				return err
			}
		} else {
			// get kernel and initrd tarball and ucode cpio archive from container
			log.Infof("Extract kernel image: %s", m.Kernel.Ref())
			kf := newKernelFilter(kernelRef, iw, m.Kernel.Cmdline, m.Kernel.Binary, m.Kernel.Tar, m.Kernel.UCode, opts.DecompressKernel)
			err := ImageTar("kernel", kernelRef, "", kf, "", opts)
			if err != nil {
				return fmt.Errorf("failed to extract kernel image and tarball: %v", err)
			}
			err = kf.Close()
			if err != nil {
				return fmt.Errorf("close error: %v", err)
			}
		}
	}

	// convert init images to tarballs
	if len(m.Init) != 0 {
		log.Infof("Add init containers:")
	}
	apkTar := moby.NewAPKTarWriter(iw, "init")
	initRefs := m.InitRefs()
	var oldInitRefs []*reference.Spec
	if oldConfig != nil {
		oldInitRefs = oldConfig.InitRefs()
	}
	for i, ii := range initRefs {
		if len(oldInitRefs) > i && oldInitRefs[i].String() == ii.String() {
			if err := extractPackageFilesFromTar(in, apkTar, ii.String(), fmt.Sprintf("init[%d]", i)); err != nil {
				return err
			}
		} else {
			log.Infof("Process init image: %s", ii)
			err := ImageTar(fmt.Sprintf("init[%d]", i), ii, "", apkTar, resolvconfSymlink, opts)
			if err != nil {
				return fmt.Errorf("failed to build init tarball from %s: %v", ii, err)
			}
		}
	}
	if err := apkTar.WriteAPKDB(); err != nil {
		return err
	}

	if len(m.Volumes) != 0 {
		log.Infof("Add volumes:")
	}

	for i, vol := range m.Volumes {
		log.Infof("Process volume image: %s", vol.Name)
		// there is an Image, so we need to extract it, either from inputTar or from the image
		if oldConfig != nil && len(oldConfig.Volumes) > i && oldConfig.Volumes[i].Image == vol.Image {
			if err := extractPackageFilesFromTar(in, iw, vol.Image, fmt.Sprintf("volumes[%d]", i)); err != nil {
				return err
			}
			continue
		}
		location := fmt.Sprintf("volume[%d]", i)
		lower, tmpDir, merged := vol.LowerDir(), vol.TmpDir(), vol.MergedDir()
		lowerPath := strings.TrimPrefix(lower, "/") + "/"

		// get volume tarball from container
		switch {
		case vol.ImageRef() == nil || vol.Format == "" || vol.Format == "filesystem":
			if err := ImageTar(location, vol.ImageRef(), lowerPath, apkTar, resolvconfSymlink, opts); err != nil {
				return fmt.Errorf("failed to build volume filesystem tarball from %s: %v", vol.Name, err)
			}
		case vol.Format == "oci":
			// convert platforms into imagespec platforms
			platforms := make([]imagespec.Platform, len(vol.Platforms))
			for i, p := range vol.Platforms {
				platform, err := v1.ParsePlatform(p)
				if err != nil {
					return fmt.Errorf("failed to parse platform %s: %v", p, err)
				}
				platforms[i] = imagespec.Platform{
					Architecture: platform.Architecture,
					OS:           platform.OS,
					Variant:      platform.Variant,
				}
			}
			if err := ImageOCITar(location, vol.ImageRef(), lowerPath, apkTar, opts, platforms); err != nil {
				return fmt.Errorf("failed to build volume OCI v1 layout tarball from %s: %v", vol.Name, err)
			}
		}

		// make upper and merged dirs which will be used for mounting
		// no need to make lower dir, as it is made automatically by ImageTar()
		tmpPath := strings.TrimPrefix(tmpDir, "/") + "/"
		tmphdr := &tar.Header{
			Name:     tmpPath,
			Mode:     0755,
			Typeflag: tar.TypeDir,
			ModTime:  defaultModTime,
			Format:   tar.FormatPAX,
			PAXRecords: map[string]string{
				moby.PaxRecordLinuxkitSource:   "linuxkit.volumes",
				moby.PaxRecordLinuxkitLocation: location,
			},
		}
		if err := apkTar.WriteHeader(tmphdr); err != nil {
			return err
		}
		mergedPath := strings.TrimPrefix(merged, "/") + "/"
		mhdr := &tar.Header{
			Name:     mergedPath,
			Mode:     0755,
			Typeflag: tar.TypeDir,
			ModTime:  defaultModTime,
			Format:   tar.FormatPAX,
			PAXRecords: map[string]string{
				moby.PaxRecordLinuxkitSource:   "linuxkit.volumes",
				moby.PaxRecordLinuxkitLocation: location,
			},
		}
		if err := apkTar.WriteHeader(mhdr); err != nil {
			return err
		}
	}

	if len(m.Onboot) != 0 {
		log.Infof("Add onboot containers:")
	}
	for i, image := range m.Onboot {
		if oldConfig != nil && len(oldConfig.Onboot) > i && oldConfig.Onboot[i].Equal(image) {
			if err := extractPackageFilesFromTar(in, iw, image.Image, fmt.Sprintf("onboot[%d]", i)); err != nil {
				return err
			}
		} else {
			so := fmt.Sprintf("%03d", i)
			if err := outputImage(image, "onboot", i, so+"-", m, idMap, dupMap, iw, opts); err != nil {
				return err
			}
		}
	}

	if len(m.Onshutdown) != 0 {
		log.Infof("Add onshutdown containers:")
	}
	for i, image := range m.Onshutdown {
		if oldConfig != nil && len(oldConfig.Onshutdown) > i && oldConfig.Onshutdown[i].Equal(image) {
			if err := extractPackageFilesFromTar(in, iw, image.Image, fmt.Sprintf("onshutdown[%d]", i)); err != nil {
				return err
			}
		} else {
			so := fmt.Sprintf("%03d", i)
			if err := outputImage(image, "onshutdown", i, so+"-", m, idMap, dupMap, iw, opts); err != nil {
				return err
			}
		}
	}

	if len(m.Services) != 0 {
		log.Infof("Add service containers:")
	}
	for i, image := range m.Services {
		if oldConfig != nil && len(oldConfig.Services) > i && oldConfig.Services[i].Equal(image) {
			if err := extractPackageFilesFromTar(in, iw, image.Image, fmt.Sprintf("services[%d]", i)); err != nil {
				return err
			}
		} else {
			if err := outputImage(image, "services", i, "", m, idMap, dupMap, iw, opts); err != nil {
				return err
			}
		}
	}

	// add files
	if err := filesystem(m, iw, idMap); err != nil {
		return fmt.Errorf("failed to add filesystem parts: %v", err)
	}

	// add anything additional for this output type
	if addition != nil {
		err = addition(iw)
		if err != nil {
			return fmt.Errorf("failed to add additional files: %v", err)
		}
	}

	// complete the sbom consolidation
	if opts.SbomGenerator != nil {
		if err := opts.SbomGenerator.Close(iw); err != nil {
			return err
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
	ref              *reference.Spec
}

func newKernelFilter(ref *reference.Spec, tw *tar.Writer, cmdline string, kernel string, tar, ucode *string, decompressKernel bool) *kernelFilter {
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
	return &kernelFilter{ref: ref, tw: tw, cmdline: cmdline, kernel: kernelName, tar: tarName, ucode: ucodeName, decompressKernel: decompressKernel}
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
	err := tarAppend(k.ref, k.tw, tr)
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
				Name:       "boot",
				Mode:       0755,
				Typeflag:   tar.TypeDir,
				ModTime:    defaultModTime,
				Format:     tar.FormatPAX,
				PAXRecords: hdr.PAXRecords,
			}
			if err := tw.WriteHeader(whdr); err != nil {
				return err
			}
		}
		// add the cmdline in /boot/cmdline
		whdr := &tar.Header{
			Name:       "boot/cmdline",
			Mode:       0644,
			Size:       int64(len(k.cmdline)),
			ModTime:    defaultModTime,
			Format:     tar.FormatPAX,
			PAXRecords: hdr.PAXRecords,
		}
		if err := tw.WriteHeader(whdr); err != nil {
			return err
		}
		_, err = tw.Write([]byte(k.cmdline))
		if err != nil {
			return err
		}
		// Stash the kernel header and prime the buffer for the kernel
		k.hdr = &tar.Header{
			Name:       "boot/kernel",
			Mode:       hdr.Mode,
			Size:       hdr.Size,
			ModTime:    defaultModTime,
			Format:     tar.FormatPAX,
			PAXRecords: hdr.PAXRecords,
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
				Name:       "boot",
				Mode:       0755,
				Typeflag:   tar.TypeDir,
				ModTime:    defaultModTime,
				Format:     tar.FormatPAX,
				PAXRecords: hdr.PAXRecords,
			}
			if err := tw.WriteHeader(whdr); err != nil {
				return err
			}
		}
		whdr := &tar.Header{
			Name:       "boot/ucode.cpio",
			Mode:       hdr.Mode,
			Size:       hdr.Size,
			ModTime:    defaultModTime,
			Format:     tar.FormatPAX,
			PAXRecords: hdr.PAXRecords,
		}
		if err := tw.WriteHeader(whdr); err != nil {
			return err
		}
	default:
		k.discard = true
	}

	return nil
}

func tarAppend(ref *reference.Spec, iw *tar.Writer, tr *tar.Reader) error {
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		hdr.Format = tar.FormatPAX
		if hdr.PAXRecords == nil {
			hdr.PAXRecords = make(map[string]string)
		}
		hdr.PAXRecords[moby.PaxRecordLinuxkitSource] = ref.String()
		hdr.PAXRecords[moby.PaxRecordLinuxkitLocation] = "kernel"
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
			return nil, fmt.Errorf("unsupported bzImage version: %d.%d", versionMajor, versionMinor)
		}

		setupSectors := uint32(s[setupSectorsIdx])
		payloadOff := binary.LittleEndian.Uint32(s[payloadIdx : payloadIdx+4])
		payloadLen := binary.LittleEndian.Uint32(s[payloadLengthIdx : payloadLengthIdx+4])
		payloadOff += (setupSectors + 1) * sectorSize
		log.Debugf("bzImage: Payload at Offset: %d Length: %d", payloadOff, payloadLen)

		if len(s) < int(payloadOff+payloadLen) {
			return nil, fmt.Errorf("compressed bzImage payload exceeds size of image")
		}

		if bytes.HasPrefix(s[payloadOff:], []byte(gzipMagic)) {
			log.Debugf("bzImage: gzip signature at offset: %d", payloadOff)
			return gunzip(bytes.NewBuffer(s[payloadOff : payloadOff+payloadLen]))
		}
		// TODO(rn): Add more supported formats
		return nil, fmt.Errorf("unsupported bzImage payload format at offset %d", payloadOff)
	}

	return nil, fmt.Errorf("no compressed kernel or no supported format found")
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
func metadata(m moby.Moby, md string) ([]byte, error) {
	// Make sure the Image strings are update to date with the refs
	moby.UpdateImages(&m)
	switch md {
	case "json":
		return json.MarshalIndent(m, "", "    ")
	case "yaml":
		return yaml.Marshal(m)
	default:
		return []byte{}, fmt.Errorf("unsupported metadata type: %s", md)
	}
}

func filesystem(m moby.Moby, tw *tar.Writer, idMap map[string]uint32) error {
	// TODO also include the files added in other parts of the build
	var addedFiles = map[string]bool{}

	if len(m.Files) != 0 {
		log.Infof("Add files:")
	}
	for filecount, f := range m.Files {
		log.Infof("  %s", f.Path)
		if f.Path == "" {
			return errors.New("did not specify path for file")
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
				return fmt.Errorf("cannot parse file mode as octal value: %v", err)
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

		uid, err := moby.IDNumeric(f.UID, idMap)
		if err != nil {
			return err
		}
		gid, err := moby.IDNumeric(f.GID, idMap)
		if err != nil {
			return err
		}

		var contents []byte
		if f.Contents != nil {
			contents = []byte(*f.Contents)
		}
		if !f.Directory && f.Symlink == "" && f.Contents == nil {
			if f.Source == "" && f.Metadata == "" {
				return fmt.Errorf("contents of file (%s) not specified", f.Path)
			}
			if f.Source != "" && f.Metadata != "" {
				return fmt.Errorf("specified Source and Metadata for file: %s", f.Path)
			}
			if f.Source != "" {
				source := f.Source
				if len(source) > 2 && source[:2] == "~/" {
					source = util.HomeDir() + source[1:]
				}
				if f.Optional {
					_, err := os.Stat(source)
					if err != nil {
						// skip if not found or readable
						log.Debugf("skipping file [%s] as not readable and marked optional", source)
						continue
					}
				}
				var err error
				contents, err = os.ReadFile(source)
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
				return fmt.Errorf("specified Contents and Metadata for file: %s", f.Path)
			}
			if f.Source != "" {
				return fmt.Errorf("specified Contents and Source for file: %s", f.Path)
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
					PAXRecords: map[string]string{
						moby.PaxRecordLinuxkitSource:   "linuxkit.files",
						moby.PaxRecordLinuxkitLocation: fmt.Sprintf("files[%d]", filecount),
					},
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
			PAXRecords: map[string]string{
				moby.PaxRecordLinuxkitSource:   "linuxkit.files",
				moby.PaxRecordLinuxkitLocation: fmt.Sprintf("files[%d]", filecount),
			},
		}
		if f.Directory {
			if f.Contents != nil {
				return errors.New("directory with contents not allowed")
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

// extractPackageFilesFromTar reads files from the input tar and extracts those that have the correct
// PAXRecords - keys and values - to the tarWriter.
func extractPackageFilesFromTar(inTar *os.File, tw tarWriter, image, section string) error {
	log.Infof("Copy %s files from input tar: %s", section, image)
	// copy kernel files over
	if _, err := inTar.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to seek to beginning of input tar: %w", err)
	}
	tr := tar.NewReader(inTar)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read input tar: %w", err)
		}
		if hdr.PAXRecords == nil {
			continue
		}
		if hdr.PAXRecords[moby.PaxRecordLinuxkitSource] == image && hdr.PAXRecords[moby.PaxRecordLinuxkitLocation] == section {
			if err := tw.WriteHeader(hdr); err != nil {
				return fmt.Errorf("failed to write header: %w", err)
			}
			if _, err := io.Copy(tw, tr); err != nil {
				return fmt.Errorf("failed to copy %s file: %w", section, err)
			}
		}
	}
	return nil
}
