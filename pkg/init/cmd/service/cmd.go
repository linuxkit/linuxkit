package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/namespaces"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	log "github.com/sirupsen/logrus"
)

func parseCmd(ctx context.Context, command string, args []string) (*log.Entry, string, string, string, string) {
	invoked := filepath.Base(os.Args[0])
	flags := flag.NewFlagSet(command, flag.ExitOnError)
	flags.Usage = func() {
		fmt.Printf("USAGE: %s %s [service]\n\n", invoked, command)
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

	log := log.WithFields(log.Fields{
		"service": service,
	})

	return log, service, *sock, *path, *dumpSpec
}

func stopCmd(ctx context.Context, args []string) {
	log, service, sock, path, _ := parseCmd(ctx, "stop", args)

	log.Infof("Stopping service: %q", service)
	id, pid, msg, err := stop(ctx, service, sock, path)
	if err != nil {
		log.WithError(err).Fatal(msg)
	}

	log.Debugf("Stopped %s pid %d", id, pid)
}

func startCmd(ctx context.Context, args []string) {
	log, service, sock, path, dumpSpec := parseCmd(ctx, "start", args)

	log.Infof("Starting service: %q", service)
	id, pid, msg, err := start(ctx, service, sock, path, dumpSpec)
	if err != nil {
		log.WithError(err).Fatal(msg)
	}

	log.Debugf("Started %s pid %d", id, pid)
}

func restartCmd(ctx context.Context, args []string) {
	// validate arguments with the command as "restart"
	parseCmd(ctx, "restart", args)

	stopCmd(ctx, args)
	startCmd(ctx, args)
}

type logio struct {
	config cio.Config
}

func (c *logio) Config() cio.Config {
	return c.config
}

func (c *logio) Cancel() {
}

func (c *logio) Wait() {
}

func (c *logio) Close() error {
	return nil
}

func stop(ctx context.Context, service, sock, basePath string) (string, uint32, string, error) {
	path := filepath.Join(basePath, service)

	runtimeConfig := getRuntimeConfig(path)

	client, err := containerd.New(sock)
	if err != nil {
		return "", 0, "creating containerd client", err
	}

	if runtimeConfig.Namespace != "" {
		ctx = namespaces.WithNamespace(ctx, runtimeConfig.Namespace)
	}

	ctr, err := client.LoadContainer(ctx, service)
	if err != nil {
		return "", 0, "loading container", err
	}

	task, err := ctr.Task(ctx, nil)
	if err != nil {
		return "", 0, "fetching task", err
	}

	id := ctr.ID()
	pid := task.Pid()

	err = task.Kill(ctx, 9)
	if err != nil {
		return "", 0, "killing task", err
	}

	_, err = task.Wait(ctx)
	if err != nil {
		return "", 0, "waiting for task to exit", err
	}

	_, err = task.Delete(ctx)
	if err != nil {
		return "", 0, "deleting task", err
	}

	err = ctr.Delete(ctx)
	if err != nil {
		return "", 0, "deleting container", err
	}

	return id, pid, "", nil
}

func start(ctx context.Context, service, sock, basePath, dumpSpec string) (string, uint32, string, error) {
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

	if runtimeConfig.Namespace != "" {
		ctx = namespaces.WithNamespace(ctx, runtimeConfig.Namespace)
	}

	ctr, err := client.NewContainer(ctx, service, containerd.WithSpec(spec))
	if err != nil {
		return "", 0, "failed to create container", err
	}

	logger := GetLog(varLogDir)

	io := func(id string) (cio.IO, error) {
		stdoutFile := logger.Path(service + ".out")
		stderrFile := logger.Path(service + ".err")
		return &logio{
			cio.Config{
				Stdin:    "/dev/null",
				Stdout:   stdoutFile,
				Stderr:   stderrFile,
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
