package main

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
)

const (
	systemLogDir         = "editionslogs"
	systemContainerLabel = "com.docker.editions.system"
)

// SystemContainerCapturer gets the logs from containers which are run
// specifically for Docker Editions.
type SystemContainerCapturer struct{}

// Capture writes output from a CommandCapturer to a tar archive
func (s SystemContainerCapturer) Capture(parentCtx context.Context, w *tar.Writer) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultCaptureTimeout)
	defer cancel()

	errCh := make(chan error)

	go func() {
		// URL encoded.
		// Reads more like this:
		// /v1.25/containers/json?all=1&filters={"label":{"com.docker.editions.system":true}}
		resp, err := dockerHTTPGet(ctx, "/containers/json?all=1&filters=%7B%22label%22%3A%7B%22"+systemContainerLabel+"%22%3Atrue%7D%7D")
		if err != nil {
			errCh <- err
			return
		}

		defer resp.Body.Close()

		names := []struct {
			ID    string `json:"id"`
			Names []string
		}{}

		if err := json.NewDecoder(resp.Body).Decode(&names); err != nil {
			errCh <- err
			return
		}

		for _, c := range names {
			resp, err := dockerHTTPGet(ctx, "/containers/"+c.ID+"/logs?stderr=1&stdout=1&timestamps=1&tail=all")
			if err != nil {
				log.Println("ERROR (get request):", err)
				continue
			}

			defer resp.Body.Close()

			logLines, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println("ERROR (reading response):", err)
				continue
			}

			// Docker API returns a Names array where the names are
			// like this: ["/foobar", "/quux"]
			//
			// Use the first one from it.
			//
			// Additionally, the slash from it helps delimit a path
			// separator in the tar archive.
			//
			// TODO(nathanleclaire): This seems fragile, but I'm
			// not sure what approach would be much better.
			tarWrite(w, bytes.NewBuffer(logLines), systemLogDir+c.Names[0])
		}

		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		log.Println("System container log capture context error", ctx.Err())
	case err := <-errCh:
		if err != nil {
			log.Println("System container log capture error", err)
		}
		log.Println("System container log capture finished successfully")
	}
}
