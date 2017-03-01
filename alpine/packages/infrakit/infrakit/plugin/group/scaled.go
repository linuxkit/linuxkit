package group

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/plugin/group/types"
	"github.com/docker/infrakit/spi/flavor"
	"github.com/docker/infrakit/spi/instance"
	"sync"
)

// Scaled is a collection of instances that can be scaled up and down.
type Scaled interface {
	// CreateOne creates a single instance in the scaled group.  Parameters may be provided to customize behavior
	// of the instance.
	CreateOne(id *instance.LogicalID)

	// Health inspects the current health state of an instance.
	Health(inst instance.Description) flavor.Health

	// Destroy destroys a single instance.
	Destroy(id instance.ID)

	// List returns all instances in the group.
	List() ([]instance.Description, error)
}

type scaledGroup struct {
	settings   groupSettings
	memberTags map[string]string
	lock       sync.Mutex
}

func (s *scaledGroup) changeSettings(settings groupSettings) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.settings = settings
}

func (s *scaledGroup) CreateOne(logicalID *instance.LogicalID) {
	s.lock.Lock()
	defer s.lock.Unlock()

	tags := map[string]string{}
	for k, v := range s.memberTags {
		tags[k] = v
	}

	// Instances are tagged with a SHA of the entire instance configuration to support change detection.
	tags[configTag] = s.settings.config.InstanceHash()

	spec := instance.Spec{
		Tags:       tags,
		LogicalID:  logicalID,
		Properties: s.settings.config.Instance.Properties,
	}

	spec, err := s.settings.flavorPlugin.Prepare(
		types.RawMessage(s.settings.config.Flavor.Properties),
		spec,
		s.settings.config.Allocation)
	if err != nil {
		log.Errorf("Failed to Prepare instance: %s", err)
		return
	}

	id, err := s.settings.instancePlugin.Provision(spec)
	if err != nil {
		log.Errorf("Failed to provision: %s", err)
		return
	}

	volumeDesc := ""
	if len(spec.Attachments) > 0 {
		volumeDesc = fmt.Sprintf(" and attachments %s", spec.Attachments)
	}

	log.Infof("Created instance %s with tags %v%s", *id, spec.Tags, volumeDesc)
}

func (s *scaledGroup) Health(inst instance.Description) flavor.Health {
	s.lock.Lock()
	defer s.lock.Unlock()

	health, err := s.settings.flavorPlugin.Healthy(
		types.RawMessage(s.settings.config.Flavor.Properties),
		inst)
	if err != nil {
		log.Warnf("Failed to check health of instance %s: %s", inst.ID, err)
		return flavor.Unknown
	}
	return health

}

func (s *scaledGroup) Destroy(id instance.ID) {
	s.lock.Lock()
	defer s.lock.Unlock()

	log.Infof("Destroying instance %s", id)
	if err := s.settings.instancePlugin.Destroy(id); err != nil {
		log.Errorf("Failed to destroy %s: %s", id, err)
	}
}

func (s *scaledGroup) List() ([]instance.Description, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.settings.instancePlugin.DescribeInstances(s.memberTags)
}
