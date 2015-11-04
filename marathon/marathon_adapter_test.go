package marathon

import (
	"errors"
	"testing"

	"github.com/x-cray/marathon-service-registrator/mocks"
	"github.com/x-cray/marathon-service-registrator/types"

	log "github.com/Sirupsen/logrus"
	. "github.com/franela/goblin"
	"github.com/stretchr/testify/mock"
	marathonClient "github.com/x-cray/go-marathon"
)

func Test(t *testing.T) {
	log.SetLevel(log.FatalLevel)

	g := Goblin(t)
	g.Describe("MarathonAdapter", func() {
		var client *mocks.MarathonClient
		var resolver *mocks.AddressResolver

		applications := &marathonClient.Applications{
			Apps: []marathonClient.Application{
				marathonClient.Application{
					ID: "/app/staging/web-app",
					Env: map[string]string{
						"SERVICE_TAGS":      "production",
						"SERVICE_80_NAME":   "web-app-1",
						"SERVICE_8080_NAME": "web-app-2",
					},
					Ports: []int{
						80,
						8080,
					},
					Tasks: []*marathonClient.Task{
						&marathonClient.Task{
							ID:    "0000-web-app-12345098765",
							AppID: "/app/staging/web-app",
							Host:  "web.eu-west-1.internal",
							Ports: []int{
								31045,
								31046,
							},
						},
					},
				},
			},
		}

		g.BeforeEach(func() {
			client = &mocks.MarathonClient{}
			resolver = &mocks.AddressResolver{}
		})

		g.Describe("Services()", func() {
			g.It("Should forward Marathon client errors", func() {
				// Arrange.
				client.On("Applications", mock.AnythingOfType("url.Values")).Return(nil, errors.New("marathon-error"))
				marathonAdapter := &MarathonAdapter{client: client, resolver: resolver}

				// Act.
				_, err := marathonAdapter.Services()

				// Assert.
				client.AssertExpectations(t)
				g.Assert(err.Error()).Equal("marathon-error")
			})

			g.It("Should convert Marathon applications list to services", func() {
				// Arrange.
				client.On("Applications", mock.AnythingOfType("url.Values")).Return(applications, nil)
				resolver.On("Resolve", mock.AnythingOfType("string")).Return("127.0.0.1", nil)
				marathonAdapter := &MarathonAdapter{client: client, resolver: resolver}

				// Act.
				services, err := marathonAdapter.Services()

				// Assert.
				client.AssertExpectations(t)
				resolver.AssertExpectations(t)
				g.Assert(len(services)).Equal(2)
				g.Assert(services[0]).Equal(&types.Service{
					ID:   "0000-web-app-12345098765:31045",
					Name: "web-app-80",
					IP:   "127.0.0.1",
					Port: 31045,
					Tags: []string {"production"},
				})
				g.Assert(services[1]).Equal(&types.Service{
					ID:   "0000-web-app-12345098765:31046",
					Name: "web-app-8080",
					IP:   "127.0.0.1",
					Port: 31046,
					Tags: []string {"production"},
				})
				g.Assert(err).Equal(nil)
			})
		})
	})
}
