package main

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	healthcheckTimeout = 5 * time.Second
	dockerSock         = "/var/run/docker.sock"
	lgtm               = "LGTM"
	httpMagicPort      = ":44554" // chosen arbitrarily due to IANA availability -- might change
	bucket             = "editionsdiagnostics"
	sessionIDField     = "session"
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

// Listen starts the HTTPDiagnosticListener and sets up handlers for its endpoints
func (h HTTPDiagnosticListener) Listen() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), healthcheckTimeout)
		defer cancel()

		// Vendoring the Docker Go client is overkill for this
		// small program, and we can fairly safely rely on the
		// `docker` client's existence locally, so we just
		// shell out here.
		cmd := exec.CommandContext(ctx, "docker", "info")
		errCh := make(chan error)

		go func() {
			_, err := cmd.CombinedOutput()
			errCh <- err
		}()

		select {
		case err := <-errCh:
			if err != nil {
				http.Error(w, "Docker daemon ping error", http.StatusInternalServerError)
				return
			}
		case <-ctx.Done():
			http.Error(w, "Docker daemon ping timed out", http.StatusServiceUnavailable)
			return
		}

		if _, err := w.Write([]byte(lgtm)); err != nil {
			log.Println("Error writing HTTP success response:", err)
			return
		}
	})

	http.HandleFunc("/diagnose", func(w http.ResponseWriter, r *http.Request) {
		diagnosticsSessionID := r.FormValue(sessionIDField)

		if diagnosticsSessionID == "" {
			http.Error(w, "No 'session' field specified for diagnostics run", http.StatusBadRequest)
			return
		}

		hostname, err := os.Hostname()
		if err != nil {
			http.Error(w, "Error getting hostname:"+err.Error(), http.StatusInternalServerError)
			return
		}

		// To keep URL cleaner
		hostname = strings.Replace(hostname, ".", "-", -1)

		if _, err := w.Write([]byte("OK hostname=" + hostname + " session=" + diagnosticsSessionID + "\n")); err != nil {
			http.Error(w, "Error writing: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Do the actual capture and uplaod to S3 in the background.
		// No need to make caller sit and wait for the result.  They
		// probably have a lot of other things going on, like other
		// servers to request diagnostics for.
		//
		// TODO(nathanleclaire): Potentially, endpoint to check the
		// result of this capture and upload process as well.
		go func() {
			dir, err := ioutil.TempDir("", "diagnostics")
			if err != nil {
				log.Println("Error creating temp dir on diagnostic request:: ", err)
				return
			}

			file, err := ioutil.TempFile(dir, "diagnostics")
			if err != nil {
				log.Println("Error creating temp file on diagnostic request:", err)
				return
			}

			tarWriter := tar.NewWriter(file)

			Capture(tarWriter, cloudCaptures)

			if err := tarWriter.Close(); err != nil {
				log.Println("Error closing archive writer: ", err)
				return
			}

			if err := file.Close(); err != nil {
				log.Println("Error closing file: ", err)
				return
			}

			readFile, err := os.Open(file.Name())
			if err != nil {
				log.Println("Error opening report file to upload: ", err)
				return
			}
			defer readFile.Close()

			buf := &bytes.Buffer{}
			contentLength, err := io.Copy(buf, readFile)
			if err != nil {
				log.Println("Error copying to buffer: ", err)
				return
			}

			reportURI := fmt.Sprintf("https://%s.s3.amazonaws.com/%s-%s.tar", bucket, diagnosticsSessionID, hostname)

			uploadReq, err := http.NewRequest("PUT", reportURI, buf)
			if err != nil {
				log.Println("Error getting bucket request: ", err)
				return
			}

			uploadReq.Header.Set("x-amz-acl", "bucket-owner-full-control")
			uploadReq.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
			uploadReq.Header.Set("Content-Length", strconv.Itoa(int(contentLength)))

			client := &http.Client{}

			if uploadResp, err := client.Do(uploadReq); err != nil {
				log.Println("Error writing: ", err)
				body, err := ioutil.ReadAll(uploadResp.Body)
				if err != nil {
					log.Println("Error reading response body: ", err)
				}
				log.Println(string(body))
				return
			}
			log.Println("No error sending S3 request")
			log.Println("Diagnostics request finished")
		}()
	})

	// Start HTTP server to indicate general Docker health.
	// TODO: no magic port?
	http.ListenAndServe(httpMagicPort, nil)
}
