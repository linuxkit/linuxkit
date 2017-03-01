package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/plugin"
	"github.com/docker/infrakit/plugin/util"
)

// Client is an HTTP client that communicates via unix sockets.
type Client struct {
	path string
	c    *http.Client
}

// newHTTPClient creates an HTTP client that sends requests to a unix socket.
func newHTTPClient(socketPath string) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			Dial: func(proto, _ string) (conn net.Conn, err error) {
				return net.Dial("unix", socketPath)
			},
		},
	}
}

// New returns an HTTP client hat sends requests to a unix socket.
func New(socketPath string) *Client {
	return &Client{
		path: socketPath,
		c:    newHTTPClient(socketPath),
	}
}

// GetHTTPClient returns the http client
func (d *Client) GetHTTPClient() *http.Client {
	return d.c
}

// String returns a string representation
func (d *Client) String() string {
	return d.path
}

// Call implements the Callable interface.  Makes a call to a supported endpoint.
func (d *Client) Call(endpoint plugin.Endpoint, message, result interface{}) ([]byte, error) {

	ep, err := util.GetHTTPEndpoint(endpoint)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("http://socket%s", ep.Path)

	tee := new(bytes.Buffer)
	var payload, body io.Reader
	if message != nil {
		if buff, err := json.Marshal(message); err == nil {
			payload = bytes.NewBuffer(buff)
		} else {
			return nil, err
		}
		body = io.TeeReader(payload, tee)
	}

	request, err := http.NewRequest(strings.ToUpper(ep.Method), url, body)
	if err != nil {
		return nil, err
	}
	resp, err := d.c.Do(request)

	logrus.Debugln("REQ --", d.path, "url=", url, "request=", string(tee.Bytes()), "err=", err)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	buff, err := ioutil.ReadAll(resp.Body)

	logrus.Debugln("RESP -", d.path, "url=", url, "response=", string(buff), "err=", err)

	switch resp.StatusCode {

	case http.StatusOK:
		if result != nil {
			err = json.Unmarshal(buff, result)
		}
		return buff, err

	case http.StatusBadRequest:
		// try to unmarshal an error structure
		m := struct {
			Error string `json:"error,omitempty"`
		}{}
		err = json.Unmarshal(buff, &m)
		if err == nil && m.Error != "" {
			// found error message
			return nil, errors.New(m.Error)
		}
	}
	return nil, fmt.Errorf("error:%d, msg=%s", resp.StatusCode, string(buff))
}
