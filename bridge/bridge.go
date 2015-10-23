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

	marathon *marathon.MarathonAdapter
	registry *consul.ConsulAdapter
	services map[string][]*types.Service
	config   *types.Config
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
		config:   c,
		marathon: marathon,
		registry: consul,
		services: make(map[string][]*types.Service),
	}, nil
}

func (b *Bridge) ListenForEvents() {
	b.marathon.ListenForEvents()
}

// Perform full synchronization of Marathon tasks to service registry.
func (b *Bridge) Sync() error {
	b.Lock()
	defer b.Unlock()

	// Get services from registry and build ip:port-indexed service map.
	registryServices, err := b.registry.Services()
	if err != nil {
		return err
	}

	log.WithField("prefix", "bridge").Infof("Received %d services from registry", len(registryServices))

	registryServicesMap := make(map[string]*types.Service)
	for _, service := range registryServices {
		registryServicesMap[service.MapKey()] = service
	}

	// Get services from Marathon and build ip-indexed service map.
	// Determine not yet registered services (existing in Marathon and absent in registry).
	marathonServices, err := b.marathon.Services()
	if err != nil {
		return err
	}

	log.WithField("prefix", "bridge").Infof("Received %d services from Marathon", len(marathonServices))

	marathonServicesMap := make(map[string]map[int]*types.Service)
	for _, service := range marathonServices {
		entry, ok := marathonServicesMap[service.IP]
		if !ok {
			entry = make(map[int]*types.Service)
			marathonServicesMap[service.IP] = entry
		}

		entry[service.Port] = service

		// If service is not yet registered we need to register it.
		if registryServicesMap[service.MapKey()] == nil {
			b.registry.Register(service)
		}
	}

	// Deregister dangling services (existing in registry and absent in Marathon).
	for _, registryService := range registryServices {
		// If service is registered and we don't have it in Marathon we need to deregister it.
		if marathonServicesMap[registryService.IP] == nil || marathonServicesMap[registryService.IP][registryService.Port] == nil {
			b.registry.Deregister(registryService)
		}
	}

	return nil
}
