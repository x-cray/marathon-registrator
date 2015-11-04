package consul

import (
	"errors"
	"net/url"

	"github.com/x-cray/marathon-service-registrator/types"

	log "github.com/Sirupsen/logrus"
	consulapi "github.com/hashicorp/consul/api"
)

func New(uri *url.URL, dryRun bool) (*ConsulAdapter, error) {
	config := consulapi.DefaultConfig()
	config.Address = uri.Host
	config.Scheme = uri.Scheme

	log.WithField("prefix", "consul").Infof("Connecting to Consul at %v", uri)
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
	log.WithField("prefix", "consul").Debugf("Current leader %s", leader)

	return nil
}

func (r *ConsulAdapter) Register(service *types.Service) error {
	if r.dryRun {
		log.WithFields(log.Fields{
			"prefix": "consul",
			"name":   service.Name,
			"ip":     service.IP,
			"port":   service.Port,
		}).Info("[dry-run] Would register service")
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
		log.WithFields(log.Fields{
			"prefix": "consul",
			"id":     service.ID,
			"name":   service.Name,
			"ip":     service.IP,
			"port":   service.Port,
		}).Info("[dry-run] Would deregister service")
		return nil
	}

	return r.client.Agent().ServiceDeregister(service.ID)
}

func (r *ConsulAdapter) AdvertiseAddr() (string, error) {
	info, err := r.client.Agent().Self()
	if err != nil {
		return "", err
	}

	config := info["Config"]
	if config != nil {
		addr, ok := config["AdvertiseAddr"].(string)
		if ok {
			return addr, nil
		}
	}

	return "", errors.New("Advertized address was not found")
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

		log.WithFields(log.Fields{
			"prefix": "consul",
			"id":     v.ID,
			"name":   v.Service,
			"ip":     v.Address,
			"port":   v.Port,
		}).Debugf("Service")
	}

	return out, nil
}
