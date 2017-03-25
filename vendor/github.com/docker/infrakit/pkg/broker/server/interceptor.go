package server

import (
	"fmt"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
)

// Interceptor implements http Handler and is used to intercept incoming requests to subscribe
// to topics and perform some validations before allowing the subscription to be established.
// Tasks that the interceptor can do include authn and authz, topic validation, etc.
type Interceptor struct {

	// Do is the body of the http handler -- required
	Do http.HandlerFunc

	// Pre is called to before actual subscription to a topic happens.
	// This is the hook where validation and authentication / authorization checks happen.
	Pre func(topic string, headers map[string][]string) error

	// Post is called when the client disconnects.  This is optional.
	Post func(topic string)
}

// ServeHTTP calls the before and after subscribe methods.
func (i *Interceptor) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	topic := clean(req.URL.Query().Get("topic"))

	// need to strip out the / because it was added by the client
	if strings.Index(topic, "/") == 0 {
		topic = topic[1:]
	}

	err := i.Pre(topic, req.Header)
	if err != nil {
		log.Warningln("Error:", err)
		http.Error(rw, err.Error(), getStatusCode(err))
		return
	}

	i.Do.ServeHTTP(rw, req)

	if i.Post != nil {
		i.Post(topic)
	}
}

func getStatusCode(e error) int {
	switch e.(type) {
	case ErrInvalidTopic:
		return http.StatusNotFound
	case ErrNotAuthorized:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

// ErrInvalidTopic is the error raised when topic is invalid
type ErrInvalidTopic string

func (e ErrInvalidTopic) Error() string {
	return fmt.Sprintf("invalid topic: %s", string(e))
}

// ErrNotAuthorized is the error raised when the user isn't authorized
type ErrNotAuthorized string

func (e ErrNotAuthorized) Error() string {
	return fmt.Sprintf("not authorized: %s", string(e))
}
