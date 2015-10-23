package marathon

import (
	"os"
	"net"
	"net/url"

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

func (m *MarathonAdapter) ListenForEvents() {

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
		log.WithFields(log.Fields{
			"prefix": "marathon",
			"id": app.ID,
		}).Debug("App")
		for _, task := range app.Tasks {
			taskIP, err := net.ResolveIPAddr("ip", task.Host)
			if err != nil {
				return nil, err
			}

			log.WithFields(log.Fields{
				"prefix": "marathon",
				"host": task.Host,
				"ip": taskIP,
				"ports": task.Ports,
			}).Debug("Task")

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
