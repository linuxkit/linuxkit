package main

import (
	"context"
	"flag"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const daemonCommand = "containerd"

var (
	daemonState string
	daemonRoot  string
	noDaemon    bool
	con         *connection
	config      = &ContainConfig{
		Namespace: "contain-test",
		Socket:    "/containerd.sock",
		HostMountContainer: []Contain{
			Contain{
				Command:     []string{"ls"},
				Name:        "test",
				Image:       "docker.io/library/alpine:latest",
				Source:      "/etc",
				Destination: "/mnt",
			},
		},
	}
)

func init() {
	flag.StringVar(&config.Socket, "address", config.Socket, "The address to the containerd socket for use in the tests")
	flag.StringVar(&daemonState, "state", daemonState, "")
	flag.BoolVar(&noDaemon, "no-daemon", false, "Do not start a dedicated daemon for the tests")
	flag.StringVar(&daemonRoot, "root", daemonRoot, "")
	flag.Parse()
}

func TestMain(m *testing.M) {
	shutdown := func() {}
	var err error
	if stat, _ := os.Stat(config.Socket); stat == nil && !noDaemon {
		shutdown, err = setupContainerd()
		if err != nil {
			log.Errorln(err)
		}
	}

	con, err = setup(config)
	if err != nil {
		log.Errorln(err)
		return
	}

	log.Infoln("starting tests")
	m.Run()

	shutdown()
}

func setupContainerd() (func(), error) {
	log.Infoln("starting daemon")
	cmd := exec.CommandContext(
		context.Background(),
		daemonCommand,
		"--address", config.Socket,
		"--log-level", "debug")
	if err := cmd.Start(); err != nil {
		cmd.Wait()
		return func() {}, errors.New("failed to start daemon")
	}

	return func() {
		log.Infoln("shutdown containerd")
		done := make(chan error, 1)
		go func() {
			done <- cmd.Wait()
		}()
		select {
		case <-time.After(5 * time.Second):
			if err := cmd.Process.Kill(); err != nil {
				log.Fatal("failed to kill: ", err)
			}
			log.Println("containerd killed as timeout reached")
			os.Remove(config.Socket)
		case err := <-done:
			if err != nil {
				log.Printf("containerd done with error = %v", err)
			} else {
				log.Print("containerd done gracefully without error")
			}
		}
	}, nil
}

func Test_Execute(t *testing.T) {
	type args struct {
		args    []string
		mounter Contain
		con     *connection
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "tester",
			args: args{
				args:    []string{"ls"},
				mounter: config.HostMountContainer[0],
				con:     con,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Execute(tt.args.args, tt.args.mounter, tt.args.con); (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
