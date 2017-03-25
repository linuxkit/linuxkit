package client

import (
	"bytes"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/pkg/spi"
	"github.com/gorilla/rpc/v2/json2"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"
)

type client struct {
	http http.Client
	addr string
}

// New creates a new Client that communicates with a unix socket and validates the remote API.
func New(socketPath string, api spi.InterfaceSpec) (Client, error) {
	dialUnix := func(proto, addr string) (conn net.Conn, err error) {
		return net.Dial("unix", socketPath)
	}

	unvalidatedClient := &client{addr: socketPath, http: http.Client{Transport: &http.Transport{Dial: dialUnix}}}
	cl := &handshakingClient{client: unvalidatedClient, iface: api, lock: &sync.Mutex{}}

	// check handshake
	if err := cl.handshake(); err != nil {
		// Note - we still return the client with the possibility of doing a handshake later on
		// if we provide an api for the plugin to recheck later.  This way, individual components
		// can stay running and recalibrate themselves after the user has corrected the problems.
		return cl, err
	}
	return cl, nil
}

func (c client) Addr() string {
	return c.addr
}

func (c client) Call(method string, arg interface{}, result interface{}) error {
	message, err := json2.EncodeClientRequest(method, arg)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "http://a/", bytes.NewReader(message))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	requestData, err := httputil.DumpRequest(req, true)
	if err == nil {
		log.Debugf("Sending request %s", string(requestData))
	} else {
		log.Error(err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	responseData, err := httputil.DumpResponse(resp, true)
	if err == nil {
		log.Debugf("Received response %s", string(responseData))
	} else {
		log.Error(err)
	}

	return json2.DecodeClientResponse(resp.Body, result)
}
