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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const defaultNameForStdin = "moby"

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
		name             string
		dir              string
		outputFile       string
		sizeString       string
		pull             bool
		docker           bool
		decompressKernel bool
		arch             string
		cacheDir         flagOverEnvVarOverDefaultString
		buildFormats     formatList
		outputTypes      = moby.OutputTypes()
	)
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build a bootable OS image from a yaml configuration file",
		Long: `Build a bootable OS image from a yaml configuration file.

The generated image can be in one of multiple formats which can be run on various platforms.
`,
		Example: `  linuxkit build [options] <file>[.yml]`,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
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
					if moby.Streamable(o) {
						return fmt.Errorf("Format type %s must be the only format specified", o)
					}
				}
			}

			if len(buildFormats) == 1 && moby.Streamable(buildFormats[0]) {
				if outputFile == "" {
					outputFile = filepath.Join(dir, name+"."+buildFormats[0])
					// stop the errors in the validation below
					name = ""
					dir = ""
				}
			} else {
				err := moby.ValidateFormats(buildFormats, cacheDir.String())
				if err != nil {
					return fmt.Errorf("Error parsing formats: %v", err)
				}
			}

			var outfile *os.File
			if outputFile != "" {
				if len(buildFormats) > 1 {
					return fmt.Errorf("The -output option can only be specified when generating a single output format")
				}
				if name != "" {
					return fmt.Errorf("The -output option cannot be specified with -name")
				}
				if dir != "" {
					return fmt.Errorf("The -output option cannot be specified with -dir")
				}
				if !moby.Streamable(buildFormats[0]) {
					return fmt.Errorf("The -output option cannot be specified for build type %s as it cannot be streamed", buildFormats[0])
				}
				if outputFile == "-" {
					outfile = os.Stdout
				} else {
					var err error
					outfile, err = os.Create(outputFile)
					if err != nil {
						log.Fatalf("Cannot open output file: %v", err)
					}
					defer outfile.Close()
				}
			}

			size, err := getDiskSizeMB(sizeString)
			if err != nil {
				log.Fatalf("Unable to parse disk size: %v", err)
			}

			var m moby.Moby
			for _, arg := range args {
				var config []byte
				if conf := arg; conf == "-" {
					var err error
					config, err = io.ReadAll(os.Stdin)
					if err != nil {
						return fmt.Errorf("Cannot read stdin: %v", err)
					}
				} else if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
					buffer := new(bytes.Buffer)
					response, err := http.Get(arg)
					if err != nil {
						return fmt.Errorf("Cannot fetch remote yaml file: %v", err)
					}
					defer response.Body.Close()
					_, err = io.Copy(buffer, response.Body)
					if err != nil {
						return fmt.Errorf("Error reading http body: %v", err)
					}
					config = buffer.Bytes()
				} else {
					var err error
					config, err = os.ReadFile(conf)
					if err != nil {
						return fmt.Errorf("Cannot open config file: %v", err)
					}
				}

				c, err := moby.NewConfig(config)
				if err != nil {
					return fmt.Errorf("Invalid config: %v", err)
				}
				m, err = moby.AppendConfig(m, c)
				if err != nil {
					return fmt.Errorf("Cannot append config files: %v", err)
				}
			}

			var tf *os.File
			var w io.Writer
			if outfile != nil {
				w = outfile
			} else {
				if tf, err = os.CreateTemp("", ""); err != nil {
					log.Fatalf("Error creating tempfile: %v", err)
				}
				defer os.Remove(tf.Name())
				w = tf
			}

			// this is a weird interface, but currently only streamable types can have additional files
			// need to split up the base tarball outputs from the secondary stages
			var tp string
			if moby.Streamable(buildFormats[0]) {
				tp = buildFormats[0]
			}
			err = moby.Build(m, w, moby.BuildOpts{Pull: pull, BuilderType: tp, DecompressKernel: decompressKernel, CacheDir: cacheDir.String(), DockerCache: docker, Arch: arch})
			if err != nil {
				return fmt.Errorf("%v", err)
			}

			if outfile == nil {
				image := tf.Name()
				if err := tf.Close(); err != nil {
					return fmt.Errorf("Error closing tempfile: %v", err)
				}

				log.Infof("Create outputs:")
				err = moby.Formats(filepath.Join(dir, name), image, buildFormats, size, cacheDir.String())
				if err != nil {
					return fmt.Errorf("Error writing outputs: %v", err)
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
	cacheDir = flagOverEnvVarOverDefaultString{def: defaultLinuxkitCache(), envVar: envVarCacheDir}
	cmd.Flags().Var(&cacheDir, "cache", fmt.Sprintf("Directory for caching and finding cached image, overrides env var %s", envVarCacheDir))

	return cmd
}
