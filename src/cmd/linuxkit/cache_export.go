package main

import (
	"flag"
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

	cacheDir := fs.String("cache", defaultLinuxkitCache(), "Directory for caching and finding cached image")
	arch := fs.String("arch", runtime.GOARCH, "Architecture to resolve an index to an image, if the provided image name is an index")
	outfile := fs.String("outfile", "", "Path to file to save output, '-' for stdout")

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

	p, err := cachepkg.NewProvider(*cacheDir)
	if err != nil {
		log.Fatalf("unable to read a local cache: %v", err)
	}
	desc, err := p.FindDescriptor(fullname)
	if err != nil {
		log.Fatalf("unable to find image named %s: %v", name, err)
	}
	ref, err := reference.Parse(fullname)
	if err != nil {
		log.Fatalf("invalid image name %s: %v", name, err)
	}

	src := p.NewSource(&ref, *arch, desc)
	reader, err := src.V1TarReader()
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

	io.Copy(w, reader)
}
