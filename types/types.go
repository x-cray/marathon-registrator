package types

import (
	"fmt"
	"net/url"
	"time"
)

type SchedulerAdapter interface {
	Services() ([]*Service, error)
	ListenForEvents(channel EventsChannel) error
}

type RegistryAdapter interface {
	Services() ([]*Service, error)
	Ping() error
	Register(service *Service) error
	Deregister(service *Service) error
	AdvertiseAddr() (string, error)
}

type Service struct {
	ID   string
	Name string
	Port int
	IP   string
	Tags []string
}

func (s *Service) String() string {
	return fmt.Sprintf("service: %s, id: %s at %s:%d", s.Name, s.ID, s.IP, s.Port)
}

func (s *Service) MapKey() string {
	return fmt.Sprintf("%s:%d", s.IP, s.Port)
}

type ServiceAction int

const (
	// ServiceWentUp denotes service availability
	ServiceWentUp ServiceAction = 1 << iota

	// ServiceWentDown denotes service unavailability
	ServiceWentDown
)

const serviceActionMap = map[ServiceAction]string{
	ServiceWentUp:   "went up",
	ServiceWentDown: "went down",
}

func (a ServiceAction) String() string {
	return serviceActionMap[a]
}

// ServiceEvent is the definition for an event occurred to Service in scheduler.
type ServiceEvent struct {
	Service       *Service
	Action        ServiceAction
	OriginalEvent interface{}
}

func (event *ServiceEvent) String() string {
	return fmt.Sprintf("%s — %s", event.Service, event.Action)
}

// EventsChannel is a channel to receive events upon.
type EventsChannel chan *ServiceEvent

type Config struct {
	Marathon       string
	Consul         *url.URL
	DryRun         bool
	ResyncInterval time.Duration
}
