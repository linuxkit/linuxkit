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
	systemContainerLabel = "system"
)

// SystemContainerCapturer gets the logs from containers which are run
// specifically for Docker Editions.
type SystemContainerCapturer struct{}

// Capture writes output from a CommandCapturer to a tar archive
func (s SystemContainerCapturer) Capture(parentCtx context.Context, w *tar.Writer) {
	done := make(chan struct{})

	ctx, cancel := context.WithTimeout(context.Background(), defaultCaptureTimeout)
	defer cancel()

	go func() {
		resp, err := dockerHTTPGet(ctx, "/containers/json?all=1&label="+systemContainerLabel)
		if err != nil {
			log.Println("ERROR:", err)
			return
		}

		defer resp.Body.Close()

		names := []struct {
			ID    string `json:"id"`
			Names []string
		}{}

		if err := json.NewDecoder(resp.Body).Decode(&names); err != nil {
			log.Println("ERROR:", err)
			return
		}

		for _, c := range names {
			resp, err := dockerHTTPGet(ctx, "/containers/"+c.ID+"/logs?stderr=1&stdout=1&timestamps=1")
			if err != nil {
				log.Println("ERROR:", err)
				continue
			}

			defer resp.Body.Close()

			logLines, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println("ERROR:", err)
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

		done <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		log.Println("System container log capture error", ctx.Err())
	case <-done:
		log.Println("System container log capture finished")
	}
}
