package bridge

import (
	"sync"

	"github.com/x-cray/marathon-service-registrator/consul"
	"github.com/x-cray/marathon-service-registrator/marathon"
	"github.com/x-cray/marathon-service-registrator/types"

	log "github.com/Sirupsen/logrus"
	"fmt"
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

	marathonServices, err := b.marathon.Services()
	if err != nil {
		return err
	}

	log.Infof("Received %d services from Marathon", len(marathonServices))

	marathonServicesMap := make(map[string]map[int]*types.Service)
	for _, service := range marathonServices {
		entry, ok := marathonServicesMap[service.IP]
		if !ok {
			entry = make(map[int]*types.Service)
			marathonServicesMap[service.IP] = entry
		}

		entry[service.Port] = service
	}

	registryServices, err := b.registry.Services()
	if err != nil {
		return err
	}

	log.Infof("Received %d services from registry", len(registryServices))

	registryServicesMap := make(map[string]*types.Service)
	for _, service := range registryServices {
		registryServicesMap[fmt.Sprintf("%s:%d", service.IP, service.Port)] = service
	}

	return nil
}
