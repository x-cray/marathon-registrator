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
}
