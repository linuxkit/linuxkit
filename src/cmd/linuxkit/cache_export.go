package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"

	"github.com/containerd/containerd/reference"
	cachepkg "github.com/linuxkit/linuxkit/src/cmd/linuxkit/cache"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
	log "github.com/sirupsen/logrus"
)

func cacheExport(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)

	cacheDir := flagOverEnvVarOverDefaultString{def: defaultLinuxkitCache(), envVar: envVarCacheDir}
	fs.Var(&cacheDir, "cache", fmt.Sprintf("Directory for caching and finding cached image, overrides env var %s", envVarCacheDir))
	arch := fs.String("arch", runtime.GOARCH, "Architecture to resolve an index to an image, if the provided image name is an index")
	outfile := fs.String("outfile", "", "Path to file to save output, '-' for stdout")
	format := fs.String("format", "oci", "export format, one of 'oci', 'filesystem'")
	tagName := fs.String("name", "", "override the provided image name in the exported tar file; useful only for format=oci")

	if err := fs.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}

	// get the requested images
	if fs.NArg() < 1 {
		log.Fatal("At least one image name is required")
	}
	names := fs.Args()
	name := names[0]
	fullname := util.ReferenceExpand(name)

	p, err := cachepkg.NewProvider(cacheDir.String())
	if err != nil {
		log.Fatalf("unable to read a local cache: %v", err)
	}
	ref, err := reference.Parse(fullname)
	if err != nil {
		log.Fatalf("invalid image name %s: %v", name, err)
	}
	desc, err := p.FindDescriptor(&ref)
	if err != nil {
		log.Fatalf("unable to find image named %s: %v", name, err)
	}

	src := p.NewSource(&ref, *arch, desc)
	var reader io.ReadCloser
	switch *format {
	case "oci":
		fullTagName := fullname
		if *tagName != "" {
			fullTagName = util.ReferenceExpand(*tagName)
		}
		reader, err = src.V1TarReader(fullTagName)
	case "filesystem":
		reader, err = src.TarReader()
	default:
		log.Fatalf("requested unknown format %s: %v", name, err)
	}
	if err != nil {
		log.Fatalf("error getting reader for image %s: %v", name, err)
	}
	defer reader.Close()

	// try to write the output file
	var w io.Writer
	switch {
	case outfile == nil, *outfile == "":
		log.Fatal("'outfile' flag is required")
	case *outfile == "-":
		w = os.Stdout
	default:
		f, err := os.OpenFile(*outfile, os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			log.Fatalf("unable to open %s: %v", *outfile, err)
		}
		defer f.Close()
		w = f
	}

	_, _ = io.Copy(w, reader)
}
