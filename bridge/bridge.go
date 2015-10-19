package bridge

import (
	"log"
	"net"
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

	marathonApplications, err := b.marathon.Applications()
	if err != nil {
		return err
	}

	consulServices, err := b.registry.Services()
	if err != nil {
		return err
	}

	for _, app := range marathonApplications.Apps {
		log.Printf("App %v\n", app.ID)
		for _, task := range app.Tasks {
			taskIP, err := net.ResolveIPAddr("ip", task.Host)
			if err != nil {
				return err
			}
			log.Printf("- task at %s, ports: %v\n", taskIP, task.Ports)
		}
	}

	for _, service := range consulServices {
		log.Printf("Service %s: %s: port %d\n", service.ID, service.IP, service.Port)
	}

	return nil
}
