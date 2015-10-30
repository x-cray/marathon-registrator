package marathon

import (
	"net"
	"net/url"
	"strings"

	"github.com/x-cray/marathon-service-registrator/types"

	log "github.com/Sirupsen/logrus"
	marathon "github.com/x-cray/go-marathon"
)

var (
	startupTaskStatuses = map[string]bool{
		"TASK_RUNNING": true,
	}

	terminalTaskStatuses = map[string]bool{
		"TASK_FINISHED": true,
		"TASK_FAILED":   true,
		"TASK_KILLED":   true,
		"TASK_LOST":     true,
	}
)

type MarathonAdapter struct {
	client marathon.Marathon
}

func New(marathonURL string) (*MarathonAdapter, error) {
	config := marathon.NewDefaultConfig()
	config.URL = marathonURL
	config.RequestTimeout = 60 // 60 seconds
	config.EventsTransport = marathon.EventsTransportSSE

	log.WithField("prefix", "marathon").Infof("Connecting to Marathon at %v", marathonURL)
	client, err := marathon.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &MarathonAdapter{client: client}, nil
}

func (m *MarathonAdapter) ListenForEvents(channel types.EventsChannel) error {
	update := make(marathon.EventsChannel, 5)
	if err := m.client.AddEventsListener(update, marathon.EVENTS_APPLICATIONS|marathon.EVENT_FRAMEWORK_MESSAGE); err != nil {
		return err
	}

	// Convert Marathon events to abstract events and write to output channel.
	go func() {
		for {
			event := <-update
			channel <- toServiceEvent(event)
		}
	}()

	return nil
}

func toServiceEvent(marathonEvent *marathon.Event) (result *types.ServiceEvent) {
	// Instantiate result object.
	result = &types.ServiceEvent{
		OriginalEvent: marathonEvent.Event,
		Action:        types.ServiceUnchanged,
	}

	// Task status update event suggests that Marathon cached services list
	// should be updated:
	// - when ServiceStopped we need to remove service from cache
	// - when ServiceStarted we need to repopulate cache with fresh Marathon services
	statusUpdateEvent, ok := marathonEvent.Event.(*marathon.EventStatusUpdate)
	if ok {
		result.ServiceID = statusUpdateEvent.TaskID
		address, err := net.ResolveIPAddr("ip", statusUpdateEvent.Host)
		if err != nil {
			log.WithFields(log.Fields{
				"prefix":   "marathon",
				"hostname": statusUpdateEvent.Host,
				"err":      err,
			}).Warn("Unable to resolve address")
		} else {
			result.IP = address.IP.String()
		}
		if terminalTaskStatuses[statusUpdateEvent.TaskStatus] {
			result.Action = types.ServiceStopped
		} else if startupTaskStatuses[statusUpdateEvent.TaskStatus] {
			result.Action = types.ServiceStarted
		}
	}

	// Health status change event suggests that service should be
	// registered/unregistered in service registry.
	healthStatusChangeEvent, ok := marathonEvent.Event.(*marathon.EventHealthCheckChanged)
	if ok {
		result.ServiceID = healthStatusChangeEvent.TaskID
		if healthStatusChangeEvent.Alive {
			result.Action = types.ServiceWentUp
		} else {
			result.Action = types.ServiceWentDown
		}
	}

	return
}

func toService(task *marathon.Task, port int, app *marathon.Application) (*types.Service, error) {
	taskIP, err := net.ResolveIPAddr("ip", task.Host)
	if err != nil {
		return nil, err
	}

	idTokens := strings.Split(app.ID, "/")
	name := idTokens[len(idTokens)-1]

	return &types.Service{
		ID:   task.ID,
		Name: name,
		IP:   taskIP.String(),
		Port: port,
	}, nil
}

func (m *MarathonAdapter) Services() ([]*types.Service, error) {
	params := make(url.Values)
	params.Add("embed", "apps.tasks")
	applications, err := m.client.Applications(params)
	if err != nil {
		return nil, err
	}

	result := make([]*types.Service, 0)
	for _, app := range applications.Apps {
		for _, task := range app.Tasks {
			for _, port := range task.Ports {
				service, err := toService(task, port, &app)
				if err != nil {
					return nil, err
				}

				log.WithFields(log.Fields{
					"prefix": "marathon",
					"id":     service.ID,
					"name":   service.Name,
					"ip":     service.IP,
					"port":   service.Port,
				}).Debug("Service")
				result = append(result, service)
			}
		}
	}

	return result, nil
}
