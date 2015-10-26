package types

import (
	"fmt"
	"net/url"
	"time"

	"github.com/hashicorp/consul-template/logging"
)

type SchedulerAdapter interface {
	Services() ([]*Service, error)
	ListenForEvents()
}

type RegistryAdapter interface {
	Services() ([]*Service, error)
	Ping() error
	Register(service *Service) error
	Deregister(service *Service) error
	AdvertiseAddr() (string, error)
}

type Config struct {
	Marathon       string
	Consul         *url.URL
	DryRun         bool
	ResyncInterval time.Duration
	LogLevel       string
	Syslog         *logging.Config
}

type Service struct {
	ID    string
	Name  string
	Port  int
	IP    string
	Tags  []string
	Attrs map[string]string
	TTL   int
}

func (s *Service) MapKey() string {
	return fmt.Sprintf("%s:%d", s.IP, s.Port)
}
