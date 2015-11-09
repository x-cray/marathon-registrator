package consul

import (
	"errors"
	"net/url"
	"strings"

	"github.com/x-cray/marathon-service-registrator/types"

	log "github.com/Sirupsen/logrus"
	consulAPI "github.com/hashicorp/consul/api"
)

type consulAdapter struct {
	client *consulAPI.Client
	dryRun bool
}

func New(uri *url.URL, dryRun bool) (*consulAdapter, error) {
	config := consulAPI.DefaultConfig()
	config.Address = uri.Host
	config.Scheme = uri.Scheme

	log.WithField("prefix", "consul").Infof("Connecting to Consul at %v", uri)
	client, err := consulAPI.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &consulAdapter{
		client: client,
		dryRun: dryRun,
	}, nil
}

// Ping will try to connect to consul by attempting to retrieve the current leader.
func (r *consulAdapter) Ping() error {
	status := r.client.Status()
	leader, err := status.Leader()
	if err != nil {
		return err
	}
	log.WithField("prefix", "consul").Debugf("Current leader %s", leader)

	return nil
}

func (r *consulAdapter) Register(group *types.ServiceGroup) error {
	for _, service := range group.Services {
		if r.dryRun {
			log.WithFields(log.Fields{
				"prefix": "consul",
				"ip":     group.IP,
				"name":   service.Name,
				"port":   service.ExposedPort,
			}).Info("[dry-run] Would register service")
			continue
		}

		log.WithFields(log.Fields{
			"prefix": "consul",
			"ip":     group.IP,
			"name":   service.Name,
			"port":   service.ExposedPort,
		}).Info("Registering service")

		registration := new(consulAPI.AgentServiceRegistration)
		registration.Address = group.IP
		registration.ID = service.ID
		registration.Name = service.Name
		registration.Tags = service.Tags
		registration.Port = service.ExposedPort

		err := r.client.Agent().ServiceRegister(registration)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *consulAdapter) Deregister(group *types.ServiceGroup) error {
	for _, service := range group.Services {
		if r.dryRun {
			log.WithFields(log.Fields{
				"prefix": "consul",
				"ip":     group.IP,
				"id":     service.ID,
				"name":   service.Name,
				"port":   service.ExposedPort,
			}).Info("[dry-run] Would deregister service")
			continue
		}

		log.WithFields(log.Fields{
			"prefix": "consul",
			"ip":     group.IP,
			"id":     service.ID,
			"name":   service.Name,
			"port":   service.ExposedPort,
		}).Info("Deregistering service")

		err := r.client.Agent().ServiceDeregister(service.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *consulAdapter) AdvertiseAddr() (string, error) {
	info, err := r.client.Agent().Self()
	if err != nil {
		return "", err
	}

	config := info["Config"]
	if config != nil {
		address, ok := config["AdvertiseAddr"].(string)
		if ok {
			return address, nil
		}
	}

	return "", errors.New("Advertized address was not found")
}

func groupID(serviceID string) string {
	i := strings.LastIndex(serviceID, ":")
	if i > 0 {
		return serviceID[:i]
	}

	return serviceID
}

func (r *consulAdapter) Services() ([]*types.ServiceGroup, error) {
	services, err := r.client.Agent().Services()
	if err != nil {
		return nil, err
	}

	out := make([]*types.ServiceGroup, len(services))
	i := 0
	for _, v := range services {
		group := &types.ServiceGroup{
			ID:   groupID(v.ID),
			IP:   v.Address,
			Services: []*types.Service{
				&types.Service{
					ID:          v.ID,
					Name:        v.Service,
					Tags:        v.Tags,
					ExposedPort: v.Port,
				},
			},
		}
		out[i] = group
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
