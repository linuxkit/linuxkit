package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/moby"
	mobybuild "github.com/linuxkit/linuxkit/src/cmd/linuxkit/moby/build"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/spec"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

const (
	defaultNameForStdin = "moby"
	defaultSbomFilename = "sbom.spdx.json"
)

type formatList []string

func (f *formatList) String() string {
	return fmt.Sprint(*f)
}

func (f *formatList) Set(value string) error {
	// allow comma separated options or multiple options
	for _, cs := range strings.Split(value, ",") {
		*f = append(*f, cs)
	}
	return nil
}
func (f *formatList) Type() string {
	return "[]string"
}

func buildCmd() *cobra.Command {

	var (
		name               string
		dir                string
		outputFile         string
		sizeString         string
		pull               bool
		docker             bool
		decompressKernel   bool
		arch               string
		cacheDir           flagOverEnvVarOverDefaultString
		buildFormats       formatList
		outputTypes        = mobybuild.OutputTypes()
		noSbom             bool
		sbomOutputFilename string
		inputTar           string
		sbomCurrentTime    bool
		dryRun             bool
	)
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build a bootable OS image from a yaml configuration file",
		Long: `Build a bootable OS image from a yaml configuration file.

The generated image can be in one of multiple formats which can be run on various platforms.
`,
		Example: `  linuxkit build [options] <file>[.yml]`,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" && outputFile == "" {
				conf := args[len(args)-1]
				if conf == "-" {
					name = defaultNameForStdin
				} else {
					name = strings.TrimSuffix(filepath.Base(conf), filepath.Ext(conf))
				}
			}

			// There are two types of output, they will probably be split into "build" and "package" later
			// the basic outputs are tarballs, while the packaged ones are the LinuxKit out formats that
			// cannot be streamed but we do allow multiple ones to be built.

			if len(buildFormats) == 0 {
				if outputFile == "" {
					buildFormats = formatList{"kernel+initrd"}
				} else {
					buildFormats = formatList{"tar"}
				}
			}

			log.Debugf("Formats selected: %s", buildFormats.String())

			if len(buildFormats) > 1 {
				for _, o := range buildFormats {
					if mobybuild.Streamable(o) {
						return fmt.Errorf("format type %s must be the only format specified", o)
					}
				}
			}

			if len(buildFormats) == 1 && mobybuild.Streamable(buildFormats[0]) {
				if outputFile == "" {
					outputFile = filepath.Join(dir, name+"."+buildFormats[0])
					// stop the errors in the validation below
					name = ""
					dir = ""
				}
			} else {
				err := mobybuild.ValidateFormats(buildFormats, cacheDir.String())
				if err != nil {
					return fmt.Errorf("error parsing formats: %v", err)
				}
			}

			if inputTar != "" && pull {
				return fmt.Errorf("cannot use --input-tar and --pull together")
			}

			var outfile *os.File
			if outputFile != "" {
				if len(buildFormats) > 1 {
					return fmt.Errorf("the -output option can only be specified when generating a single output format")
				}
				if name != "" {
					return fmt.Errorf("the -output option cannot be specified with -name")
				}
				if dir != "" {
					return fmt.Errorf("the -output option cannot be specified with -dir")
				}
				if !mobybuild.Streamable(buildFormats[0]) {
					return fmt.Errorf("the -output option cannot be specified for build type %s as it cannot be streamed", buildFormats[0])
				}
				if outputFile == "-" {
					outfile = os.Stdout
				} else {
					var err error
					outfile, err = os.Create(outputFile)
					if err != nil {
						log.Fatalf("cannot open output file: %v", err)
					}
					defer func() { _ = outfile.Close() }()
				}
			}

			size, err := getDiskSizeMB(sizeString)
			if err != nil {
				log.Fatalf("unable to parse disk size: %v", err)
			}

			var (
				m                  moby.Moby
				templatesSupported bool
			)
			for _, arg := range args {
				var config []byte
				if conf := arg; conf == "-" {
					var err error
					config, err = io.ReadAll(os.Stdin)
					if err != nil {
						return fmt.Errorf("cannot read stdin: %v", err)
					}
				} else if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
					buffer := new(bytes.Buffer)
					response, err := http.Get(arg)
					if err != nil {
						return fmt.Errorf("cannot fetch remote yaml file: %v", err)
					}
					defer func() { _ = response.Body.Close() }()
					_, err = io.Copy(buffer, response.Body)
					if err != nil {
						return fmt.Errorf("error reading http body: %v", err)
					}
					config = buffer.Bytes()
				} else {
					var err error
					config, err = os.ReadFile(conf)
					if err != nil {
						return fmt.Errorf("cannot open config file: %v", err)
					}
					// templates are only supported for local files
					templatesSupported = true
				}
				var pkgFinder spec.PackageResolver
				if templatesSupported {
					pkgFinder = createPackageResolver(filepath.Dir(arg))
				}
				c, err := moby.NewConfig(config, pkgFinder)
				if err != nil {
					return fmt.Errorf("invalid config: %v", err)
				}
				m, err = moby.AppendConfig(m, c)
				if err != nil {
					return fmt.Errorf("cannot append config files: %v", err)
				}
			}

			if dryRun {
				yml, err := yaml.Marshal(m)
				if err != nil {
					return fmt.Errorf("error generating YAML: %v", err)
				}
				fmt.Println(string(yml))
				return nil
			}

			var (
				tf *os.File
				w  io.Writer
			)
			if outfile != nil {
				w = outfile
			} else {
				if tf, err = os.CreateTemp("", ""); err != nil {
					log.Fatalf("error creating tempfile: %v", err)
				}
				defer func() { _ = os.Remove(tf.Name()) }()
				w = tf
			}
			if inputTar != "" && inputTar == outputFile {
				return fmt.Errorf("input-tar and output file cannot be the same")
			}

			// this is a weird interface, but currently only streamable types can have additional files
			// need to split up the base tarball outputs from the secondary stages
			var tp string
			if mobybuild.Streamable(buildFormats[0]) {
				tp = buildFormats[0]
			}
			var sbomGenerator *mobybuild.SbomGenerator
			if !noSbom {
				sbomGenerator, err = mobybuild.NewSbomGenerator(sbomOutputFilename, sbomCurrentTime)
				if err != nil {
					return fmt.Errorf("error creating sbom generator: %v", err)
				}
			}
			err = mobybuild.Build(m, w, mobybuild.BuildOpts{Pull: pull, BuilderType: tp, DecompressKernel: decompressKernel, CacheDir: cacheDir.String(), DockerCache: docker, Arch: arch, SbomGenerator: sbomGenerator, InputTar: inputTar})
			if err != nil {
				return fmt.Errorf("%v", err)
			}

			if outfile == nil {
				image := tf.Name()
				if err := tf.Close(); err != nil {
					return fmt.Errorf("error closing tempfile: %v", err)
				}

				log.Infof("Create outputs:")
				err = mobybuild.Formats(filepath.Join(dir, name), image, buildFormats, size, arch, cacheDir.String())
				if err != nil {
					return fmt.Errorf("error writing outputs: %v", err)
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name to use for output files")
	cmd.Flags().StringVar(&dir, "dir", "", "Directory for output files, default current directory")
	cmd.Flags().StringVar(&outputFile, "o", "", "File to use for a single output, or '-' for stdout")
	cmd.Flags().StringVar(&sizeString, "size", "1024M", "Size for output image, if supported and fixed size")
	cmd.Flags().BoolVar(&pull, "pull", false, "Always pull images")
	cmd.Flags().BoolVar(&docker, "docker", false, "Check for images in docker before linuxkit cache")
	cmd.Flags().BoolVar(&decompressKernel, "decompress-kernel", false, "Decompress the Linux kernel (default false)")
	cmd.Flags().StringVar(&arch, "arch", runtime.GOARCH, "target architecture for which to build")
	cmd.Flags().VarP(&buildFormats, "format", "f", "Formats to create [ "+strings.Join(outputTypes, " ")+" ]")
	cmd.Flags().StringVar(&inputTar, "input-tar", "", "path to tar from previous linuxkit build to use as input; if provided, will take files from images from this tar, using OCI images only to replace or update files. Always copies to a temporary working directory to avoid overwriting. Only works if input-tar file has the linuxkit.yaml used to build it in the exact same location. Incompatible with --pull")
	cacheDir = flagOverEnvVarOverDefaultString{def: defaultLinuxkitCache(), envVar: envVarCacheDir}
	cmd.Flags().Var(&cacheDir, "cache", fmt.Sprintf("Directory for caching and finding cached image, overrides env var %s", envVarCacheDir))
	cmd.Flags().BoolVar(&noSbom, "no-sbom", false, "suppress consolidation of sboms on input container images to a single sbom and saving in the output filesystem")
	cmd.Flags().BoolVar(&sbomCurrentTime, "sbom-current-time", false, "whether to use the current time as the build time in the sbom; this will make the build non-reproducible (default false)")
	cmd.Flags().StringVar(&sbomOutputFilename, "sbom-output", defaultSbomFilename, "filename to save the output to in the root filesystem")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Do not actually build, just print the final yml file that would be used, including all merges and templates")

	return cmd
}
