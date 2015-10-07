package config

import (
	"time"

	"github.com/hashicorp/consul-template/logging"
)

type Config struct {
	Marathon       string
	Consul         string
	ResyncInterval time.Duration
	LogLevel       string
	Syslog         *logging.Config
}
