package server

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/armon/go-radix"
	"github.com/docker/infrakit/pkg/types"
)

// the amount of time to wait when pushing a message to
// a slow client or a client that closed after `range clients` started.
const patience time.Duration = time.Second * 1

type subscription struct {
	topic      string
	exactMatch bool
	ch         chan []byte
}

type event struct {
	topic string
	data  []byte
}

// Broker is the event message broker
type Broker struct {

	// Close this to stop
	stop chan struct{}

	// Close this to stop the run loop
	finish chan struct{}

	// Events are pushed to this channel by the main events-gathering routine
	notifier chan *event

	// New client connections
	newClients chan subscription

	// Closed client connections
	closingClients chan subscription

	// Client connections registry
	clients *radix.Tree

	// how many clients
	count int
}

// NewBroker returns an instance of the broker
func NewBroker() *Broker {
	b := &Broker{
		stop:           make(chan struct{}),
		finish:         make(chan struct{}),
		notifier:       make(chan *event, 1),
		newClients:     make(chan subscription),
		closingClients: make(chan subscription),
		clients:        radix.New(),
	}
	go b.run()
	return b
}

// Stop stops the broker and exits the goroutine
func (b *Broker) Stop() {
	close(b.stop)
}

func clean(topic string) string {
	if len(topic) == 0 || topic == "." {
		return "/"
	}

	if topic[0] != '/' {
		return "/" + topic
	}
	return topic
}

//Check the topic ends with `/`
func checkExactMatch(topic string) bool {
	return strings.LastIndex(topic, "/") != len(topic)-1
}

// Publish publishes a message at the topic
func (b *Broker) Publish(topic string, data interface{}, optionalTimeout ...time.Duration) error {
	any, err := types.AnyValue(data)
	if err != nil {
		return err
	}

	topic = clean(topic)

	if len(optionalTimeout) > 0 {
		select {
		case b.notifier <- &event{topic: topic, data: any.Bytes()}:
		case <-time.After(optionalTimeout[0]):
			return fmt.Errorf("timeout sending %v", topic)
		}
	} else {
		b.notifier <- &event{topic: topic, data: any.Bytes()}
	}

	return nil
}

// ServerHTTP implements the HTTP handler
func (b *Broker) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	defer func() {
		if v := recover(); v != nil {
			log.Warningln("broker.ServeHTTP recovered:", v)
		}
	}()

	topic := clean(req.URL.Query().Get("topic"))

	// flusher is required for streaming
	flusher, ok := rw.(http.Flusher)
	if !ok {
		http.Error(rw, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	// Each connection registers its own message channel with the Broker's connections registry
	messageChan := make(chan []byte)

	// Signal the broker that we have a new connection
	b.newClients <- subscription{topic: topic, exactMatch: checkExactMatch(topic), ch: messageChan}

	// Remove this client from the map of connected clients
	// when this handler exits.
	defer func() {
		b.closingClients <- subscription{topic: topic, ch: messageChan}
	}()

	// Listen to connection close and un-register messageChan
	notify := rw.(http.CloseNotifier).CloseNotify()

	for {
		select {
		case <-b.stop:
			close(b.finish)
			return

		case <-notify:
			return
		default:

			// Write to the ResponseWriter
			// Server Sent Events compatible
			fmt.Fprintf(rw, "data: %s\n\n", <-messageChan)

			// Flush the data immediatly instead of buffering it for later.
			flusher.Flush()
		}
	}
}

func (b *Broker) run() {
	for {
		select {
		case <-b.finish:

			// Disconnect all clients
			b.clients.Walk(
				func(key string, value interface{}) bool {
					chset, ok := value.(map[chan []byte]bool)
					if !ok {
						panic("assert-failed")
					}
					for ch := range chset {
						log.Infoln("Closing client connection at", key, "ch=", ch)
						close(ch)
					}
					return false
				})

			log.Infoln("Broker finished")
			return

		case subscription := <-b.newClients:

			// A new client has connected.
			// Register their message channel
			subs := map[chan []byte]bool{subscription.ch: subscription.exactMatch}
			v, has := b.clients.Get(subscription.topic)
			if has {
				if v, ok := v.(map[chan []byte]bool); !ok {
					panic("assert-failed: not a map of channels")
				} else {
					v[subscription.ch] = subscription.exactMatch
					subs = v
				}
			}

			b.clients.Insert(subscription.topic, subs)
			b.count++
			log.Infof("Connected: topic=%s => %d registered clients, ch=%v", subscription.topic, b.count, subscription.ch)

		case subscription := <-b.closingClients:

			// A client has dettached and we want to stop sending messages
			if v, has := b.clients.Get(subscription.topic); has {
				if subs, ok := v.(map[chan []byte]bool); !ok {
					panic("assert-failed: not a map of channels")

				} else {

					delete(subs, subscription.ch)

					if len(subs) == 0 {
						b.clients.Delete(subscription.topic)
					} else {
						b.clients.Insert(subscription.topic, subs)
					}

					b.count--
					log.Infof("Disconnected: topic=%s => %d registered clients, ch=%v", subscription.topic, b.count, subscription.ch)
				}
			}

		case event, open := <-b.notifier:

			if !open {
				log.Infoln("Stopping broker")
				return
			}

			// Remove any \n because it's meaningful in SSE spec.
			// We could use base64 encode, but it hurts interoperability with browser/ javascript clients.
			data := bytes.Replace(event.data, []byte("\n"), nil, -1)

			b.clients.WalkPath(event.topic,

				func(key string, value interface{}) bool {
					chset, ok := value.(map[chan []byte]bool)
					if !ok {
						panic("assert-failed")
					}

					for ch, exact := range chset {
						if exact && event.topic != key {
							return false
						}
						select {
						case ch <- data:
						case <-time.After(patience):
							log.Print("Skipping client.")
						}
					}
					return false
				})
		}
	}

}
