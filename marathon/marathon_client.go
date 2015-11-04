package marathon

import (
	"net/url"

	marathonClient "github.com/x-cray/go-marathon"
)

type MarathonClient interface {
	Applications(url.Values) (*marathonClient.Applications, error)
	AddEventsListener(channel marathonClient.EventsChannel, filter int) error
	RemoveEventsListener(channel marathonClient.EventsChannel)
}
