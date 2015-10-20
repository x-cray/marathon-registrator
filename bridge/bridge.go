package bridge

import (
	"log"
	"sync"

	"github.com/x-cray/marathon-service-registrator/consul"
	"github.com/x-cray/marathon-service-registrator/marathon"
	"github.com/x-cray/marathon-service-registrator/types"
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

	log.Printf("Received %d services from Marathon", len(marathonServices))

	registryServices, err := b.registry.Services()
	if err != nil {
		return err
	}

	log.Printf("Received %d services from registry", len(registryServices))

//	tasksMap := make(map[string]*types.Service)
//	servicesMap := make(map[string]*types.Service)

	return nil
}
