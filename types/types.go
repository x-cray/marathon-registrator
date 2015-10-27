package types

import (
	"fmt"
	"net/url"
	"time"
)

type SchedulerAdapter interface {
	Services() ([]*Service, error)
	ListenForEvents() (EventsChannel, error)
}

type RegistryAdapter interface {
	Services() ([]*Service, error)
	Ping() error
	Register(service *Service) error
	Deregister(service *Service) error
	AdvertiseAddr() (string, error)
}

// Event is the definition for a event in scheduler.
type Event struct {
	ID    int
	Name  string
	Event interface{}
}

func (event *Event) String() string {
	return fmt.Sprintf("type: %s, event: %s", event.Name, event.Event)
}

// EventsChannel is a channel to receive events upon.
type EventsChannel chan *Event

type Config struct {
	Marathon       string
	Consul         *url.URL
	DryRun         bool
	ResyncInterval time.Duration
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
