package server

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type handler int

func TestUnixSocketServer(t *testing.T) {

	servedCall := make(chan struct{})

	router := mux.NewRouter()
	router.HandleFunc("/test", func(resp http.ResponseWriter, req *http.Request) {
		close(servedCall)
		return
	})

	socket := filepath.Join(os.TempDir(), fmt.Sprintf("%d.sock", time.Now().Unix()))
	stop, errors, err := StartPluginAtPath(socket, router)

	require.NoError(t, err)

	client := &http.Client{
		Transport: &http.Transport{
			Dial: func(proto, addr string) (conn net.Conn, err error) {
				return net.Dial("unix", socket)
			},
		},
	}
	resp, err := client.Get("http://local/test")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	<-servedCall

	// Now we stop the server
	close(stop)

	// We shouldn't block here.
	<-errors
}
