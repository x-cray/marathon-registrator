package consul

import (
	"log"
	"net/url"

	"github.com/x-cray/marathon-service-registrator/types"

	consulapi "github.com/hashicorp/consul/api"
)

const DefaultInterval = "10s"

func New(uri *url.URL, dryRun bool) (*ConsulAdapter, error) {
	config := consulapi.DefaultConfig()
	config.Address = uri.Host
	config.Scheme = uri.Scheme

	log.Printf("Connecting to Consul at %+v\n", uri)
	client, err := consulapi.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &ConsulAdapter{
		client: client,
		dryRun: dryRun,
	}, nil
}

type ConsulAdapter struct {
	client *consulapi.Client
	dryRun bool
}

// Ping will try to connect to consul by attempting to retrieve the current leader.
func (r *ConsulAdapter) Ping() error {
	status := r.client.Status()
	leader, err := status.Leader()
	if err != nil {
		return err
	}
	log.Println("consul: current leader ", leader)

	return nil
}

func (r *ConsulAdapter) Register(service *types.Service) error {
	if r.dryRun {
		log.Printf("Would register service %s at %s:%d", service.Name, service.IP, service.Port)
		return nil
	}

	registration := new(consulapi.AgentServiceRegistration)
	registration.ID = service.ID
	registration.Name = service.Name
	registration.Port = service.Port
	registration.Tags = service.Tags
	registration.Address = service.IP

	return r.client.Agent().ServiceRegister(registration)
}

func (r *ConsulAdapter) Deregister(service *types.Service) error {
	if r.dryRun {
		log.Printf("Would deregister service %s", service.ID)
		return nil
	}

	return r.client.Agent().ServiceDeregister(service.ID)
}

func (r *ConsulAdapter) Services() ([]*types.Service, error) {
	services, err := r.client.Agent().Services()
	if err != nil {
		return nil, err
	}
	out := make([]*types.Service, len(services))
	i := 0
	for _, v := range services {
		s := &types.Service{
			ID:   v.ID,
			Name: v.Service,
			Port: v.Port,
			Tags: v.Tags,
			IP:   v.Address,
		}
		out[i] = s
		i++

		log.Printf("Service %s: %s: port %d", v.Service, v.Address, v.Port)
	}

	return out, nil
}
