package main

import (
	"archive/tar"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
)

const defaultNameForStdin = "moby"

type outputList []string

func (o *outputList) String() string {
	return fmt.Sprint(*o)
}

func (o *outputList) Set(value string) error {
	// allow comma seperated options or multiple options
	for _, cs := range strings.Split(value, ",") {
		*o = append(*o, cs)
	}
	return nil
}

var streamable = map[string]bool{
	"docker": true,
	"tar":    true,
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

// Process the build arguments and execute build
func build(args []string) {
	var buildOut outputList

	outputTypes := []string{}
	for k := range streamable {
		outputTypes = append(outputTypes, k)
	}
	for k := range outFuns {
		outputTypes = append(outputTypes, k)
	}
	sort.Strings(outputTypes)

	buildCmd := flag.NewFlagSet("build", flag.ExitOnError)
	buildCmd.Usage = func() {
		fmt.Printf("USAGE: %s build [options] <file>[.yml] | -\n\n", os.Args[0])
		fmt.Printf("Options:\n")
		buildCmd.PrintDefaults()
	}
	buildName := buildCmd.String("name", "", "Name to use for output files")
	buildDir := buildCmd.String("dir", "", "Directory for output files, default current directory")
	buildOutputFile := buildCmd.String("o", "", "File to use for a single output, or '-' for stdout")
	buildSize := buildCmd.String("size", "1024M", "Size for output image, if supported and fixed size")
	buildPull := buildCmd.Bool("pull", false, "Always pull images")
	buildDisableTrust := buildCmd.Bool("disable-content-trust", false, "Skip image trust verification specified in trust section of config (default false)")
	buildHyperkit := buildCmd.Bool("hyperkit", false, "Use hyperkit for LinuxKit based builds where possible")
	buildCmd.Var(&buildOut, "output", "Output types to create [ "+strings.Join(outputTypes, " ")+" ]")

	if err := buildCmd.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}
	remArgs := buildCmd.Args()

	if len(remArgs) == 0 {
		fmt.Println("Please specify a configuration file")
		buildCmd.Usage()
		os.Exit(1)
	}

	name := *buildName
	if name == "" {
		conf := remArgs[len(remArgs)-1]
		if conf == "-" {
			name = defaultNameForStdin
		} else {
			name = strings.TrimSuffix(filepath.Base(conf), filepath.Ext(conf))
		}
	}

	// There are two types of output, they will probably be split into "build" and "package" later
	// the basic outputs are tarballs, while the packaged ones are the LinuxKit out formats that
	// cannot be streamed but we do allow multiple ones to be built.

	if len(buildOut) == 0 {
		if *buildOutputFile == "" {
			buildOut = outputList{"kernel+initrd"}
		} else {
			buildOut = outputList{"tar"}
		}
	}

	log.Debugf("Outputs selected: %s", buildOut.String())

	if len(buildOut) > 1 {
		for _, o := range buildOut {
			if streamable[o] {
				log.Fatalf("Output type %s must be the only output specified", o)
			}
		}
	}

	if len(buildOut) == 1 && streamable[buildOut[0]] {
		if *buildOutputFile == "" {
			*buildOutputFile = filepath.Join(*buildDir, name+"."+buildOut[0])
			// stop the errors in the validation below
			*buildName = ""
			*buildDir = ""
		}

	} else {
		err := validateOutputs(buildOut)
		if err != nil {
			log.Errorf("Error parsing outputs: %v", err)
			buildCmd.Usage()
			os.Exit(1)
		}
	}

	var outputFile *os.File
	var addition addFun
	if *buildOutputFile != "" {
		if len(buildOut) > 1 {
			log.Fatal("The -output option can only be specified when generating a single output format")
		}
		if *buildName != "" {
			log.Fatal("The -output option cannot be specified with -name")
		}
		if *buildDir != "" {
			log.Fatal("The -output option cannot be specified with -dir")
		}
		if !streamable[buildOut[0]] {
			log.Fatalf("The -output option cannot be specified for build type %s as it cannot be streamed", buildOut[0])
		}
		if *buildOutputFile == "-" {
			outputFile = os.Stdout
		} else {
			var err error
			outputFile, err = os.Create(*buildOutputFile)
			if err != nil {
				log.Fatalf("Cannot open output file: %v", err)
			}
			defer outputFile.Close()
		}
		addition = additions[buildOut[0]]
	}

	size, err := getDiskSizeMB(*buildSize)
	if err != nil {
		log.Fatalf("Unable to parse disk size: %v", err)
	}

	var moby Moby
	for _, arg := range remArgs {
		var config []byte
		if conf := arg; conf == "-" {
			var err error
			config, err = ioutil.ReadAll(os.Stdin)
			if err != nil {
				log.Fatalf("Cannot read stdin: %v", err)
			}
		} else {
			var err error
			config, err = ioutil.ReadFile(conf)
			if err != nil {
				log.Fatalf("Cannot open config file: %v", err)
			}
		}

		m, err := NewConfig(config)
		if err != nil {
			log.Fatalf("Invalid config: %v", err)
		}
		moby = AppendConfig(moby, m)
	}

	if *buildDisableTrust {
		log.Debugf("Disabling content trust checks for this build")
		moby.Trust = TrustConfig{}
	}

	var buf *bytes.Buffer
	var w io.Writer
	if outputFile != nil {
		w = outputFile
	} else {
		buf = new(bytes.Buffer)
		w = buf
	}
	buildInternal(moby, w, *buildPull, addition)

	if outputFile == nil {
		image := buf.Bytes()
		log.Infof("Create outputs:")
		err = outputs(filepath.Join(*buildDir, name), image, buildOut, size, *buildHyperkit)
		if err != nil {
			log.Fatalf("Error writing outputs: %v", err)
		}
	}
}

// Parse a string which is either a number in MB, or a number with
// either M (for Megabytes) or G (for GigaBytes) as a suffix and
// returns the number in MB. Return 0 if string is empty.
func getDiskSizeMB(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	sz := len(s)
	if strings.HasSuffix(s, "G") {
		i, err := strconv.Atoi(s[:sz-1])
		if err != nil {
			return 0, err
		}
		return i * 1024, nil
	}
	if strings.HasSuffix(s, "M") {
		s = s[:sz-1]
	}
	return strconv.Atoi(s)
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

// Perform the actual build process
// TODO return error not panic
func buildInternal(m Moby, w io.Writer, pull bool, addition addFun) {
	iw := tar.NewWriter(w)

	if m.Kernel.Image != "" {
		// get kernel and initrd tarball from container
		log.Infof("Extract kernel image: %s", m.Kernel.Image)
		kf := newKernelFilter(iw, m.Kernel.Cmdline)
		err := ImageTar(m.Kernel.Image, "", kf, enforceContentTrust(m.Kernel.Image, &m.Trust), pull)
		if err != nil {
			log.Fatalf("Failed to extract kernel image and tarball: %v", err)
		}
		err = kf.Close()
		if err != nil {
			log.Fatalf("Close error: %v", err)
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
			log.Fatalf("Failed to build init tarball from %s: %v", ii, err)
		}
	}

	if len(m.Onboot) != 0 {
		log.Infof("Add onboot containers:")
	}
	for i, image := range m.Onboot {
		log.Infof("  Create OCI config for %s", image.Image)
		useTrust := enforceContentTrust(image.Image, &m.Trust)
		config, err := ConfigToOCI(image, useTrust)
		if err != nil {
			log.Fatalf("Failed to create config.json for %s: %v", image.Image, err)
		}
		so := fmt.Sprintf("%03d", i)
		path := "containers/onboot/" + so + "-" + image.Name
		err = ImageBundle(path, image.Image, config, iw, useTrust, pull)
		if err != nil {
			log.Fatalf("Failed to extract root filesystem for %s: %v", image.Image, err)
		}
	}

	if len(m.Services) != 0 {
		log.Infof("Add service containers:")
	}
	for _, image := range m.Services {
		log.Infof("  Create OCI config for %s", image.Image)
		useTrust := enforceContentTrust(image.Image, &m.Trust)
		config, err := ConfigToOCI(image, useTrust)
		if err != nil {
			log.Fatalf("Failed to create config.json for %s: %v", image.Image, err)
		}
		path := "containers/services/" + image.Name
		err = ImageBundle(path, image.Image, config, iw, useTrust, pull)
		if err != nil {
			log.Fatalf("Failed to extract root filesystem for %s: %v", image.Image, err)
		}
	}

	// add files
	err := filesystem(m, iw)
	if err != nil {
		log.Fatalf("failed to add filesystem parts: %v", err)
	}

	// add anything additional for this output type
	if addition != nil {
		err = addition(iw)
		if err != nil {
			log.Fatalf("Failed to add additional files")
		}
	}

	err = iw.Close()
	if err != nil {
		log.Fatalf("initrd close error: %v", err)
	}

	return
}

// kernelFilter is a tar.Writer that transforms a kernel image into the output we want on underlying tar writer
type kernelFilter struct {
	tw          *tar.Writer
	buffer      *bytes.Buffer
	cmdline     string
	discard     bool
	foundKernel bool
	foundKTar   bool
}

func newKernelFilter(tw *tar.Writer, cmdline string) *kernelFilter {
	return &kernelFilter{tw: tw, cmdline: cmdline}
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
	if !k.foundKTar {
		return errors.New("did not find kernel.tar in kernel image")
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
	case "kernel":
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
	case "kernel.tar":
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

func filesystem(m Moby, tw *tar.Writer) error {
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
		if !f.Directory && f.Contents == "" && f.Symlink == "" {
			if f.Source == "" {
				return errors.New("Contents of file not specified")
			}
			if len(f.Source) > 2 && f.Source[:2] == "~/" {
				f.Source = homeDir() + f.Source[1:]
			}
			contents, err := ioutil.ReadFile(f.Source)
			if err != nil {
				return err
			}

			f.Contents = string(contents)
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
				}
				err := tw.WriteHeader(hdr)
				if err != nil {
					return err
				}
				addedFiles[root] = true
			}
		}
		addedFiles[f.Path] = true
		if f.Directory {
			if f.Contents != "" {
				return errors.New("Directory with contents not allowed")
			}
			hdr := &tar.Header{
				Name:     f.Path,
				Typeflag: tar.TypeDir,
				Mode:     mode,
			}
			err := tw.WriteHeader(hdr)
			if err != nil {
				return err
			}
		} else if f.Symlink != "" {
			hdr := &tar.Header{
				Name:     f.Path,
				Typeflag: tar.TypeSymlink,
				Mode:     mode,
				Linkname: f.Symlink,
			}
			err := tw.WriteHeader(hdr)
			if err != nil {
				return err
			}
		} else {
			hdr := &tar.Header{
				Name: f.Path,
				Mode: mode,
				Size: int64(len(f.Contents)),
			}
			err := tw.WriteHeader(hdr)
			if err != nil {
				return err
			}
			_, err = tw.Write([]byte(f.Contents))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
