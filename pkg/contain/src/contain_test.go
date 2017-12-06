package main

import (
	"context"
	"flag"
	"os/exec"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	daemonCommand = "containerd"
	daemonState   = "/run/containerd"
	daemonRoot    = "var/lib/containerd"
	daemonAddress = "/containerd.sock"
	con           *connection
	config        = &ContainConfig{
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
	flag.StringVar(&daemonAddress, "address", daemonAddress, "The address to the containerd socket for use in the tests")
	flag.StringVar(&daemonState, "state", daemonState, "")
	flag.StringVar(&daemonRoot, "root", daemonRoot, "")
	flag.Parse()
}

func TestMain(m *testing.M) {
	var err error
	log.Infoln("starting daemon")
	cmd := exec.CommandContext(
		context.Background(),
		daemonCommand,
		"--address", daemonAddress,
		"--log-level", "debug")
	if err := cmd.Start(); err != nil {
		cmd.Wait()
		log.Errorln("failed to start daemon")
		return
	}

	con, err = setup(config)
	if err != nil {
		log.Errorln(err)
		return
	}

	log.Infoln("starting tests")
	m.Run()
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
	case err := <-done:
		if err != nil {
			log.Printf("containerd done with error = %v", err)
		} else {
			log.Print("containerd done gracefully without error")
		}
	}
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
