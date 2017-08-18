package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	log "github.com/sirupsen/logrus"
)

func startCmd(args []string) {
	invoked := filepath.Base(os.Args[0])
	flags := flag.NewFlagSet("start", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Printf("USAGE: %s start [service]\n\n", invoked)
		fmt.Printf("Options:\n")
		flags.PrintDefaults()
	}

	sock := flags.String("sock", defaultSocket, "Path to containerd socket")
	path := flags.String("path", defaultPath, "Path to service configs")

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

	log.Infof("Starting service: %q", service)
	log := log.WithFields(log.Fields{
		"service": service,
	})

	id, pid, msg, err := start(service, *sock, *path, *dumpSpec)
	if err != nil {
		log.WithError(err).Fatal(msg)
	}

	log.Debugf("Started %s pid %d", id, pid)
}

type cio struct {
	config containerd.IOConfig
}

func (c *cio) Config() containerd.IOConfig {
	return c.config
}

func (c *cio) Cancel() {
}

func (c *cio) Wait() {
}

func (c *cio) Close() error {
	return nil
}

func start(service, sock, basePath, dumpSpec string) (string, uint32, string, error) {
	path := filepath.Join(basePath, service)

	runtimeConfig := getRuntimeConfig(path)

	rootfs := filepath.Join(path, "rootfs")

	if err := prepareFilesystem(path, runtimeConfig); err != nil {
		return "", 0, "preparing filesystem", err
	}

	client, err := containerd.New(sock)
	if err != nil {
		return "", 0, "creating containerd client", err
	}

	ctx := namespaces.WithNamespace(context.Background(), "default")

	var spec *specs.Spec
	specf, err := os.Open(filepath.Join(path, "config.json"))
	if err != nil {
		return "", 0, "failed to read service spec", err
	}
	if err := json.NewDecoder(specf).Decode(&spec); err != nil {
		return "", 0, "failed to parse service spec", err
	}

	spec.Root.Path = rootfs

	if dumpSpec != "" {
		d, err := os.Create(dumpSpec)
		if err != nil {
			return "", 0, "failed to open file for spec dump", err
		}
		enc := json.NewEncoder(d)
		enc.SetIndent("", "    ")
		if err := enc.Encode(&spec); err != nil {
			return "", 0, "failed to write spec dump", err
		}

	}

	ctr, err := client.NewContainer(ctx, service, containerd.WithSpec(spec))
	if err != nil {
		return "", 0, "failed to create container", err
	}

	io := func(id string) (containerd.IO, error) {
		logfile := filepath.Join("/var/log", service+".log")
		// We just need this to exist.
		if err := ioutil.WriteFile(logfile, []byte{}, 0600); err != nil {
			// if we cannot write to log, discard output
			logfile = "/dev/null"
		}
		return &cio{
			containerd.IOConfig{
				Stdin:    "/dev/null",
				Stdout:   logfile,
				Stderr:   logfile,
				Terminal: false,
			},
		}, nil
	}

	task, err := ctr.NewTask(ctx, io)
	if err != nil {
		// Don't bother to destroy the container here.
		return "", 0, "failed to create task", err
	}

	if err := prepareProcess(int(task.Pid()), runtimeConfig); err != nil {
		return "", 0, "preparing process", err
	}

	if err := task.Start(ctx); err != nil {
		// Don't destroy the container here so it can be inspected for debugging.
		return "", 0, "failed to start task", err
	}

	return ctr.ID(), task.Pid(), "", nil
}
