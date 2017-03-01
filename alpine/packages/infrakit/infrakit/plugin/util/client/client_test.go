package client

import (
	"encoding/json"
	"github.com/docker/infrakit/plugin/util"
	"github.com/docker/infrakit/plugin/util/server"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"testing"
)

func testRespond(t *testing.T, resp http.ResponseWriter, obj interface{}) {
	buff, err := json.Marshal(obj)
	require.NoError(t, err)
	resp.Write(buff)
	return
}

func read(t *testing.T, req *http.Request, obj interface{}) {
	defer req.Body.Close()
	err := json.NewDecoder(req.Body).Decode(obj)
	require.NoError(t, err)
	return
}

type testRequest struct {
	Name  string
	Count int
}

type testResponse struct {
	Name   string
	Status bool
}

func TestUnixClient(t *testing.T) {

	serverReq := testRequest{
		Name:  "unix-client",
		Count: 1000,
	}

	serverResp := testResponse{
		Name:   "unix-server",
		Status: true,
	}

	router := mux.NewRouter()
	router.HandleFunc("/test", func(resp http.ResponseWriter, req *http.Request) {

		input := testRequest{}
		read(t, req, &input)
		require.Equal(t, serverReq, input)

		testRespond(t, resp, serverResp)
		return
	}).Methods("POST")

	dir, err := ioutil.TempDir("", "infrakit-client-test")
	require.NoError(t, err)

	socketPath := filepath.Join(dir, "server.sock")
	stop, errors, err := server.StartPluginAtPath(socketPath, router)

	require.NoError(t, err)
	require.NotNil(t, stop)
	require.NotNil(t, errors)

	response := testResponse{}
	_, err = New(socketPath).Call(&util.HTTPEndpoint{Method: "post", Path: "/test"}, serverReq, &response)

	require.NoError(t, err)
	require.Equal(t, serverResp, response)

	// Now we stop the server
	close(stop)
}
