package marathon

import (
	"log"
	"os"
	"net"
	"net/url"

	"github.com/x-cray/marathon-service-registrator/types"

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

	log.Printf("Connecting to Marathon at %v\n", marathonUri)
	client, err := marathon.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &MarathonAdapter{client: client}, nil
}

func (m *MarathonAdapter) ListenForEvents() {

}

func (m *MarathonAdapter) Services() ([]*types.Service, error) {
	params := make(url.Values)
	params.Add("embed", "apps.tasks")
	applications, err := m.client.Applications(params)
	if err != nil {
		return nil, err
	}

	result := make([]*types.Service, len(applications.Apps))
	for _, app := range applications.Apps {
		log.Printf("App %v", app.ID)
		for _, task := range app.Tasks {
			taskIP, err := net.ResolveIPAddr("ip", task.Host)
			if err != nil {
				return nil, err
			}
			log.Printf("- task %s at %s, ports: %v", task.ID, task.Host, task.Ports)
			for port := range task.Ports {
				result = append(result, &types.Service{
					ID: task.ID,
					Name: app.ID,
					IP: taskIP.String(),
					Port: port,
				})
			}
		}
	}

	return result, nil
}
