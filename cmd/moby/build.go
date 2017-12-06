package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/moby/tool/src/moby"
	log "github.com/sirupsen/logrus"
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

// Process the build arguments and execute build
func build(args []string) {
	var buildFormats formatList

	outputTypes := moby.OutputTypes()

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
	buildCmd.Var(&buildFormats, "format", "Formats to create [ "+strings.Join(outputTypes, " ")+" ]")

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

	if len(buildFormats) == 0 {
		if *buildOutputFile == "" {
			buildFormats = formatList{"kernel+initrd"}
		} else {
			buildFormats = formatList{"tar"}
		}
	}

	log.Debugf("Formats selected: %s", buildFormats.String())

	if len(buildFormats) > 1 {
		for _, o := range buildFormats {
			if moby.Streamable(o) {
				log.Fatalf("Format type %s must be the only format specified", o)
			}
		}
	}

	if len(buildFormats) == 1 && moby.Streamable(buildFormats[0]) {
		if *buildOutputFile == "" {
			*buildOutputFile = filepath.Join(*buildDir, name+"."+buildFormats[0])
			// stop the errors in the validation below
			*buildName = ""
			*buildDir = ""
		}
	} else {
		err := moby.ValidateFormats(buildFormats)
		if err != nil {
			log.Errorf("Error parsing formats: %v", err)
			buildCmd.Usage()
			os.Exit(1)
		}
	}

	var outputFile *os.File
	if *buildOutputFile != "" {
		if len(buildFormats) > 1 {
			log.Fatal("The -output option can only be specified when generating a single output format")
		}
		if *buildName != "" {
			log.Fatal("The -output option cannot be specified with -name")
		}
		if *buildDir != "" {
			log.Fatal("The -output option cannot be specified with -dir")
		}
		if !moby.Streamable(buildFormats[0]) {
			log.Fatalf("The -output option cannot be specified for build type %s as it cannot be streamed", buildFormats[0])
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
	}

	size, err := getDiskSizeMB(*buildSize)
	if err != nil {
		log.Fatalf("Unable to parse disk size: %v", err)
	}

	var m moby.Moby
	for _, arg := range remArgs {
		var config []byte
		if conf := arg; conf == "-" {
			var err error
			config, err = ioutil.ReadAll(os.Stdin)
			if err != nil {
				log.Fatalf("Cannot read stdin: %v", err)
			}
		} else if strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
			buffer := new(bytes.Buffer)
			response, err := http.Get(arg)
			if err != nil {
				log.Fatalf("Cannot fetch remote yaml file: %v", err)
			}
			defer response.Body.Close()
			_, err = io.Copy(buffer, response.Body)
			if err != nil {
				log.Fatalf("Error reading http body: %v", err)
			}
			config = buffer.Bytes()
		} else {
			var err error
			config, err = ioutil.ReadFile(conf)
			if err != nil {
				log.Fatalf("Cannot open config file: %v", err)
			}
		}

		c, err := moby.NewConfig(config)
		if err != nil {
			log.Fatalf("Invalid config: %v", err)
		}
		m, err = moby.AppendConfig(m, c)
		if err != nil {
			log.Fatalf("Cannot append config files: %v", err)
		}
	}

	if *buildDisableTrust {
		log.Debugf("Disabling content trust checks for this build")
		m.Trust = moby.TrustConfig{}
	}

	var tf *os.File
	var w io.Writer
	if outputFile != nil {
		w = outputFile
	} else {
		if tf, err = ioutil.TempFile("", ""); err != nil {
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
	err = moby.Build(m, w, *buildPull, tp)
	if err != nil {
		log.Fatalf("%v", err)
	}

	if outputFile == nil {
		image := tf.Name()
		if err := tf.Close(); err != nil {
			log.Fatalf("Error closing tempfile: %v", err)
		}

		log.Infof("Create outputs:")
		err = moby.Formats(filepath.Join(*buildDir, name), image, buildFormats, size)
		if err != nil {
			log.Fatalf("Error writing outputs: %v", err)
		}
	}
}
