package server

import (
	"net"
	"net/http"
)

// ListenAndServeOnSocket starts a minimal server (mostly for testing) listening at the given unix socket path.
func ListenAndServeOnSocket(socketPath string, optionalURLPattern ...string) (*Broker, error) {
	urlPattern := "/"
	if len(optionalURLPattern) > 0 {
		urlPattern = optionalURLPattern[0]
	}
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, err
	}
	broker := NewBroker()
	mux := http.NewServeMux()
	mux.Handle(urlPattern, broker)
	httpServer := &http.Server{
		Handler: mux,
	}
	go httpServer.Serve(listener)

	return broker, nil
}
