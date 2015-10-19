package marathon

import (
	"log"
	"os"
	"net/url"

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

func (m *MarathonAdapter) Applications() (*marathon.Applications, error) {
	params := make(url.Values)
	params.Add("embed", "apps.tasks")
	applications, err := m.client.Applications(params)
	if err != nil {
		return nil, err
	}

	return applications, nil
}
