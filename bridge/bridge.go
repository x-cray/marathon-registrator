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

	scheduler types.SchedulerAdapter
	registry  types.RegistryAdapter
	config    *types.Config
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

func (b *Bridge) ListenForEvents() error {
	_, err := b.scheduler.ListenForEvents()
	if err != nil {
		return err
	}

	log.WithField("prefix", "bridge").Infof("Registered for Marathon event stream")
	return nil
}

// Perform full synchronization of Marathon tasks to service registry.
func (b *Bridge) Sync() error {
	b.Lock()
	defer b.Unlock()

	actionsPerformed := false

	// Get services from registry.
	registryServices, err := b.registry.Services()
	if err != nil {
		return err
	}

	// Get registry's advertize address.
	advertizeAddr, err := b.registry.AdvertiseAddr()
	if err != nil {
		return err
	}

	log.WithField("prefix", "bridge").Infof("Registry advertize address is %s", advertizeAddr)
	log.WithField("prefix", "bridge").Infof("Received %d services from registry", len(registryServices))

	// Build ip:port-indexed service map.
	registryServicesMap := make(map[string]*types.Service)
	for _, service := range registryServices {
		registryServicesMap[service.MapKey()] = service
	}

	// Get services from Marathon.
	marathonServices, err := b.scheduler.Services()
	if err != nil {
		return err
	}

	// Build service map of services running on registry advertized host.
	marathonServicesMap := make(map[string]*types.Service)
	for _, service := range marathonServices {
		if service.IP == advertizeAddr {
			marathonServicesMap[service.MapKey()] = service
		}
	}

	log.WithField("prefix", "bridge").Infof(
		"Received %d services from Marathon, %d are running on registry advertized address",
		len(marathonServices),
		len(marathonServicesMap),
	)

	// Register Marathon services absent in registry.
	for _, marathonService := range marathonServicesMap {
		// If service is not yet registered we need to register it.
		if registryServicesMap[marathonService.MapKey()] == nil {
			err := b.registry.Register(marathonService)
			if err != nil {
				return err
			}
			actionsPerformed = true
		}
	}

	// Deregister dangling services (existing in registry and absent in Marathon).
	for _, registryService := range registryServicesMap {
		// If service is registered and we don't have it in Marathon we need to deregister it.
		if marathonServicesMap[registryService.MapKey()] == nil {
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
