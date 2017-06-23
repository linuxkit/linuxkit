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
	"runtime"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/moby/tool/src/moby"
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

// Process the build arguments and execute build
func build(args []string) {
	var buildOut outputList

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
	buildHyperkit := buildCmd.Bool("hyperkit", runtime.GOOS == "darwin", "Use hyperkit for LinuxKit based builds where possible")
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
			if moby.Streamable(o) {
				log.Fatalf("Output type %s must be the only output specified", o)
			}
		}
	}

	if len(buildOut) == 1 && moby.Streamable(buildOut[0]) {
		if *buildOutputFile == "" {
			*buildOutputFile = filepath.Join(*buildDir, name+"."+buildOut[0])
			// stop the errors in the validation below
			*buildName = ""
			*buildDir = ""
		}

	} else {
		err := moby.ValidateOutputs(buildOut)
		if err != nil {
			log.Errorf("Error parsing outputs: %v", err)
			buildCmd.Usage()
			os.Exit(1)
		}
	}

	var outputFile *os.File
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
		if !moby.Streamable(buildOut[0]) {
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
				log.Fatal("Cannot fetch remote yaml file: %v", err)
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
		m = moby.AppendConfig(m, c)
	}

	if *buildDisableTrust {
		log.Debugf("Disabling content trust checks for this build")
		m.Trust = moby.TrustConfig{}
	}

	var buf *bytes.Buffer
	var w io.Writer
	if outputFile != nil {
		w = outputFile
	} else {
		buf = new(bytes.Buffer)
		w = buf
	}
	// this is a weird interface, but currently only streamable types can have additional files
	// need to split up the base tarball outputs from the secondary stages
	var tp string
	if moby.Streamable(buildOut[0]) {
		tp = buildOut[0]
	}
	err = moby.Build(m, w, *buildPull, tp)
	if err != nil {
		log.Fatalf("%v", err)
	}

	if outputFile == nil {
		image := buf.Bytes()
		log.Infof("Create outputs:")
		err = moby.Outputs(filepath.Join(*buildDir, name), image, buildOut, size, *buildHyperkit)
		if err != nil {
			log.Fatalf("Error writing outputs: %v", err)
		}
	}
}
