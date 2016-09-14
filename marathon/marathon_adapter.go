package marathon

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/x-cray/marathon-registrator/types"

	log "github.com/Sirupsen/logrus"
	marathonClient "github.com/gambol99/go-marathon"
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

// Adapter is the implementation of RegistryAdapter for Marathon.
type Adapter struct {
	client   Client
	resolver AddressResolver
}

// New creates a new Adapter.
func New(marathonURL string) (*Adapter, error) {
	config := marathonClient.NewDefaultConfig()
	config.URL = marathonURL
	config.EventsTransport = marathonClient.EventsTransportSSE

	log.WithField("prefix", "marathon").Infof("Connecting to Marathon at %v", marathonURL)
	client, err := marathonClient.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &Adapter{
		client:   client,
		resolver: &defaultAddressResolver{},
	}, nil
}

// ListenForEvents subscribes to Marathon events and publishes them to channel.
func (m *Adapter) ListenForEvents(channel types.EventsChannel) error {
	update := make(marathonClient.EventsChannel, 5)
	eventTypes := marathonClient.EventIDApplications | marathonClient.EventIDFrameworkMessage
	if err := m.client.AddEventsListener(update, eventTypes); err != nil {
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

func (m *Adapter) toServiceHealthCheck(marathonHealthCheck *marathonClient.HealthCheck) (result *types.ServiceHealthCheck) {
	result = &types.ServiceHealthCheck{}

	return
}

func (m *Adapter) toServiceEvent(marathonEvent *marathonClient.Event) (result *types.ServiceEvent) {
	// Instantiate result object.
	result = &types.ServiceEvent{
		OriginalEvent: marathonEvent.Event,
		Action:        types.ServiceUnchanged,
	}

	// Task status update event suggests that Marathon cached services list
	// should be updated:
	// - when ServiceStopped we need to remove service from cache and deregister it
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

func mapDefault(dict map[string]string, key, defaultValue string) string {
	v, ok := dict[key]
	if !ok || v == "" {
		return defaultValue
	}
	return v
}

func extractServiceMetadata(source, destination map[string]string, port string) {
	for k, v := range source {
		if !strings.HasPrefix(k, "SERVICE_") {
			continue
		}

		key := strings.ToLower(strings.TrimPrefix(k, "SERVICE_"))
		portKey := strings.SplitN(key, "_", 2)
		_, err := strconv.Atoi(portKey[0])
		if err == nil && len(portKey) > 1 {
			if portKey[0] != port {
				continue
			}
			destination[portKey[1]] = v
		} else {
			destination[key] = v
		}
	}
}

func serviceMetadata(application *marathonClient.Application, port int) map[string]string {
	result := make(map[string]string)
	stringPort := strconv.Itoa(port)
	extractServiceMetadata(*application.Env, result, stringPort)
	extractServiceMetadata(*application.Labels, result, stringPort)
	return result
}

func parseTags(tagString string) []string {
	var tags []string
	if tagString != "" {
		tags = append(tags, strings.Split(tagString, ",")...)
	}
	return tags
}

func originalPorts(app *marathonClient.Application) []int {
	if app.Container != nil && app.Container.Docker != nil {
		var res []int
		for _, portMapping := range *app.Container.Docker.PortMappings {
			res = append(res, portMapping.ContainerPort)
		}
		return res
	}

	return app.Ports
}

func isHealthy(task *marathonClient.Task, app *marathonClient.Application) bool {
	// Tasks' health has not yet been checked.
	if len(*app.HealthChecks) != len(task.HealthCheckResults) {
		return false
	}

	// Tasks' health is not ok.
	for _, checkResult := range task.HealthCheckResults {
		if !checkResult.Alive {
			return false
		}
	}

	return true
}

func (m *Adapter) toServiceGroup(task *marathonClient.Task, app *marathonClient.Application) (*types.ServiceGroup, error) {
	taskIP, err := m.resolver.Resolve(task.Host)
	if err != nil {
		return nil, err
	}

	originalPorts := originalPorts(app)
	if len(task.Ports) != len(originalPorts) {
		return nil, errors.New("Task original and exposed ports count mismatch")
	}

	idTokens := strings.Split(app.ID, "/")
	defaultName := idTokens[len(idTokens)-1]
	isGroup := len(task.Ports) > 1
	services := make([]*types.Service, len(task.Ports))
	serviceGroup := &types.ServiceGroup{
		ID:       task.ID,
		IP:       taskIP,
		Services: services,
	}

	for i, exposedPort := range task.Ports {
		originalPort := originalPorts[i]
		name := defaultName
		if isGroup {
			name += fmt.Sprintf("-%d", originalPort)
		}
		metadata := serviceMetadata(app, originalPort)
		service := &types.Service{
			ID:           fmt.Sprintf("%s:%d", serviceGroup.ID, originalPort),
			Name:         mapDefault(metadata, "name", name),
			Tags:         parseTags(mapDefault(metadata, "tags", "")),
			Healthy:      isHealthy(task, app),
			OriginalPort: originalPort,
			ExposedPort:  exposedPort,
		}
		services[i] = service
	}

	return serviceGroup, nil
}

// Services returns the list of registered services.
func (m *Adapter) Services() ([]*types.ServiceGroup, error) {
	params := make(url.Values)
	params.Add("embed", "apps.tasks")
	applications, err := m.client.Applications(params)
	if err != nil {
		return nil, err
	}

	var result []*types.ServiceGroup
	for _, app := range applications.Apps {
		for _, task := range app.Tasks {
			group, err := m.toServiceGroup(task, &app)
			if err != nil {
				return nil, err
			}

			result = append(result, group)
			for _, service := range group.Services {
				log.WithFields(log.Fields{
					"prefix": "marathon",
					"ip":     group.IP,
					"id":     service.ID,
					"name":   service.Name,
					"port":   service.ExposedPort,
				}).Debug("Service")
			}
		}
	}

	return result, nil
}
