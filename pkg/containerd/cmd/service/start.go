package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

func start(args []string) {
	invoked := filepath.Base(os.Args[0])
	flags := flag.NewFlagSet("start", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Printf("USAGE: %s start [service]\n\n", invoked)
		fmt.Printf("Options:\n")
		flags.PrintDefaults()
	}

	sock := flags.String("sock", "/run/containerd/containerd.sock", "Path to containerd socket")

	dumpSpec := flags.String("dump-spec", "", "Dump container spec to file before start")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}
	args = flags.Args()

	if len(args) != 1 {
		fmt.Println("Please specify the service")
		flags.Usage()
		os.Exit(1)
	}

	service := args[0]
	rootfs := filepath.Join("/containers/services", service, "rootfs")
	log.Infof("Starting service: %q", service)
	log := log.WithFields(log.Fields{
		"service": service,
	})

	client, err := containerd.New(*sock)
	if err != nil {
		log.WithError(err).Fatal("creating containerd client")
	}

	ctx := namespaces.WithNamespace(context.Background(), "default")

	var spec *specs.Spec
	specf, err := os.Open(filepath.Join("/containers/services", service, "config.json"))
	if err != nil {
		log.WithError(err).Fatal("failed to read service spec")
	}
	if err := json.NewDecoder(specf).Decode(&spec); err != nil {
		log.WithError(err).Fatal("failed to parse service spec")
	}

	log.Debugf("Rootfs is %s", rootfs)

	spec.Root.Path = rootfs

	if *dumpSpec != "" {
		d, err := os.Create(*dumpSpec)
		if err != nil {
			log.WithError(err).Fatal("failed to open file for spec dump")
		}
		enc := json.NewEncoder(d)
		enc.SetIndent("", "    ")
		if err := enc.Encode(&spec); err != nil {
			log.WithError(err).Fatal("failed to write spec dump")
		}

	}

	ctr, err := client.NewContainer(ctx, service, containerd.WithSpec(spec))
	if err != nil {
		log.WithError(err).Fatal("failed to create container")
	}

	io := func() (*containerd.IO, error) {
		logfile := filepath.Join("/var/log", service+".log")
		// We just need this to exist.
		if err := ioutil.WriteFile(logfile, []byte{}, 0666); err != nil {
			log.WithError(err).Fatal("failed to touch logfile")
		}
		return &containerd.IO{
			Stdin:    "/dev/null",
			Stdout:   logfile,
			Stderr:   logfile,
			Terminal: false,
		}, nil
	}

	task, err := ctr.NewTask(ctx, io)
	if err != nil {
		// Don't bother to destroy the container here.
		log.WithError(err).Fatal("failed to create task")
	}

	if err := task.Start(ctx); err != nil {
		// Don't destroy the container here so it can be inspected for debugging.
		log.WithError(err).Fatal("failed to start task")
	}

	log.Debugf("Started %s pid %d", ctr.ID(), task.Pid())
}
