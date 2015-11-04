package marathon

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/x-cray/marathon-service-registrator/types"

	log "github.com/Sirupsen/logrus"
	marathonClient "github.com/x-cray/go-marathon"
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
	client   MarathonClient
	resolver AddressResolver
}

func New(marathonURL string) (*MarathonAdapter, error) {
	config := marathonClient.NewDefaultConfig()
	config.URL = marathonURL
	config.RequestTimeout = 60 // 60 seconds
	config.EventsTransport = marathonClient.EventsTransportSSE

	log.WithField("prefix", "marathon").Infof("Connecting to Marathon at %v", marathonURL)
	client, err := marathonClient.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &MarathonAdapter{
		client:   client,
		resolver: &defaultAddressResolver{},
	}, nil
}

func (m *MarathonAdapter) ListenForEvents(channel types.EventsChannel) error {
	update := make(marathonClient.EventsChannel, 5)
	if err := m.client.AddEventsListener(update, marathonClient.EVENTS_APPLICATIONS|marathonClient.EVENT_FRAMEWORK_MESSAGE); err != nil {
		return err
	}

	// Convert Marathon events to abstract events and write to output channel.
	go func() {
		for {
			event := <-update
			channel <- m.toServiceEvent(event)
		}
	}()

	return nil
}

func mapDefault(m map[string]string, key, default_ string) string {
	v, ok := m[key]
	if !ok || v == "" {
		return default_
	}
	return v
}

func extractServiceMetadata(source, destination map[string]string, port string) {
	for k, v := range source {
		if !strings.HasPrefix(k, "SERVICE_") {
			continue
		}

		key := strings.ToLower(strings.TrimPrefix(k, "SERVICE_"))
		portkey := strings.SplitN(key, "_", 2)
		_, err := strconv.Atoi(portkey[0])
		if err == nil && len(portkey) > 1 {
			if portkey[0] != port {
				continue
			}
			destination[portkey[1]] = v
		} else {
			destination[key] = v
		}
	}
}

func serviceMetadata(application *marathonClient.Application, port int) map[string]string {
	result := make(map[string]string)
	stringPort := string(port)
	extractServiceMetadata(application.Env, result, stringPort)
	extractServiceMetadata(application.Labels, result, stringPort)
	return result
}

func (m *MarathonAdapter) toServiceEvent(marathonEvent *marathonClient.Event) (result *types.ServiceEvent) {
	// Instantiate result object.
	result = &types.ServiceEvent{
		OriginalEvent: marathonEvent.Event,
		Action:        types.ServiceUnchanged,
	}

	// Task status update event suggests that Marathon cached services list
	// should be updated:
	// - when ServiceStopped we need to remove service from cache
	// - when ServiceStarted we need to repopulate cache with fresh Marathon services
	statusUpdateEvent, ok := marathonEvent.Event.(*marathonClient.EventStatusUpdate)
	if ok {
		result.ServiceID = statusUpdateEvent.TaskID
		address, err := m.resolver.Resolve(statusUpdateEvent.Host)
		if err == nil {
			result.IP = address
		}

		if terminalTaskStatuses[statusUpdateEvent.TaskStatus] {
			result.Action = types.ServiceStopped
		} else if startupTaskStatuses[statusUpdateEvent.TaskStatus] {
			result.Action = types.ServiceStarted
		}
	}

	// Health status change event suggests that service should be
	// registered/unregistered in service registry.
	healthStatusChangeEvent, ok := marathonEvent.Event.(*marathonClient.EventHealthCheckChanged)
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

func (m *MarathonAdapter) toService(task *marathonClient.Task, port int, isgroup bool, app *marathonClient.Application) (*types.Service, error) {
	taskIP, err := m.resolver.Resolve(task.Host)
	if err != nil {
		return nil, err
	}

	idTokens := strings.Split(app.ID, "/")
	defaultName := idTokens[len(idTokens)-1]

	return &types.Service{
		ID:   task.ID,
		Name: defaultName,
		IP:   taskIP,
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
				service, err := m.toService(task, port, len(task.Ports) > 1, &app)
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
