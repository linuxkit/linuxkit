package metadata

import (
	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/pkg/spi/metadata"
	"github.com/docker/infrakit/pkg/types"
)

// NewPluginFromData creates a plugin out of a simple data map.  Note the updates to the map
// is not guarded and synchronized with the reads.
func NewPluginFromData(data map[string]interface{}) metadata.Plugin {
	return &plugin{data: data}
}

// NewPluginFromChannel returns a plugin implementation where reads and writes are serialized
// via channel of functions that have a view to the metadata.  Closing the write channel stops
// the serialized read/writes and falls back to unserialized reads.
func NewPluginFromChannel(writes <-chan func(map[string]interface{})) metadata.Plugin {

	readChan := make(chan func(map[string]interface{}))
	p := &plugin{reads: readChan}

	go func() {

		defer func() {
			if r := recover(); r != nil {
				log.Warningln("Plugin stopped:", r)
			}
		}()

		data := map[string]interface{}{}
		for {
			select {
			case writer, open := <-writes:
				if !open {
					close(readChan)
					p.reads = nil
					return
				}
				writer(data)

			case reader := <-p.reads:
				copy := data
				reader(copy)
			}
		}
	}()
	return p
}

type plugin struct {
	data  map[string]interface{}
	reads chan func(data map[string]interface{})
}

// List returns a list of *child nodes* given a path, which is specified as a slice
// where for i > j path[i] is the parent of path[j]
func (p *plugin) List(path metadata.Path) ([]string, error) {
	if p.reads == nil && p.data != nil {
		return List(path, p.data), nil
	}

	children := make(chan []string)

	p.reads <- func(data map[string]interface{}) {
		children <- List(path, data)
		return
	}

	return <-children, nil
}

// Get retrieves the value at path given.
func (p *plugin) Get(path metadata.Path) (*types.Any, error) {
	if p.reads == nil && p.data != nil {
		return types.AnyValue(Get(path, p.data))
	}

	value := make(chan *types.Any)
	err := make(chan error)

	p.reads <- func(data map[string]interface{}) {
		v, e := types.AnyValue(Get(path, data))
		value <- v
		err <- e
		return
	}

	return <-value, <-err
}
