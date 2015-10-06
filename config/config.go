package config

import (
	"time"
)

type Config struct {
	ResyncInterval time.Duration
	Marathon       string
	Consul         string
}
