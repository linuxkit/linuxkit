package group

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/spi/instance"
	"reflect"
	"sync"
	"time"
)

type quorum struct {
	scaled       Scaled
	LogicalIDs   []instance.LogicalID
	pollInterval time.Duration
	stop         chan bool
}

// NewQuorum creates a supervisor for a group of instances operating in a quorum.
func NewQuorum(scaled Scaled, logicalIDs []instance.LogicalID, pollInterval time.Duration) Supervisor {
	return &quorum{
		scaled:       scaled,
		LogicalIDs:   logicalIDs,
		pollInterval: pollInterval,
		stop:         make(chan bool),
	}
}

func (q *quorum) PlanUpdate(scaled Scaled, settings groupSettings, newSettings groupSettings) (updatePlan, error) {

	if !reflect.DeepEqual(settings.config.Allocation.LogicalIDs, newSettings.config.Allocation.LogicalIDs) {
		return nil, errors.New("Logical ID changes to a quorum is not currently supported")
	}

	return &rollingupdate{
		desc: fmt.Sprintf(
			"Performs a rolling update on %d instances",
			len(settings.config.Allocation.LogicalIDs)),
		scaled:     scaled,
		updatingTo: newSettings,
		stop:       make(chan bool),
	}, nil
}

func (q *quorum) Stop() {
	close(q.stop)
}

func (q *quorum) Run() {
	ticker := time.NewTicker(q.pollInterval)

	q.converge()
	for {
		select {
		case <-ticker.C:
			q.converge()

		case <-q.stop:
			ticker.Stop()
			return
		}
	}
}

func (q *quorum) converge() {
	descriptions, err := q.scaled.List()
	if err != nil {
		log.Errorf("Failed to check group: %s", err)
		return
	}

	log.Debugf("Found existing instances: %v", descriptions)

	unknownIPs := []instance.Description{}
	for _, description := range descriptions {
		if description.LogicalID == nil {
			log.Warnf("Instance %s has no logical ID", description.ID)
			continue
		}

		matched := false
		for _, expectedID := range q.LogicalIDs {
			if expectedID == *description.LogicalID {
				matched = true
			}
		}
		if !matched {
			unknownIPs = append(unknownIPs, description)
		}
	}

	grp := sync.WaitGroup{}

	for _, unknownInstance := range unknownIPs {
		log.Warnf("Destroying instances with unknown IP address: %+v", unknownInstance)

		grp.Add(1)
		go func() {
			defer grp.Done()
			q.scaled.Destroy(unknownInstance.ID)
		}()
	}

	missingIDs := []instance.LogicalID{}
	for _, expectedID := range q.LogicalIDs {
		matched := false
		for _, description := range descriptions {
			if description.LogicalID == nil {
				continue
			}

			if expectedID == *description.LogicalID {
				matched = true
			}
		}
		if !matched {
			missingIDs = append(missingIDs, expectedID)
		}
	}

	for _, missingID := range missingIDs {
		log.Infof("Logical ID %s is missing, provisioning new instance", missingID)
		id := missingID

		grp.Add(1)
		go func() {
			defer grp.Done()

			q.scaled.CreateOne(&id)
		}()
	}

	grp.Wait()
}
