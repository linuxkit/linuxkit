package main

import (
	"context"
	"os"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	configFile = "config.yml"
)

type connection struct {
	client    *containerd.Client
	namespace context.Context
}

func main() {
	if len(os.Args) < 2 {
		log.Errorln(errors.New("not enough arguments"))
		os.Exit(1)
	}
	log.SetLevel(log.DebugLevel)
	err := start(os.Args[1:])
	if err != nil {
		log.Errorln(err.Error())
		os.Exit(1)
	}
	log.Infoln("end")
}

func start(args []string) error {
	log.Infoln("parsing configfile:", configFile)
	mounterConfig, err := loadConfig(configFile)
	if err != nil {
		return err
	}
	mounter := Contain{}
	err, mounter = getContain(args, mounterConfig)
	if err != nil {
		return err
	}
	con, err := setup(mounterConfig)
	if err != nil {
		return err
	}

	return Execute(args, mounter, con)
}

func setup(config *ContainConfig) (*connection, error) {
	con := &connection{}
	log.Infoln("connecting to:", config.Socket)
	client, err := containerd.New(config.Socket)
	log.Infoln("connected")
	if err != nil {
		return con, err
	}
	ctx := context.Background()
	namespace := namespaces.WithNamespace(ctx, config.Namespace)
	if err != nil {
		return con, err
	}
	serving, err := client.IsServing(ctx)
	if err != nil {
		return con, err
	}

	if !serving {
		client.Close()
		if err == nil {
			err = errors.New("connection was successful but service is not available")
		}
		return con, err
	}
	con.client = client
	con.namespace = namespace

	return con, nil
}
