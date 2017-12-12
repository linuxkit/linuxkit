package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/containerd/containerd"
	"github.com/containerd/containerd/errdefs"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func cleanupTask(ctx context.Context, ctr containerd.Container) error {
	task, err := ctr.Task(ctx, nil)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "getting task")
	}

	deleteErr := make(chan error, 1)
	deleteCtx, deleteCancel := context.WithCancel(ctx)
	defer deleteCancel()

	go func(ctx context.Context, ch chan error) {
		_, err := task.Delete(ctx)
		if err != nil {
			ch <- errors.Wrap(err, "killing task")
		}
		ch <- nil
	}(deleteCtx, deleteErr)

	sig := syscall.SIGKILL
	if err := task.Kill(ctx, sig); err != nil && !errdefs.IsNotFound(err) {
		return errors.Wrapf(err, "killing task with %q", sig)
	}

	select {
	case err := <-deleteErr:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func systemInitCmd(ctx context.Context, args []string) {
	invoked := filepath.Base(os.Args[0])
	flags := flag.NewFlagSet("system-init", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Printf("USAGE: %s system-init\n\n", invoked)
		fmt.Printf("Options:\n")
		flags.PrintDefaults()
	}

	sock := flags.String("sock", defaultSocket, "Path to containerd socket")
	path := flags.String("path", defaultPath, "Path to service configs")
	binary := flags.String("containerd", defaultContainerd, "Path to containerd")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}
	args = flags.Args()

	if len(args) != 0 {
		fmt.Println("Unexpected argument")
		flags.Usage()
		os.Exit(1)
	}

	// remove (unlikely) old containerd socket
	_ = os.Remove(*sock)

	// start up containerd
	cmd := exec.Command(*binary)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.WithError(err).Fatal("cannot start containerd")
	}

	// wait for containerd socket to appear
	for {
		_, err := os.Stat(*sock)
		if err == nil {
			break
		}
		err = cmd.Process.Signal(syscall.Signal(0))
		if err != nil {
			// process not there, wait() to find error
			err = cmd.Wait()
			log.WithError(err).Fatal("containerd process exited")
		}
		time.Sleep(100 * time.Millisecond)
	}

	// connect to containerd
	client, err := containerd.New(*sock)
	if err != nil {
		log.WithError(err).Fatal("creating containerd client")
	}

	ctrs, err := client.Containers(ctx)
	if err != nil {
		log.WithError(err).Fatal("listing containers")
	}

	// Clean up any old containers
	// None of the errors in this loop are fatal since we want to
	// keep trying.
	for _, ctr := range ctrs {
		log.Infof("Cleaning up stale service: %q", ctr.ID())
		log := log.WithFields(log.Fields{
			"service": ctr.ID(),
		})

		if err := cleanupTask(ctx, ctr); err != nil {
			log.WithError(err).Error("cleaning up task")
		}

		if err := ctr.Delete(ctx); err != nil {
			log.WithError(err).Error("deleting container")
		}
	}

	// Start up containers
	files, err := ioutil.ReadDir(*path)
	// just skip if there is an error, eg no such path
	if err != nil {
		return
	}
	for _, file := range files {
		if id, pid, msg, err := start(ctx, file.Name(), *sock, *path, ""); err != nil {
			log.WithError(err).Error(msg)
		} else {
			log.Debugf("Started %s pid %d", id, pid)
		}
	}
}
