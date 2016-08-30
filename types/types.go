package types

import (
	"fmt"
	"net/url"
	"time"
)

type SchedulerAdapter interface {
	Services() ([]*ServiceGroup, error)
	ListenForEvents(channel EventsChannel) error
}

type RegistryAdapter interface {
	Services() ([]*ServiceGroup, error)
	Ping() error
	Register(group *ServiceGroup) error
	Deregister(group *ServiceGroup) error
	AdvertiseAddr() (string, error)
}

// ServiceGroup represents the collection of services which expose multiple ports.
// Most of the time it will hold the single Service instance, but if the service exposes multiple ports, it will contain
// multiple services named by appending exposed port number to them, i.e. foo-service-3000, foo-service-4001, etc.
type ServiceGroup struct {
	ID           string
	IP           string
	Services     []*Service
	HealthChecks []*ServiceHealthCheck
}

// Service represents a single entry in the service registry.
type Service struct {
	ID           string
	Name         string
	Tags         []string
	Healthy      bool
	OriginalPort int
	ExposedPort  int
}

// ServiceHealthCheck represents health check definition.
type ServiceHealthCheck struct {
	ID           string
	Name         string
	Tags         []string
	Healthy      bool
	OriginalPort int
	ExposedPort  int
}

func (group *ServiceGroup) ServiceKey(service *Service) string {
	return fmt.Sprintf("%s:%s:%d", service.Name, group.IP, service.ExposedPort)
}

type ServiceAction int

const (
	// ServiceUnchanged denotes unchanged service status
	ServiceUnchanged ServiceAction = 1 << iota

	// ServiceWentUp denotes service availability
	ServiceWentUp

	// ServiceWentDown denotes service unavailability
	ServiceWentDown

	// ServiceStarted denotes added service instance
	ServiceStarted

	// ServiceStopped denotes removed service instance
	ServiceStopped
)

var serviceActionDescriptions = map[int]string{
	int(ServiceUnchanged): "unchanged",
	int(ServiceWentUp):    "went up",
	int(ServiceWentDown):  "went down",
	int(ServiceStarted):   "started",
	int(ServiceStopped):   "stopped",
}

func (action ServiceAction) String() string {
	return serviceActionDescriptions[int(action)]
}

// ServiceEvent is the definition for an event occurred to Service in scheduler.
type ServiceEvent struct {
	ServiceID     string
	IP            string
	Action        ServiceAction
	OriginalEvent interface{}
}

func (event *ServiceEvent) String() string {
	return fmt.Sprintf("%s — %s, %+v", event.ServiceID, event.Action, event.OriginalEvent)
}

// EventsChannel is a channel to receive events upon.
type EventsChannel chan *ServiceEvent

type Config struct {
	Marathon       string
	Consul         *url.URL
	DryRun         bool
	ResyncInterval time.Duration
}
