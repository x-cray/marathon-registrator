package marathon

import (
	"net/url"

	marathonClient "github.com/gambol99/go-marathon"
)

// Client is the excerpt interface from Marathon API client to generate mocks.
type Client interface {
	Applications(url.Values) (*marathonClient.Applications, error)
	AddEventsListener(channel marathonClient.EventsChannel, filter int) error
	RemoveEventsListener(channel marathonClient.EventsChannel)
}
