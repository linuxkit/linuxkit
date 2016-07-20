package main

import (
	"archive/tar"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

const (
	dockerSock    = "/var/run/docker.sock"
	lgtm          = "LGTM"
	httpMagicPort = ":44554" // chosen arbitrarily due to IANA availability -- might change
)

var (
	cloudCaptures = []Capturer{}
)

func init() {
	for _, c := range commonCmdCaptures {
		cloudCaptures = append(cloudCaptures, c)
	}
}

// HTTPDiagnosticListener sets a health check and optional diagnostic endpoint
// for cloud editions.
type HTTPDiagnosticListener struct{}

func (h HTTPDiagnosticListener) Listen() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat(dockerSock); os.IsNotExist(err) {
			http.Error(w, "Docker socket not found -- daemon is down", http.StatusServiceUnavailable)
			return
		}
		if _, err := w.Write([]byte(lgtm)); err != nil {
			log.Println("Error writing HTTP success response:", err)
			return
		}
	})

	http.HandleFunc("/diagnose", func(w http.ResponseWriter, r *http.Request) {
		dir, err := ioutil.TempDir("", "diagnostics")
		if err != nil {
			log.Println("Error creating temp dir on diagnostic request:", err)
			return
		}

		file, err := ioutil.TempFile(dir, "diagnostics")
		if err != nil {
			log.Println("Error creating temp file on diagnostic request:", err)
			return
		}

		tarWriter := tar.NewWriter(file)

		Capture(tarWriter, cloudCaptures)

		// TODO: upload written (and gzipped?) tar file to our S3
		// bucket with specific path convention (per-user?  by date?)
	})

	// Start HTTP server to indicate general Docker health.
	// TODO: no magic port?
	http.ListenAndServe(httpMagicPort, nil)
}
