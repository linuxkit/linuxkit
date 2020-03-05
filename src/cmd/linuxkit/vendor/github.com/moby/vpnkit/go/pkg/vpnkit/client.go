package vpnkit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/moby/vpnkit/go/pkg/vpnkit/transport"
)

const (
	ListPath         = "/forwards/list"
	ExposePortPath   = "/forwards/expose/port"
	ExposePipePath   = "/forwards/expose/pipe"
	UnexposePortPath = "/forwards/unexpose/port"
	UnexposePipePath = "/forwards/unexpose/pipe"
	DumpStatePath    = "/forwards/dump"
)
const httpTimeout = 120 * time.Second

func NewClient(path string) (Client, error) {
	t := transport.Choose(path)
	return &httpClient{
		client: http.Client{
			Timeout: httpTimeout,
			Transport: &http.Transport{
				DialContext: func(c context.Context, _, _ string) (net.Conn, error) {
					return t.Dial(c, path)
				},
			},
		},
	}, nil
}

type httpClient struct {
	client http.Client
}

// ExposeError should be reported through to the user
type ExposeError struct {
	Message string `json:"message"`
}

func (e *ExposeError) Error() string {
	return e.Message
}

func (h *httpClient) Expose(_ context.Context, port *Port) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(port); err != nil {
		return err
	}
	path := ExposePortPath
	if port.Proto == Unix {
		path = ExposePipePath
	}
	res, err := h.client.Post("http://unix"+path, "application/json", &buf)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode == 400 {
		var exposeError ExposeError
		dec := json.NewDecoder(res.Body)
		if err := dec.Decode(&exposeError); err != nil {
			fmt.Printf("failed to decode: %v\n", err)
			return err
		}
		return &exposeError
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf(path+" returned unexpected status: %d", res.StatusCode)
	}
	return nil
}

func (h *httpClient) Unexpose(_ context.Context, port *Port) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(port); err != nil {
		return err
	}
	path := UnexposePortPath
	if port.Proto == Unix {
		path = UnexposePipePath
	}
	res, err := h.client.Post("http://unix"+path, "application/json", &buf)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf(path+" returned unexpected status: %d", res.StatusCode)
	}
	return nil
}

func (h *httpClient) ListExposed(context.Context) ([]Port, error) {
	res, err := h.client.Get("http://unix" + ListPath)
	if err != nil {
		fmt.Printf("GET failed with %v\n", err)
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(ListPath+" returned unexpected status: %d", res.StatusCode)
	}
	dec := json.NewDecoder(res.Body)
	var ports []Port
	if err := dec.Decode(&ports); err != nil {
		return nil, err
	}
	return ports, nil
}

func (h *httpClient) DumpState(_ context.Context, w io.Writer) error {
	res, err := h.client.Get("http://unix" + DumpStatePath)
	if err != nil {
		fmt.Printf("GET failed with %v\n", err)
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf(DumpStatePath+" returned unexpected status: %d", res.StatusCode)
	}
	_, err = io.Copy(w, res.Body)
	return err
}
