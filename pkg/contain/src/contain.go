package main

import (
	"context"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cio"
	"github.com/containerd/containerd/oci"
	specs "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func Execute(args []string, mounter Contain, con *connection) error {
	mountContainer, err := getContainer(con, mounter.Name)
	if err != nil {
		if err.Error() == ContainerNotFoundError {
			log.Warnln("container not found")
			mountContainer, err = createContainer(con, mounter)
		}
		if err != nil {
			return err
		}
	}
	return startTask(args, mountContainer, con.namespace)
}

func startTask(args []string, mountContainer containerd.Container, namespace context.Context) error {
	log.Infoln("update args:", args)
	mountContainer.Update(namespace, WithUpdatedSpecs(updateContainerArgs(append([]string{"echo"}, args...))))
	log.Infoln("start task on:", mountContainer.ID())
	task, err := mountContainer.NewTask(namespace, cio.Stdio)
	if err != nil {
		return err
	}
	defer task.Delete(namespace)
	exitChan, err := task.Wait(namespace)
	if err != nil {
		return err
	}
	exit := <-exitChan
	if err = exit.Error(); err != nil {
		return err
	}
	return err
}

func createContainer(con *connection, contain Contain) (containerd.Container, error) {
	var image containerd.Image
	client := con.client
	context := con.namespace
	log.Infoln("searching for image:", contain.Image)
	images, err := client.ListImages(context)
	if err != nil {
		return nil, err
	}
	for _, i := range images {
		if i.Name() == contain.Image {
			log.Infoln("found image:", i.Name())
			image = i
		}
	}
	if image == nil {
		log.Infoln("image not found. Pulling: ", contain.Image)
		image, err = client.Pull(context, contain.Image, containerd.WithPullUnpack)
		if err != nil {
			return nil, err
		}
	}

	log.Infoln("creating new container:", contain.Name)
	return client.NewContainer(
		context,
		contain.Name,
		containerd.WithNewSnapshot(contain.Name, image),
		containerd.WithNewSpec(
			oci.WithImageConfig(image),
			WithMounterMounts(contain.Source, contain.Destination),
			oci.WithHostNamespace(specs.PIDNamespace),
		),
	)
}

const ContainerNotFoundError = "Container not Found!"

func getContainer(con *connection, name string) (containerd.Container, error) {
	containers, err := con.client.Containers(con.namespace)
	log.Infof("searching for container %v in namespace %v", name, "?")
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		if container.ID() == name {
			return container, nil
		}
	}
	return nil, errors.New(ContainerNotFoundError)
}
