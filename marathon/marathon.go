package marathon

import (
	"net"
	"net/url"
	"os"
	"strings"

	"github.com/x-cray/marathon-service-registrator/types"

	log "github.com/Sirupsen/logrus"
	marathon "github.com/gambol99/go-marathon"
)

type MarathonAdapter struct {
	client marathon.Marathon
	events chan string
}

func New(marathonUri string) (*MarathonAdapter, error) {
	config := marathon.NewDefaultConfig()
	config.URL = marathonUri
	config.LogOutput = os.Stdout

	log.WithField("prefix", "marathon").Infof("Connecting to Marathon at %v", marathonUri)
	client, err := marathon.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &MarathonAdapter{client: client}, nil
}

func (m *MarathonAdapter) ListenForEvents() (types.EventsChannel, error) {
	update := make(marathon.EventsChannel, 5)
	result := make(types.EventsChannel, 5)
	if err := m.client.AddEventsListener(update, marathon.EVENTS_APPLICATIONS); err != nil {
		return nil, err
	} else {
		for {
			event := <-update
			result <- &types.Event{
				ID:    event.ID,
				Name:  event.Name,
				Event: event.Event,
			}
		}
	}

	return result, nil
}

func serviceFromTask(task *marathon.Task, port int, app *marathon.Application) (*types.Service, error) {
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
				service, err := serviceFromTask(task, port, &app)
				if err != nil {
					return nil, err
				}

				log.WithFields(log.Fields{
					"prefix": "marathon",
					"name":   service.Name,
					"ip":     service.IP,
					"port":   service.Port,
				}).Debugf("Service")
				result = append(result, service)
			}
		}
	}

	return result, nil
}
