package group

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/plugin/group/util"
	"github.com/docker/infrakit/spi/instance"
	"sort"
	"sync"
	"time"
)

// Scaler is the spi of the scaler controller which mimics the behavior
// of an autoscaling group / scale set on AWS or Azure.
type Scaler interface {
	util.RunStop
	Size() uint
	SetSize(size uint)
}

type scaler struct {
	scaled       Scaled
	size         uint
	pollInterval time.Duration
	lock         sync.Mutex
	stop         chan bool
}

// NewScalingGroup creates a supervisor that monitors a group of instances on a provisioner, attempting to maintain a
// desired size.
func NewScalingGroup(scaled Scaled, size uint, pollInterval time.Duration) Supervisor {
	return &scaler{
		scaled:       scaled,
		size:         size,
		pollInterval: pollInterval,
		stop:         make(chan bool),
	}
}

func (s *scaler) PlanUpdate(scaled Scaled, settings groupSettings, newSettings groupSettings) (updatePlan, error) {

	sizeChange := int(newSettings.config.Allocation.Size) - int(settings.config.Allocation.Size)

	instances, err := scaled.List()
	if err != nil {
		return nil, err
	}

	desired, undesired := desiredAndUndesiredInstances(instances, newSettings)

	plan := scalerUpdatePlan{
		originalSize: settings.config.Allocation.Size,
		newSize:      newSettings.config.Allocation.Size,
		scaler:       s,
		rollingPlan:  noopUpdate{},
	}

	switch {
	case sizeChange == 0:
		rollCount := len(undesired)

		if rollCount == 0 {
			if settings.config.InstanceHash() == newSettings.config.InstanceHash() {

				// This is a no-op update because:
				//  - the instance configuration is unchanged
				//  - the group contains no instances with an undesired state
				//  - the group size is unchanged
				return &noopUpdate{}, nil
			}

			// This case likely occurs because a group was created in a way that no instances are being
			// created. We proceed with the update here, which will likely only change the target
			// configuration in the scaler.

			plan.desc = "Adjusts the instance configuration, no restarts necessary"
			return &plan, nil
		}

		plan.desc = fmt.Sprintf("Performs a rolling update on %d instances", rollCount)

	case sizeChange < 0:
		rollCount := int(newSettings.config.Allocation.Size) - len(desired)
		if rollCount < 0 {
			rollCount = 0
		}

		if rollCount == 0 {
			plan.desc = fmt.Sprintf(
				"Terminates %d instances to reduce the group size to %d",
				int(sizeChange)*-1,
				newSettings.config.Allocation.Size)
		} else {
			plan.desc = fmt.Sprintf(
				"Terminates %d instances to reduce the group size to %d, "+
					" then performs a rolling update on %d instances",
				int(sizeChange)*-1,
				newSettings.config.Allocation.Size,
				rollCount)
		}

	case sizeChange > 0:
		rollCount := len(undesired)

		if rollCount == 0 {
			plan.desc = fmt.Sprintf(
				"Adds %d instances to increase the group size to %d",
				sizeChange,
				newSettings.config.Allocation.Size)
		} else {
			plan.desc = fmt.Sprintf(
				"Performs a rolling update on %d instances,"+
					" then adds %d instances to increase the group size to %d",
				rollCount,
				sizeChange,
				newSettings.config.Allocation.Size)
		}
	}

	plan.rollingPlan = &rollingupdate{
		scaled:     scaled,
		updatingTo: newSettings,
		stop:       make(chan bool),
	}

	return plan, nil
}

type scalerUpdatePlan struct {
	desc         string
	originalSize uint
	newSize      uint
	rollingPlan  updatePlan
	scaler       *scaler
}

func (s scalerUpdatePlan) Explain() string {
	return s.desc
}

func (s scalerUpdatePlan) Run(pollInterval time.Duration) error {

	// If the number of instances is being decreased, first lower the group size.  This eliminates
	// instances that would otherwise be rolled first, avoiding unnecessary work.
	// We could further optimize by selecting undesired instances to destroy, for example if the
	// scaler already has a mix of desired and undesired instances.
	if s.newSize < s.originalSize {
		s.scaler.SetSize(s.newSize)
	}

	if err := s.rollingPlan.Run(pollInterval); err != nil {
		return err
	}

	// Rolling has completed.  If the update included a group size increase, perform that now.
	if s.newSize > s.originalSize {
		s.scaler.SetSize(s.newSize)
	}

	return nil
}

func (s scalerUpdatePlan) Stop() {
	s.rollingPlan.Stop()
}

func (s *scaler) SetSize(size uint) {
	s.lock.Lock()
	defer s.lock.Unlock()

	log.Infof("Set target size to %d", size)
	s.size = size
}

func (s *scaler) getSize() uint {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.size
}

func (s *scaler) Stop() {
	close(s.stop)
}

func (s *scaler) Run() {
	ticker := time.NewTicker(s.pollInterval)

	s.converge()
	for {
		select {
		case <-ticker.C:
			s.converge()
		case <-s.stop:
			ticker.Stop()
			return
		}
	}
}

func (s *scaler) converge() {
	descriptions, err := s.scaled.List()
	if err != nil {
		log.Errorf("Failed to check size of group: %s", err)
		return
	}

	log.Debugf("Found existing instances: %v", descriptions)

	grp := sync.WaitGroup{}

	actualSize := uint(len(descriptions))
	desiredSize := s.getSize()
	switch {
	case actualSize == desiredSize:
		log.Debugf("Group has %d instances, no action is needed", desiredSize)

	case actualSize > desiredSize:
		remove := actualSize - desiredSize
		log.Infof("Removing %d instances from group to reach desired %d", remove, desiredSize)

		sorted := make([]instance.Description, len(descriptions))
		copy(sorted, descriptions)

		// Sorting first ensures that redundant operations are non-destructive.
		sort.Sort(sortByID(sorted))

		// TODO(wfarner): Consider favoring removal of instances that do not match the desired configuration by
		// injecting a sorter.
		for _, toDestroy := range sorted[:remove] {
			grp.Add(1)
			destroy := toDestroy.ID
			go func() {
				defer grp.Done()
				s.scaled.Destroy(destroy)
			}()
		}

	case actualSize < desiredSize:
		add := desiredSize - actualSize
		log.Infof("Adding %d instances to group to reach desired %d", add, desiredSize)

		for i := 0; i < int(add); i++ {
			grp.Add(1)
			go func() {
				defer grp.Done()

				s.scaled.CreateOne(nil)
			}()
		}
	}

	// Wait for outstanding actions to finish.
	// It is not imperative to avoid stepping on another removal operation by this routine
	// (within this process or another) since the selection of removal candidates is stable.
	// However, we do so here to mitigate redundant work and avoidable benign (but confusing) errors
	// when overlaps happen.
	grp.Wait()
}
