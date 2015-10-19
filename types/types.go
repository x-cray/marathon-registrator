package types

import (
	"net/url"
	"time"

	"github.com/hashicorp/consul-template/logging"
)

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

	Origin ServicePort
}

type ServicePort struct {
	HostPort          string
	HostIP            string
	ExposedPort       string
	ExposedIP         string
	PortType          string
	ContainerHostname string
	ContainerID       string
}
