package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/plugin"
	"github.com/docker/infrakit/plugin/util"
	"github.com/gorilla/mux"
)

// BuildHandler returns a http handler from a list of functions that return an endpoint and handler pair.
// There's special handling of errors from plugin.  In general, the handler should just propagate the error
// from the plugin.  If the plugin returns an error, the handler will wrap the error message in a structure
// and return with HTTP status code 400 (BAD REQUEST).  This will give the client Callable a hint which will
// try to look for a json of the form `{"error":"message"}`.  If found in the body, the Callable client will
// instead return an error with error.Error() == message to the caller.
func BuildHandler(endpoints []func() (plugin.Endpoint, plugin.Handler)) http.Handler {

	router := mux.NewRouter()
	router.StrictSlash(true)

	for _, f := range endpoints {

		endpoint, serve := f()

		ep, err := util.GetHTTPEndpoint(endpoint)
		if err != nil {
			panic(err) // This is system initialization so we have to panic
		}

		router.HandleFunc(ep.Path, func(resp http.ResponseWriter, req *http.Request) {
			defer func() {
				req.Body.Close()
				if err := recover(); err != nil {
					log.Errorf("%s: %s", err, debug.Stack())
					respond(http.StatusInternalServerError, err, resp)
					return
				}
			}()
			result, err := serve(mux.Vars(req), req.Body)

			// Returns a structure for the error and unmarshal on the other side.
			if err != nil {
				respond(http.StatusBadRequest, err, resp)
				return
			}
			respond(http.StatusOK, result, resp)
			return
		}).Methods(strings.ToUpper(ep.Method))
	}
	return router
}

func respond(status int, body interface{}, resp http.ResponseWriter) {
	if body != nil {
		var bodyJSON []byte

		switch body := body.(type) {

		case error:
			message := strings.Replace(body.Error(), "\"", "'", -1)
			bodyJSON = []byte(fmt.Sprintf(`{"error": "%s"}`, message))

		default:
			buff, err := json.Marshal(body)
			if err != nil {
				status = http.StatusInternalServerError
				message := strings.Replace(fmt.Sprintf("can't marshal:%v", body), "\"", "'", -1)
				bodyJSON = []byte(fmt.Sprintf(`{"error": "%s"}`, message))
				log.Warn("Failed to marshal response body %v: %s", body, err.Error())
			} else {
				bodyJSON = buff
			}
		}
		resp.WriteHeader(status)
		resp.Header().Set("Content-Type", "application/json")
		resp.Write(bodyJSON)
	} else {
		resp.WriteHeader(status)
	}
}
