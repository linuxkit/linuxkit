package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Infof("%s %s", r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

// serve starts a local web server
func serve(args []string) {
	flags := flag.NewFlagSet("serve", flag.ExitOnError)
	invoked := filepath.Base(os.Args[0])
	flags.Usage = func() {
		fmt.Printf("USAGE: %s serve [options]\n\n", invoked)
		fmt.Printf("Options:\n\n")
		flags.PrintDefaults()
	}
	portFlag := flags.String("port", ":8080", "Local port to serve on")
	dirFlag := flags.String("directory", ".", "Directory to serve")

	http.Handle("/", http.FileServer(http.Dir(*dirFlag)))
	log.Fatal(http.ListenAndServe(*portFlag, logRequest(http.DefaultServeMux)))
}
