package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Listen for Docker image delete events and run a command after a delay.

// Event represents the subset of the Docker event message that we're
// interested in
type Event struct {
	Type   string
	Action string
}

// String returns an Event in a human-readable form
func (e Event) String() string {
	return fmt.Sprintf("Type: %s, Action: %s", e.Type, e.Action)
}

// DelayedAction runs a function in the future at least once after every call
// to AtLeastOnceMore.
type DelayedAction struct {
	c chan interface{}
}

// NewDelayedAction creates a delayed action which guarantees to call f
// at most d after a call to AtLeastOnceMore.
func NewDelayedAction(d time.Duration, f func()) *DelayedAction {
	c := make(chan interface{})
	go func() {
		for {
			<-c
			time.Sleep(d)
			f()
		}
	}()
	return &DelayedAction{
		c: c,
	}
}

// AtLeastOnceMore guarantees to call f at least once more within the originally
// specified duration.
func (a *DelayedAction) AtLeastOnceMore() {
	select {
	case a.c <- nil:
		// Started a fresh countdown
	default:
		// There is already a countdown in progress
	}
}

func main() {
	// after-image-deletes --delay 10s -- /sbin/fstrim /var

	delay := flag.Duration("delay", time.Second*10, "maximum time to wait after an image delete before triggering")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s: run a command after images are deleted by Docker.\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example usage:\n")
		fmt.Fprintf(os.Stderr, "%s --delay 10s -- /sbin/fstrim /var\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "   -- run the command /sbin/fstrim /var at most 10s after an image is deleted.\n")
		fmt.Fprintf(os.Stderr, "      This would allow large batches of image deletions to happen and amortise the\n")
		fmt.Fprintf(os.Stderr, "      cost of the TRIM operation.\n\n")
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		flag.PrintDefaults()
	}

	flag.Parse()
	toRun := flag.Args()
	if len(toRun) == 0 {
		log.Fatalf("Please supply a program to run. For usage add -h")
	}

	log.Printf("I will run %s around %.1f seconds after an image is deleted", strings.Join(toRun, " "), delay.Seconds())

	action := NewDelayedAction(*delay, func() {
		cmdline := strings.Join(toRun, " ")
		log.Printf("Running %s", cmdline)
		cmd := exec.Command(toRun[0], toRun[1:]...)
		err := cmd.Run()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				log.Printf("%s failed: %s", cmdline, string(ee.Stderr))
				return
			}
			log.Printf("Unexpected failure while running: %s: %#v", cmdline, err)
		}
	})

	// Connect to Docker over the Unix domain socket
	httpc := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", "/var/run/docker.sock")
			},
		},
	}

RECONNECT:
	// (Re-)connect forever, reading events
	for {
		res, err := httpc.Get("http://unix/v1.24/events")
		if err != nil {
			log.Printf("Failed to connect to the Docker daemon: will retry in 1s")
			time.Sleep(time.Second)
			continue RECONNECT
		}
		// Check the server identifies as Docker. This will provide an early failure
		// if we're pointed at completely the wrong address.
		server := res.Header.Get("Server")
		if !strings.HasPrefix(server, "Docker") {
			log.Printf("Server identified as %s -- is this really Docker?", server)
			panic(errors.New("Remote server is not Docker"))
		}
		log.Printf("(Re-)connected to the Docker daemon")
		d := json.NewDecoder(res.Body)
		var event Event
		for {
			err = d.Decode(&event)
			if err != nil {
				log.Printf("Failed to read event: will retry in 1s")
				res.Body.Close()
				time.Sleep(time.Second)
				continue RECONNECT
			}
			if event.Action == "delete" && event.Type == "image" {
				log.Printf("The delayed action will happen at least once more")
				action.AtLeastOnceMore()
			}
		}
	}

}
