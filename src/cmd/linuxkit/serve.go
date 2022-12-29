package main

import (
	"net/http"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func logRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Infof("%s %s", r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func serveCmd() *cobra.Command {
	var (
		port string
		dir  string
	)
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "serve a directory over http",
		Long:  `Serve a directory over http.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			http.Handle("/", http.FileServer(http.Dir(dir)))
			log.Fatal(http.ListenAndServe(port, logRequest(http.DefaultServeMux)))

			return nil
		},
	}

	cmd.Flags().StringVar(&port, "port", ":8080", "Local port to serve on")
	cmd.Flags().StringVar(&dir, "directory", ".", "Directory to serve")

	return cmd
}
