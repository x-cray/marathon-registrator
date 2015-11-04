package bridge

import (
	"sync"

	"github.com/x-cray/marathon-service-registrator/consul"
	"github.com/x-cray/marathon-service-registrator/marathon"
	"github.com/x-cray/marathon-service-registrator/types"

	log "github.com/Sirupsen/logrus"
)

type Bridge struct {
	sync.Mutex

	scheduler             types.SchedulerAdapter
	schedulerServices     map[string]*types.Service
	registry              types.RegistryAdapter
	registryAdvertizeAddr string
	config                *types.Config
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

func (b *Bridge) cachedService(serviceID, actionText string) *types.Service {
	if service, ok := b.schedulerServices[serviceID]; ok {
		return service
	}

	log.WithField("prefix", "bridge").Warningf("Service %s was not found in scheduler cache (has %d entries). Could not %s.", serviceID, len(b.schedulerServices), actionText)
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
			if service := b.cachedService(event.ServiceID, "deregister"); service != nil {
				b.registry.Deregister(service)
				delete(b.schedulerServices, event.ServiceID)
			}
		} else {
			logSkipMessage(event.IP)
		}
	case types.ServiceWentUp:
		// Service went up, register it.
		if service := b.cachedService(event.ServiceID, "register"); service != nil {
			// Only consider services registered on current registry's advertized address.
			if service.IP == b.registryAdvertizeAddr {
				b.registry.Register(service)
			} else {
				logSkipMessage(service.IP)
			}
		}
	case types.ServiceWentDown:
		// Service went down, deregister it.
		if service := b.cachedService(event.ServiceID, "deregister"); service != nil {
			// Only consider services registered on current registry's advertized address.
			if service.IP == b.registryAdvertizeAddr {
				b.registry.Deregister(service)
			} else {
				logSkipMessage(service.IP)
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
		event := <-schedulerEvents
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
	registryServices, err := b.registry.Services()
	if err != nil {
		return err
	}

	log.WithField("prefix", "bridge").Infof("Received %d services from registry", len(registryServices))

	// Build ip:port-indexed service map.
	registryServicesMap := make(map[string]*types.Service)
	for _, service := range registryServices {
		registryServicesMap[service.MapKey()] = service
	}

	schedulerServicesMap, err := b.refreshSchedulerServices()
	if err != nil {
		return err
	}

	// Register scheduler services absent in registry.
	for _, schedulerService := range schedulerServicesMap {
		// If service is not yet registered we need to register it.
		// Only consider services registered on current registry's advertized address.
		if schedulerService.IP == b.registryAdvertizeAddr && registryServicesMap[schedulerService.MapKey()] == nil {
			err := b.registry.Register(schedulerService)
			if err != nil {
				return err
			}
			actionsPerformed = true
		}
	}

	// Deregister dangling services (existing in registry and absent in scheduler).
	for _, registryService := range registryServicesMap {
		// If service is registered and we don't have it in scheduler we need to deregister it.
		if schedulerServicesMap[registryService.MapKey()] == nil {
			err := b.registry.Deregister(registryService)
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

func (b *Bridge) refreshSchedulerServices() (map[string]*types.Service, error) {
	log.WithField("prefix", "bridge").Info("Refreshing scheduler services")

	// Get registry's advertize address.
	addr, err := b.registry.AdvertiseAddr()
	if err != nil {
		return nil, err
	}

	b.registryAdvertizeAddr = addr

	// Get services from scheduler.
	schedulerServicesArray, err := b.scheduler.Services()
	if err != nil {
		return nil, err
	}

	log.WithField("prefix", "bridge").Infof("Registry advertize address is %s", b.registryAdvertizeAddr)

	// Build 2 maps of services:
	// ServiceID-indexed and ip:port-indexed
	ipPortServices := make(map[string]*types.Service)
	b.schedulerServices = make(map[string]*types.Service)
	for _, service := range schedulerServicesArray {
		ipPortServices[service.MapKey()] = service
		b.schedulerServices[service.ID] = service
	}

	log.WithField("prefix", "bridge").Infof(
		"Received %d services from scheduler",
		len(schedulerServicesArray),
	)

	return ipPortServices, nil
}
