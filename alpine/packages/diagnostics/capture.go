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

var (
	commonCmdCaptures = []CommandCapturer{
		{"/bin/date", nil},
		{"/bin/uname", []string{"-a"}},
		{"/bin/ps", []string{"uax"}},
		{"/bin/netstat", []string{"-tulpn"}},
		{"/sbin/iptables-save", nil},
		{"/sbin/ifconfig", nil},
		{"/sbin/route", nil},
		{"/usr/sbin/brctl", []string{"show"}},
		{"/bin/dmesg", nil},
		{"/usr/bin/docker", []string{"ps"}},
		{"/usr/bin/tail", []string{"-20000", "/var/log/docker.log"}},
		{"/usr/bin/tail", []string{"-20000", "/var/log/messages"}},
		{"/bin/mount", nil},
		{"/bin/df", []string{"-h"}},
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
		{"/usr/bin/tail", []string{"-100", "/var/log/vsudd.log"}},
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
	Capture(time.Duration, *tar.Writer)
}

type CommandCapturer struct {
	command string
	args    []string
}

func (cc CommandCapturer) Capture(timeout time.Duration, w *tar.Writer) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	done := make(chan struct{})

	name := strings.Join(append([]string{path.Base(cc.command)}, cc.args...), " ")
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
