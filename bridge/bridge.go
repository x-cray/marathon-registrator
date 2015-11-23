package bridge

import (
	"sync"

	"github.com/x-cray/marathon-registrator/consul"
	"github.com/x-cray/marathon-registrator/marathon"
	"github.com/x-cray/marathon-registrator/types"

	log "github.com/Sirupsen/logrus"
)

type serviceGroupPair struct {
	service *types.Service
	group   *types.ServiceGroup
}

type Bridge struct {
	sync.Mutex

	scheduler              types.SchedulerAdapter
	schedulerServiceGroups map[string]*types.ServiceGroup
	registry               types.RegistryAdapter
	registryAdvertizeAddr  string
	config                 *types.Config
}

func New(c *types.Config) (*Bridge, error) {
	marathon, err := marathon.New(c.Marathon)
	if err != nil {
		return nil, err
	}

	consul, err := consul.New(c.Consul, c.DryRun)
	if err != nil {
		return nil, err
	}

	return &Bridge{
		config:    c,
		scheduler: marathon,
		registry:  consul,
	}, nil
}

func (b *Bridge) cachedServiceGroup(groupID, actionText string) *types.ServiceGroup {
	if group, ok := b.schedulerServiceGroups[groupID]; ok {
		return group
	}

	log.WithField("prefix", "bridge").Warningf(
		"Service group %s was not found in scheduler cache (has %d entries). Could not %s.",
		groupID,
		len(b.schedulerServiceGroups),
		actionText,
	)
	return nil
}

func logSkipMessage(ip string) {
	log.WithFields(log.Fields{
		"prefix": "bridge",
		"ip":     ip,
	}).Debug("Skipping event due to unrelated service host")
}

func (b *Bridge) processServiceEvent(event *types.ServiceEvent) error {
	b.Lock()
	defer b.Unlock()

	switch event.Action {
	case types.ServiceStarted:
		// New service is started, we need to refresh service cache.
		_, err := b.refreshSchedulerServices()
		if err != nil {
			return err
		}
	case types.ServiceStopped:
		// Service stopped, deregister and remove it from cache.
		// Only consider services registered on current registry's advertized address.
		if event.IP == b.registryAdvertizeAddr {
			if group := b.cachedServiceGroup(event.ServiceID, "deregister"); group != nil {
				b.registry.Deregister(group)
				delete(b.schedulerServiceGroups, event.ServiceID)
			}
		} else {
			logSkipMessage(event.IP)
		}
	case types.ServiceWentUp:
		// Service went up, register it.
		if group := b.cachedServiceGroup(event.ServiceID, "register"); group != nil {
			// Only consider services registered on current registry's advertized address.
			if group.IP == b.registryAdvertizeAddr {
				b.registry.Register(group)
			} else {
				logSkipMessage(group.IP)
			}
		}
	case types.ServiceWentDown:
		// Service went down, deregister it.
		if group := b.cachedServiceGroup(event.ServiceID, "deregister"); group != nil {
			// Only consider services registered on current registry's advertized address.
			if group.IP == b.registryAdvertizeAddr {
				b.registry.Deregister(group)
			} else {
				logSkipMessage(group.IP)
			}
		}
	}

	return nil
}

func (b *Bridge) ProcessSchedulerEvents() error {
	schedulerEvents := make(types.EventsChannel, 5)
	err := b.scheduler.ListenForEvents(schedulerEvents)
	if err != nil {
		return err
	}

	log.WithField("prefix", "bridge").Info("Registered for scheduler event stream")
	for {
		event, more := <-schedulerEvents
		if !more {
			break
		}

		if event.Action != types.ServiceUnchanged {
			log.WithFields(log.Fields{
				"prefix":  "bridge",
				"service": event.ServiceID,
				"action":  event.Action,
				"event":   event.OriginalEvent,
			}).Debug("Received scheduler event")
			b.processServiceEvent(event)
		}
	}

	return nil
}

// Sync performs full synchronization of scheduler tasks to service registry.
func (b *Bridge) Sync() error {
	b.Lock()
	defer b.Unlock()

	actionsPerformed := false

	// Get services from registry.
	registryServiceGroups, err := b.registry.Services()
	if err != nil {
		return err
	}

	log.WithField("prefix", "bridge").Infof("Received %d services from registry", len(registryServiceGroups))

	// Build ip:port-indexed service map.
	registryServicesMap := make(map[string]*serviceGroupPair)
	for _, group := range registryServiceGroups {
		for _, service := range group.Services {
			registryServicesMap[group.ServiceKey(service)] = &serviceGroupPair{
				service: service,
				group:   group,
			}
		}
	}

	schedulerServicesMap, err := b.refreshSchedulerServices()
	if err != nil {
		return err
	}

	// Register scheduler services absent from registry.
	for _, schedulerService := range schedulerServicesMap {
		group := schedulerService.group
		service := schedulerService.service

		// If service is not yet registered we need to register it.
		// Only consider healthy services registered on current registry's advertized address.
		if group.IP == b.registryAdvertizeAddr && service.Healthy && registryServicesMap[group.ServiceKey(service)] == nil {
			err := b.registry.Register(group)
			if err != nil {
				return err
			}
			actionsPerformed = true
		}
	}

	// Deregister dangling services (existing in registry but absent from scheduler).
	for _, registryService := range registryServicesMap {
		group := registryService.group
		service := registryService.service

		// If service is registered and we don't have it in scheduler we need to deregister it.
		if schedulerServicesMap[group.ServiceKey(service)] == nil {
			err := b.registry.Deregister(group)
			if err != nil {
				return err
			}
			actionsPerformed = true
		}
	}

	if !actionsPerformed {
		log.WithField("prefix", "bridge").Info("All services are in sync, no actions performed")
	}

	return nil
}

func (b *Bridge) refreshSchedulerServices() (map[string]*serviceGroupPair, error) {
	log.WithField("prefix", "bridge").Info("Refreshing scheduler services")

	// Get registry's advertize address.
	addr, err := b.registry.AdvertiseAddr()
	if err != nil {
		return nil, err
	}

	b.registryAdvertizeAddr = addr

	// Get services from scheduler.
	schedulerServiceGroups, err := b.scheduler.Services()
	if err != nil {
		return nil, err
	}

	log.WithField("prefix", "bridge").Infof("Registry advertize address is %s", b.registryAdvertizeAddr)

	// Build 2 maps of services:
	// ServiceID-indexed and ip:port-indexed
	ipPortServices := make(map[string]*serviceGroupPair)
	b.schedulerServiceGroups = make(map[string]*types.ServiceGroup)
	for _, group := range schedulerServiceGroups {
		b.schedulerServiceGroups[group.ID] = group
		for _, service := range group.Services {
			ipPortServices[group.ServiceKey(service)] = &serviceGroupPair{
				service: service,
				group:   group,
			}
		}
	}

	log.WithField("prefix", "bridge").Infof(
		"Received %d services from scheduler",
		len(schedulerServiceGroups),
	)

	return ipPortServices, nil
}
