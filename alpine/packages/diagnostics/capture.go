package main

import (
	"archive/tar"
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var (
	commonCmdCaptures = []CommandCapturer{
		{"/bin/date", nil},
		{"/bin/uname", []string{"-a"}},
		{"/bin/ps", []string{"uax"}},
		{"/bin/netstat", []string{"-tulpn"}},
		{"/sbin/iptables-save", nil},
		{"/sbin/ifconfig", nil},
		{"/sbin/route", nil},
		{"/usr/sbin/brctl", nil},
		{"/bin/dmesg", nil},
		{"/usr/bin/docker", []string{"ps"}},
		{"/usr/bin/tail", []string{"-100", "/var/log/docker.log"}},
		{"/usr/bin/tail", []string{"-100", "/var/log/messages"}},
		{"/bin/mount", nil},
		{"/bin/df", nil},
		{"/bin/ls", []string{"-l", "/var"}},
		{"/bin/ls", []string{"-l", "/var/lib"}},
		{"/bin/ls", []string{"-l", "/var/lib/docker"}},
		{"/usr/bin/diagnostics", nil},
		{"/bin/ping", []string{"-w", "5", "8.8.8.8"}},
		{"/bin/cat", []string{"/etc/docker/daemon.json"}},
		{"/bin/cat", []string{"/etc/network/interfaces"}},
		{"/bin/cat", []string{"/etc/resolv.conf"}},
		{"/bin/cat", []string{"/etc/sysctl.conf"}},
		{"/usr/bin/dig", []string{"docker.com"}},
		{"/usr/bin/dig", []string{"@8.8.8.8", "docker.com"}},
		{"/usr/bin/wget", []string{"-O", "-", "http://www.docker.com/"}},
		{"/usr/bin/wget", []string{"-O", "-", "http://104.239.220.248/"}}, // a www.docker.com address
		{"/usr/bin/wget", []string{"-O", "-", "http://216.58.213.68/"}},   // a www.google.com address
		{"/usr/bin/wget", []string{"-O", "-", "http://91.198.174.192/"}},  // a www.wikipedia.com address
	}
	localCmdCaptures = []CommandCapturer{
		{"/usr/bin/tail", []string{"-100", "/var/log/proxy-vsockd.log"}},
		{"/usr/bin/tail", []string{"-100", "/var/log/service-port-opener.log"}},
		{"/usr/bin/tail", []string{"-100", "/var/log/vsudd.log"}},
	}
	localCaptures = []Capturer{NewDatabaseCapturer()}
)

func init() {
	for _, c := range localCmdCaptures {
		localCaptures = append(localCaptures, c)
	}
}

// Capturer defines behavior for structs which will capture arbitrary
// diagnostic information and write it to a tar archive with a timeout.
type Capturer interface {
	Capture(time.Duration, *tar.Writer)
}

type CommandCapturer struct {
	command string
	args    []string
}

func (cc CommandCapturer) Capture(timeout time.Duration, w *tar.Writer) {
	log.Printf("Running %s", cc.command)
	c := exec.Command(cc.command, cc.args...)
	stdoutPipe, err := c.StdoutPipe()
	if err != nil {
		log.Fatalf("Failed to create stdout pipe: %s", err)
	}
	stderrPipe, err := c.StderrPipe()
	if err != nil {
		log.Fatalf("Failed to create stderr pipe: %s", err)
	}
	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	done := make(chan int)
	go func() {
		io.Copy(&stdoutBuffer, stdoutPipe)
		done <- 0
	}()
	go func() {
		io.Copy(&stderrBuffer, stderrPipe)
		done <- 0
	}()
	var timer *time.Timer
	timer = time.AfterFunc(timeout, func() {
		timer.Stop()
		if c.Process != nil {
			c.Process.Kill()
		}
	})
	_ = c.Run()
	<-done
	<-done
	timer.Stop()

	name := strings.Join(append([]string{path.Base(cc.command)}, cc.args...), " ")

	hdr := &tar.Header{
		Name: name + ".stdout",
		Mode: 0644,
		Size: int64(stdoutBuffer.Len()),
	}
	if err = w.WriteHeader(hdr); err != nil {
		log.Fatalln(err)
	}
	if _, err = w.Write(stdoutBuffer.Bytes()); err != nil {
		log.Fatalln(err)
	}
	hdr = &tar.Header{
		Name: name + ".stderr",
		Mode: 0644,
		Size: int64(stderrBuffer.Len()),
	}
	if err = w.WriteHeader(hdr); err != nil {
		log.Fatalln(err)
	}
	if _, err = w.Write(stderrBuffer.Bytes()); err != nil {
		log.Fatalln(err)
	}
}

type DatabaseCapturer struct {
	*CommandCapturer
}

func NewDatabaseCapturer() DatabaseCapturer {
	return DatabaseCapturer{
		&CommandCapturer{
			command: "/bin/cat",
		},
	}
}

func (dc DatabaseCapturer) Capture(timeout time.Duration, w *tar.Writer) {
	// Dump the database
	dbBase := "/Database/branch/master/ro"
	filepath.Walk(dbBase, func(path string, f os.FileInfo, err error) error {
		if f.Mode().IsRegular() {
			dc.CommandCapturer.args = []string{path}
			dc.CommandCapturer.Capture(timeout, w)
		}
		return nil
	})
}

// Capture is the outer level wrapper function to trigger the capturing of
// information.  Clients are expected to call it with a slice of Capturers
// which define the information to be captured.  By using an interface we can
// flexibly define various capture actions for the various listeners.
//
// It is a passed a tar.Writer which the results of the capture will be written
// to.
func Capture(w *tar.Writer, captures []Capturer) {
	t := 2 * time.Second

	for _, c := range captures {
		c.Capture(t, w)
	}
}
