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

func (b *Bridge) getCachedService(serviceID, actionText string) *types.Service {
	if service, ok := b.schedulerServices[serviceID]; ok {
		return service
	}

	log.Warningf("Service with id = %s was not found in scheduler cache. Could not %s.", serviceID, actionText)
	return nil
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
		// Only consider services registered on current registry advertized address.
		if event.IP == b.registryAdvertizeAddr {
			if service := b.getCachedService(event.ServiceID, "deregister"); service != nil {
				b.registry.Deregister(service)
				delete(b.schedulerServices, event.ServiceID)
			}
		}
	case types.ServiceWentUp:
		// Service went up, register it.
		if service := b.getCachedService(event.ServiceID, "register"); service != nil {
			b.registry.Register(service)
		}
	case types.ServiceWentDown:
		// Service went down, deregister it.
		if service := b.getCachedService(event.ServiceID, "deregister"); service != nil {
			b.registry.Deregister(service)
		}
	}

	return nil
}

func (b *Bridge) ProcessSchedulerEvents() error {
	marathonEvents := make(types.EventsChannel, 5)
	err := b.scheduler.ListenForEvents(marathonEvents)
	if err != nil {
		return err
	}

	log.WithField("prefix", "bridge").Info("Registered for scheduler event stream")
	for {
		event := <-marathonEvents
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

// Sync performs full synchronization of Marathon tasks to service registry.
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
	for _, marathonService := range schedulerServicesMap {
		// If service is not yet registered we need to register it.
		if registryServicesMap[marathonService.MapKey()] == nil {
			err := b.registry.Register(marathonService)
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
	var err error

	// Get registry's advertize address.
	b.registryAdvertizeAddr, err = b.registry.AdvertiseAddr()
	if err != nil {
		return nil, err
	}

	// Get services from scheduler.
	schedulerServicesArray, err := b.scheduler.Services()
	if err != nil {
		return nil, err
	}

	log.WithField("prefix", "bridge").Infof("Registry advertize address is %s", b.registryAdvertizeAddr)

	// Build 2 maps of services running on registry's advertized host:
	// ServiceID-indexed and ip:port-indexed
	ipPortServices := make(map[string]*types.Service)
	b.schedulerServices = make(map[string]*types.Service)
	for _, service := range schedulerServicesArray {
		if service.IP == b.registryAdvertizeAddr {
			ipPortServices[service.MapKey()] = service
			b.schedulerServices[service.ID] = service
		}
	}

	log.WithField("prefix", "bridge").Infof(
		"Received %d services from scheduler, %d are running on registry advertized address",
		len(schedulerServicesArray),
		len(ipPortServices),
	)

	return ipPortServices, nil
}
