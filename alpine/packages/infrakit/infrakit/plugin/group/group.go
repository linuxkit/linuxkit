package group

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/plugin/group/types"
	"github.com/docker/infrakit/spi/flavor"
	"github.com/docker/infrakit/spi/group"
	"github.com/docker/infrakit/spi/instance"
	"sync"
	"time"
)

const (
	groupTag  = "infrakit.group"
	configTag = "infrakit.config_sha"
)

// InstancePluginLookup helps with looking up an instance plugin by name
type InstancePluginLookup func(string) (instance.Plugin, error)

// FlavorPluginLookup helps with looking up a flavor plugin by name
type FlavorPluginLookup func(string) (flavor.Plugin, error)

// NewGroupPlugin creates a new group plugin.
func NewGroupPlugin(
	instancePlugins InstancePluginLookup,
	flavorPlugins FlavorPluginLookup,
	pollInterval time.Duration) group.Plugin {

	return &plugin{
		instancePlugins: instancePlugins,
		flavorPlugins:   flavorPlugins,
		pollInterval:    pollInterval,
		groups:          groups{byID: map[group.ID]*groupContext{}},
	}
}

type plugin struct {
	instancePlugins InstancePluginLookup
	flavorPlugins   FlavorPluginLookup
	pollInterval    time.Duration
	lock            sync.Mutex
	groups          groups
}

func (p *plugin) validate(config group.Spec) (groupSettings, error) {

	noSettings := groupSettings{}

	if config.ID == "" {
		return noSettings, errors.New("Group ID must not be blank")
	}

	parsed, err := types.ParseProperties(config)
	if err != nil {
		return noSettings, err
	}

	if parsed.Allocation.Size == 0 &&
		(parsed.Allocation.LogicalIDs == nil || len(parsed.Allocation.LogicalIDs) == 0) {

		return noSettings, errors.New("Allocation must not be blank")
	}

	if parsed.Allocation.Size > 0 && parsed.Allocation.LogicalIDs != nil && len(parsed.Allocation.LogicalIDs) > 0 {

		return noSettings, errors.New("Only one Allocation method may be used")
	}

	flavorPlugin, err := p.flavorPlugins(parsed.Flavor.Plugin)
	if err != nil {
		return noSettings, fmt.Errorf("Failed to find Flavor plugin '%s':%v", parsed.Flavor.Plugin, err)
	}

	if err := flavorPlugin.Validate(types.RawMessage(parsed.Flavor.Properties), parsed.Allocation); err != nil {
		return noSettings, err
	}

	instancePlugin, err := p.instancePlugins(parsed.Instance.Plugin)
	if err != nil {
		return noSettings, fmt.Errorf("Failed to find Instance plugin '%s':%v", parsed.Instance.Plugin, err)
	}

	if err := instancePlugin.Validate(types.RawMessage(parsed.Instance.Properties)); err != nil {
		return noSettings, err
	}

	return groupSettings{
		instancePlugin: instancePlugin,
		flavorPlugin:   flavorPlugin,
		config:         parsed,
	}, nil
}

func (p *plugin) WatchGroup(config group.Spec) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	settings, err := p.validate(config)
	if err != nil {
		return err
	}

	// Two sets of instance tags are used - one for defining membership within the group, and another used to tag
	// newly-created instances.  This allows the scaler to collect and report members of a group which have
	// membership tags but different generation-specific tags.  In practice, we use this the additional tags to
	// attach a config SHA to instances for config change detection.
	scaled := &scaledGroup{
		settings:   settings,
		memberTags: map[string]string{groupTag: string(config.ID)},
	}
	scaled.changeSettings(settings)

	var supervisor Supervisor
	if settings.config.Allocation.Size != 0 {
		supervisor = NewScalingGroup(scaled, settings.config.Allocation.Size, p.pollInterval)
	} else if len(settings.config.Allocation.LogicalIDs) > 0 {
		supervisor = NewQuorum(scaled, settings.config.Allocation.LogicalIDs, p.pollInterval)
	} else {
		panic("Invalid empty allocation method")
	}

	if _, exists := p.groups.get(config.ID); exists {
		return fmt.Errorf("Already watching group '%s'", config.ID)
	}

	p.groups.put(config.ID, &groupContext{supervisor: supervisor, scaled: scaled, settings: settings})

	go supervisor.Run()
	log.Infof("Watching group '%v'", config.ID)

	return nil
}

func (p *plugin) UnwatchGroup(id group.ID) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	grp, exists := p.groups.get(id)
	if !exists {
		return fmt.Errorf("Group '%s' is not being watched", id)
	}

	grp.supervisor.Stop()

	p.groups.del(id)
	log.Infof("Stopped watching group '%s'", id)
	return nil
}

func (p *plugin) InspectGroup(id group.ID) (group.Description, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	context, exists := p.groups.get(id)
	if !exists {
		return group.Description{}, fmt.Errorf("Group '%s' is not being watched", id)
	}

	instances, err := context.scaled.List()
	if err != nil {
		return group.Description{}, err
	}

	return group.Description{Instances: instances}, nil
}

type updatePlan interface {
	Explain() string
	Run(pollInterval time.Duration) error
	Stop()
}

type noopUpdate struct {
}

func (n noopUpdate) Explain() string {
	return "Noop"
}

func (n noopUpdate) Run(_ time.Duration) error {
	return nil
}

func (n noopUpdate) Stop() {
}

func (p *plugin) planUpdate(id group.ID, updatedSettings groupSettings) (updatePlan, error) {

	context, exists := p.groups.get(id)
	if !exists {
		return nil, fmt.Errorf("Group '%s' is not being watched", id)
	}

	return context.supervisor.PlanUpdate(context.scaled, context.settings, updatedSettings)
}

func (p *plugin) DescribeUpdate(updated group.Spec) (string, error) {
	updatedSettings, err := p.validate(updated)
	if err != nil {
		return "", err
	}

	plan, err := p.planUpdate(updated.ID, updatedSettings)
	if err != nil {
		return "", err
	}

	return plan.Explain(), nil
}

func (p *plugin) initiateUpdate(id group.ID, updatedSettings groupSettings) (*groupContext, updatePlan, error) {
	p.lock.Lock()
	defer p.lock.Unlock()

	plan, err := p.planUpdate(id, updatedSettings)
	if err != nil {
		return nil, nil, err
	}

	grp, _ := p.groups.get(id)
	if grp.getUpdate() != nil {
		return nil, nil, errors.New("Update already in progress for this group")
	}

	grp.setUpdate(plan)
	grp.changeSettings(updatedSettings)
	log.Infof("Executing update plan for '%s': %s", id, plan.Explain())
	return grp, plan, nil
}

func (p *plugin) UpdateGroup(updated group.Spec) error {
	updatedSettings, err := p.validate(updated)
	if err != nil {
		return err
	}

	grp, plan, err := p.initiateUpdate(updated.ID, updatedSettings)
	if err != nil {
		return err
	}

	err = plan.Run(p.pollInterval)
	grp.setUpdate(nil)
	log.Infof("Finished updating group %s", updated.ID)
	return err
}

func (p *plugin) StopUpdate(gid group.ID) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	grp, exists := p.groups.get(gid)
	if !exists {
		return fmt.Errorf("Group '%s' is not being watched", gid)
	}
	update := grp.getUpdate()
	if update == nil {
		return fmt.Errorf("Group '%s' is not being updated", gid)
	}

	log.Infof("Stopping update for group %s", gid)
	grp.setUpdate(nil)
	update.Stop()

	return nil
}

func (p *plugin) DestroyGroup(gid group.ID) error {
	p.lock.Lock()

	context, exists := p.groups.get(gid)
	if !exists {
		p.lock.Unlock()
		return fmt.Errorf("Group '%s' is not being watched", gid)
	}

	// The lock is released before performing blocking operations.
	p.groups.del(gid)
	p.lock.Unlock()

	context.supervisor.Stop()
	descriptions, err := context.scaled.List()
	if err != nil {
		return err
	}

	for _, desc := range descriptions {
		context.scaled.Destroy(desc.ID)
	}

	return nil
}
