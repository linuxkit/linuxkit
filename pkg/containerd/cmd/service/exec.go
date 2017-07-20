package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	log "github.com/Sirupsen/logrus"
	"github.com/containerd/console"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/namespaces"
	"golang.org/x/sys/unix"
)

type resizer interface {
	Resize(ctx context.Context, w, h uint32) error
}

type killer interface {
	Kill(context.Context, syscall.Signal) error
}

func exec(args []string) {
	invoked := filepath.Base(os.Args[0])
	flags := flag.NewFlagSet("exec", flag.ExitOnError)
	flags.Usage = func() {
		fmt.Printf("USAGE: %s exec [--tty] [service] [command] [args...]\n\n", invoked)
		fmt.Printf("Options:\n")
		flags.PrintDefaults()
	}

	sock := flags.String("sock", "/run/containerd/containerd.sock", "Path to containerd socket")
	tty := flags.Bool("tty", false, "allocate a TTY")

	if err := flags.Parse(args); err != nil {
		log.Fatal("Unable to parse args")
	}
	args = flags.Args()

	if len(args) < 1 {
		fmt.Println("Please specify the service")
		flags.Usage()
		os.Exit(1)
	}

	if len(args) < 2 {
		fmt.Println("Please specify the command to run")
		flags.Usage()
		os.Exit(1)
	}

	service := args[0]
	args = args[1:]
	execid := fmt.Sprintf("exec%d", os.Getpid())
	log := log.WithFields(log.Fields{
		"service": service,
	})

	client, err := containerd.New(*sock)
	if err != nil {
		log.WithError(err).Fatal("creating containerd client")
	}

	ctx := namespaces.WithNamespace(context.Background(), "default")

	container, err := client.LoadContainer(ctx, service)
	if err != nil {
		log.WithError(err).Fatal("loading container")
	}
	spec, err := container.Spec()
	if err != nil {
		log.WithError(err).Fatal("loading spec")
	}
	task, err := container.Task(ctx, nil)
	if err != nil {
		log.WithError(err).Fatal("getting task")
	}

	pspec := spec.Process
	pspec.Terminal = *tty
	pspec.Args = args

	io := containerd.Stdio
	if *tty {
		io = containerd.StdioTerminal
	}

	process, err := task.Exec(ctx, execid, pspec, io)
	if err != nil {
		log.WithError(err).Fatal("exec failed")
	}
	defer process.Delete(ctx)

	statusC := make(chan uint32, 1)
	go func() {
		status, err := process.Wait(ctx)
		if err != nil {
			log.WithError(err).Error("wait process")
		}
		statusC <- status
	}()
	var con console.Console
	if *tty {
		con = console.Current()
		defer con.Reset()
		if err := con.SetRaw(); err != nil {
			log.WithError(err).Fatal("setting console raw")
		}
	}
	if err := process.Start(ctx); err != nil {
		log.WithError(err).Fatal("starting process")
	}
	if *tty {
		if err := handleConsoleResize(ctx, process, con); err != nil {
			log.WithError(err).Fatal("resizing console")
		}
	} else {
		sigc := forwardAllSignals(ctx, process)
		defer stopCatch(sigc)
	}
	status := <-statusC
	if status != 0 {
		log.Errorf("Exited with code %d", int(status))
	}
}

func forwardAllSignals(ctx context.Context, task killer) chan os.Signal {
	sigc := make(chan os.Signal, 128)
	signal.Notify(sigc)
	go func() {
		for s := range sigc {
			log.Debug("forwarding signal ", s)
			if err := task.Kill(ctx, s.(syscall.Signal)); err != nil {
				log.WithError(err).Errorf("forward signal %s", s)
			}
		}
	}()
	return sigc
}

func stopCatch(sigc chan os.Signal) {
	signal.Stop(sigc)
	close(sigc)
}

func handleConsoleResize(ctx context.Context, task resizer, con console.Console) error {
	// do an initial resize of the console
	size, err := con.Size()
	if err != nil {
		return err
	}
	if err := task.Resize(ctx, uint32(size.Width), uint32(size.Height)); err != nil {
		log.WithError(err).Error("resize pty")
	}
	s := make(chan os.Signal, 16)
	signal.Notify(s, unix.SIGWINCH)
	go func() {
		for range s {
			size, err := con.Size()
			if err != nil {
				log.WithError(err).Error("get pty size")
				continue
			}
			if err := task.Resize(ctx, uint32(size.Width), uint32(size.Height)); err != nil {
				log.WithError(err).Error("resize pty")
			}
		}
	}()
	return nil
}
