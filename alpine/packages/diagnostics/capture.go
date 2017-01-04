package main

import (
	"archive/tar"
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultCommandTimeout = 5 * time.Second

	// Might eventually have some pretty long (~30s) traces in here, so 35
	// seconds seems reasonable.
	allCaptureTimeout = 35 * time.Second
)

var (
	commonCmdCaptures = []CommandCapturer{
		{"/bin/date", nil, defaultCommandTimeout},
		{"/bin/uname", []string{"-a"}, defaultCommandTimeout},
		{"/bin/ps", []string{"uax"}, defaultCommandTimeout},
		{"/bin/netstat", []string{"-tulpn"}, defaultCommandTimeout},
		{"/sbin/iptables-save", nil, defaultCommandTimeout},
		{"/sbin/ifconfig", nil, defaultCommandTimeout},
		{"/sbin/route", nil, defaultCommandTimeout},
		{"/usr/sbin/brctl", []string{"show"}, defaultCommandTimeout},
		{"/bin/dmesg", nil, defaultCommandTimeout},
		{"/usr/bin/docker", []string{"ps"}, defaultCommandTimeout},
		{"/usr/bin/docker", []string{"version"}, defaultCommandTimeout},
		{"/usr/bin/docker", []string{"info"}, defaultCommandTimeout},
		{"/usr/bin/docker", []string{"network", "ls"}, defaultCommandTimeout},
		{"/usr/bin/docker", []string{"node", "ls"}, defaultCommandTimeout},
		{"/usr/bin/docker", []string{"service", "ls"}, defaultCommandTimeout},
		{"/usr/bin/tail", []string{"-20000", "/var/log/docker.log"}, defaultCommandTimeout},
		{"/usr/bin/tail", []string{"-20000", "/var/log/messages"}, defaultCommandTimeout},
		{"/bin/mount", nil, defaultCommandTimeout},
		{"/bin/df", []string{"-h"}, defaultCommandTimeout},
		{"/bin/ls", []string{"-l", "/var"}, defaultCommandTimeout},
		{"/bin/ls", []string{"-l", "/var/lib"}, defaultCommandTimeout},
		{"/bin/ls", []string{"-l", "/var/lib/docker"}, defaultCommandTimeout},
		{"/usr/bin/diagnostics", nil, defaultCommandTimeout},
		{"/bin/ping", []string{"-w", "5", "8.8.8.8"}, 6 * time.Second},
		{"/bin/cat", []string{"/etc/docker/daemon.json"}, defaultCommandTimeout},
		{"/bin/cat", []string{"/etc/network/interfaces"}, defaultCommandTimeout},
		{"/bin/cat", []string{"/etc/resolv.conf"}, defaultCommandTimeout},
		{"/bin/cat", []string{"/etc/sysctl.conf"}, defaultCommandTimeout},
		{"/bin/cat", []string{"/proc/cpuinfo"}, defaultCommandTimeout},
		{"/usr/bin/nslookup", []string{"docker.com"}, defaultCommandTimeout},
		{"/usr/bin/nslookup", []string{"docker.com", "8.8.8.8"}, defaultCommandTimeout},
		{"/usr/bin/curl", []string{"http://www.docker.com/"}, defaultCommandTimeout},
		{"/usr/bin/curl", []string{"http://104.239.220.248/"}, defaultCommandTimeout},              // a www.docker.com address
		{"/usr/bin/curl", []string{"http://216.58.213.68/"}, defaultCommandTimeout},                // a www.google.com address
		{"/usr/bin/curl", []string{"http://91.198.174.192/"}, defaultCommandTimeout},               // a www.wikipedia.com address
		{"/bin/cat", []string{"/var/lib/docker/volumes/metadata.db"}, defaultCommandTimeout},       // [docker/docker#29636]
		{"/bin/cat", []string{"/var/lib/docker/network/files/local-kv.db"}, defaultCommandTimeout}, // [docker/docker#29636]
	}
	localCmdCaptures = []CommandCapturer{
		{"/usr/bin/tail", []string{"-100", "/var/log/proxy-vsockd.log"}, defaultCommandTimeout},
		{"/usr/bin/tail", []string{"-100", "/var/log/vsudd.log"}, defaultCommandTimeout},
	}
	localCaptures = []Capturer{NewDatabaseCapturer()}
)

func init() {
	for _, c := range commonCmdCaptures {
		localCaptures = append(localCaptures, c)
	}
	for _, c := range localCmdCaptures {
		localCaptures = append(localCaptures, c)
	}
}

// Capturer defines behavior for structs which will capture arbitrary
// diagnostic information and write it to a tar archive with a timeout.
type Capturer interface {
	Capture(context.Context, *tar.Writer)
}

type CommandCapturer struct {
	command string
	args    []string
	timeout time.Duration
}

func (cc CommandCapturer) Capture(parentCtx context.Context, w *tar.Writer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	done := make(chan struct{})

	name := strings.Join(append([]string{path.Base(cc.command)}, cc.args...), " ")
	ctx, cancel := context.WithTimeout(parentCtx, cc.timeout)
	cmd := exec.CommandContext(ctx, cc.command, cc.args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	go runCmd(cmd, done)

	select {
	case <-ctx.Done():
		log.Println("ERROR:", ctx.Err())
	case <-done:
		tarWrite(w, stdout, name+".stdout")
		tarWrite(w, stderr, name+".stderr")
	}

	cancel()
}

// TODO(nathanleclaire): Is the user of log.Fatalln in this function really the
// right choice?  i.e., should the program really exit on failure here?
func tarWrite(w *tar.Writer, buf *bytes.Buffer, headerName string) {
	contents, err := ioutil.ReadAll(buf)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("HEADER:", headerName)
	log.Println("{")
	contentLines := strings.Split(string(contents), "\n")
	for _, line := range contentLines {
		log.Println(line)
	}
	log.Println("}")

	hdr := &tar.Header{
		Name: headerName,
		Mode: 0644,
		Size: int64(len(contents)),
	}
	if err := w.WriteHeader(hdr); err != nil {
		log.Fatalln(err)
	}
	if _, err := w.Write(contents); err != nil {
		log.Fatalln(err)
	}
}

func runCmd(cmd *exec.Cmd, done chan<- struct{}) {
	if err := cmd.Run(); err != nil {
		log.Println("ERROR:", err)
	}
	done <- struct{}{}
}

type DatabaseCapturer struct {
	*CommandCapturer
}

func NewDatabaseCapturer() DatabaseCapturer {
	return DatabaseCapturer{
		&CommandCapturer{
			command: "/bin/cat",
			timeout: defaultCommandTimeout,
		},
	}
}

func (dc DatabaseCapturer) Capture(parentCtx context.Context, w *tar.Writer) {
	// Dump the database
	dbBase := "/Database/branch/master/ro"
	filepath.Walk(dbBase, func(path string, f os.FileInfo, err error) error {
		if f.Mode().IsRegular() {
			dc.CommandCapturer.args = []string{path}
			dc.CommandCapturer.Capture(parentCtx, w)
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
	allCaptureCtx, cancel := context.WithTimeout(context.Background(), allCaptureTimeout)
	done := make(chan struct{})

	go func(captures []Capturer, ctx context.Context, done chan<- struct{}) {
		for _, c := range captures {
			c.Capture(ctx, w)
		}
		done <- struct{}{}
	}(captures, allCaptureCtx, done)

	select {
	case <-allCaptureCtx.Done():
		log.Println("Global command context", allCaptureCtx.Err())
	case <-done:
		log.Println("Captures all finished")
	}

	cancel()
}
