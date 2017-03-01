package server

import (
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	log "github.com/Sirupsen/logrus"
)

// StartPluginAtPath starts an HTTP server listening on a unix socket at the specified path.
// Returns a channel to signal stop when closed, a channel to block on stopping, and an error if occurs.
func StartPluginAtPath(socketPath string, endpoint http.Handler) (chan<- struct{}, <-chan error, error) {
	log.Infoln("Listening at:", socketPath)

	engineStop, engineStopped, err := runHTTP(socketPath, &http.Server{Handler: endpoint, Addr: socketPath})
	if err != nil {
		return nil, nil, err
	}

	shutdownTasks := []func() error{}

	shutdownTasks = append(shutdownTasks, func() error {
		// close channels that others may block on for shutdown
		close(engineStop)
		err := <-engineStopped
		return err
	})

	// Triggers to start shutdown sequence
	fromKernel := make(chan os.Signal, 1)

	// kill -9 is SIGKILL and is uncatchable.
	signal.Notify(fromKernel, syscall.SIGHUP)  // 1
	signal.Notify(fromKernel, syscall.SIGINT)  // 2
	signal.Notify(fromKernel, syscall.SIGQUIT) // 3
	signal.Notify(fromKernel, syscall.SIGABRT) // 6
	signal.Notify(fromKernel, syscall.SIGTERM) // 15

	fromUser := make(chan struct{})
	stopped := make(chan error)
	go func(tasks []func() error) {
		defer close(stopped)

		select {
		case <-fromKernel:
		case <-fromUser:
		}
		for _, task := range tasks {
			if err := task(); err != nil {
				stopped <- err
				return
			}
		}
		return
	}(shutdownTasks)

	return fromUser, stopped, nil
}

// Runs the http server.  This server offers more control than the standard go's default http server.
// When the returned stop channel is closed, a clean shutdown and shutdown tasks are executed.
// The return value is a channel that can be used to block on.  An error is received if server shuts
// down in error; or a nil is received on a clean signalled shutdown.
func runHTTP(socketPath string, server *http.Server) (chan<- struct{}, <-chan error, error) {
	listener, err := net.Listen("unix", socketPath)

	if err != nil {
		return nil, nil, err
	}

	if _, err = os.Lstat(socketPath); err == nil {
		// Update socket filename permission
		if err := os.Chmod(server.Addr, 0777); err != nil {
			return nil, nil, err
		}
	} else {
		return nil, nil, err
	}

	stop := make(chan struct{})
	stopped := make(chan error)

	userInitiated := new(bool)
	go func() {
		<-stop
		*userInitiated = true
		listener.Close()
	}()

	go func() {
		// Serve will block until an error (e.g. from shutdown, closed connection) occurs.
		err := server.Serve(listener)

		defer close(stopped)

		switch {
		case !*userInitiated && err != nil:
			panic(err)
		case *userInitiated:
			stopped <- nil
		default:
			stopped <- err
		}
	}()
	return stop, stopped, nil
}
