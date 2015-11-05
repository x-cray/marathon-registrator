package marathon

import (
	"errors"
	"testing"

	"github.com/x-cray/marathon-service-registrator/types"

	log "github.com/Sirupsen/logrus"
	marathonClient "github.com/gambol99/go-marathon"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMarathonAdapter(t *testing.T) {
	log.SetLevel(log.FatalLevel)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Marathon Adapter Suite")
}

var _ = Describe("MarathonAdapter", func() {
	var (
		mockCtrl *gomock.Controller
		client   *MockMarathonClient
		resolver *MockAddressResolver
	)

	singlePortApplications := &marathonClient.Applications{
		Apps: []marathonClient.Application{
			marathonClient.Application{
				ID: "/app/staging/web-app",
				Env: map[string]string{
					"SERVICE_TAGS": "production",
					"SERVICE_NAME": "web-app",
				},
				Ports: []int{80},
				Tasks: []*marathonClient.Task{
					&marathonClient.Task{
						ID:    "web_app_2c033893-7993-11e5-8878-56847afe9799",
						AppID: "/app/staging/web-app",
						Host:  "web.eu-west-1.internal",
						Ports: []int{31045},
					},
				},
			},
		},
	}

	singlePortApplicationsWithLabels := &marathonClient.Applications{
		Apps: []marathonClient.Application{
			marathonClient.Application{
				ID: "/app/staging/web-app",
				Env: map[string]string{
					"SERVICE_TAGS": "production",
					"SERVICE_NAME": "web-app",
				},
				Labels: map[string]string{
					"SERVICE_TAGS": "production-labelled",
					"SERVICE_NAME": "web-app-labelled",
				},
				Ports: []int{80},
				Tasks: []*marathonClient.Task{
					&marathonClient.Task{
						ID:    "web_app_2c033893-7993-11e5-8878-56847afe9799",
						AppID: "/app/staging/web-app",
						Host:  "web.eu-west-1.internal",
						Ports: []int{31045},
					},
				},
			},
		},
	}

	multiPortSimpleApplications := &marathonClient.Applications{
		Apps: []marathonClient.Application{
			marathonClient.Application{
				ID: "/app/staging/web-app",
				Env: map[string]string{
					"SERVICE_TAGS": "staging",
				},
				Ports: []int{
					80,
					8080,
				},
				Tasks: []*marathonClient.Task{
					&marathonClient.Task{
						ID:    "web_app_2c033893-7993-11e5-8878-56847afe9799",
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

	multiPortComplexDockerApplications := &marathonClient.Applications{
		Apps: []marathonClient.Application{
			marathonClient.Application{
				ID: "/app/staging/web-app",
				Env: map[string]string{
					"SERVICE_TAGS":      "production",
					"SERVICE_80_NAME":   "web-app-1",
					"SERVICE_8080_NAME": "web-app-2",
				},
				Container: &marathonClient.Container{
					Docker: &marathonClient.Docker{
						PortMappings: []*marathonClient.PortMapping{
							&marathonClient.PortMapping{
								ContainerPort: 80,
							},
							&marathonClient.PortMapping{
								ContainerPort: 8080,
							},
						},
					},
				},
				Tasks: []*marathonClient.Task{
					&marathonClient.Task{
						ID:    "web_app_2c033893-7993-11e5-8878-56847afe9799",
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

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		client = NewMockMarathonClient(mockCtrl)
		resolver = NewMockAddressResolver(mockCtrl)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Services()", func() {
		It("Should forward Marathon client errors", func() {
			// Arrange.
			client.EXPECT().Applications(gomock.Any()).Return(nil, errors.New("marathon-error"))
			marathonAdapter := &marathonAdapter{client: client, resolver: resolver}

			// Act.
			_, err := marathonAdapter.Services()

			// Assert.
			Ω(err).Should(HaveOccurred())
		})

		It("Should convert Marathon single-port application to service group with 1 service", func() {
			// Arrange.
			client.EXPECT().Applications(gomock.Any()).Return(singlePortApplications, nil)
			resolver.EXPECT().Resolve("web.eu-west-1.internal").Return("10.10.10.20", nil).AnyTimes()
			marathonAdapter := &marathonAdapter{client: client, resolver: resolver}

			// Act.
			services, err := marathonAdapter.Services()

			// Assert.
			Ω(err).ShouldNot(HaveOccurred())
			Ω(services).Should(HaveLen(1))
			Ω(services[0]).Should(Equal(&types.ServiceGroup{
				ID: "web_app_2c033893-7993-11e5-8878-56847afe9799",
				IP: "10.10.10.20",
				Services: []*types.Service{
					&types.Service{
						ID:           "web_app_2c033893-7993-11e5-8878-56847afe9799:80",
						Name:         "web-app",
						Tags:         []string{"production"},
						OriginalPort: 80,
						ExposedPort:  31045,
					},
				},
			}))
		})

		It("Should convert Marathon single-port application to service group with respect to labels over environment variables", func() {
			// Arrange.
			client.EXPECT().Applications(gomock.Any()).Return(singlePortApplicationsWithLabels, nil)
			resolver.EXPECT().Resolve("web.eu-west-1.internal").Return("10.10.10.20", nil).AnyTimes()
			marathonAdapter := &marathonAdapter{client: client, resolver: resolver}

			// Act.
			services, err := marathonAdapter.Services()

			// Assert.
			Ω(err).ShouldNot(HaveOccurred())
			Ω(services).Should(HaveLen(1))
			Ω(services[0]).Should(Equal(&types.ServiceGroup{
				ID: "web_app_2c033893-7993-11e5-8878-56847afe9799",
				IP: "10.10.10.20",
				Services: []*types.Service{
					&types.Service{
						ID:           "web_app_2c033893-7993-11e5-8878-56847afe9799:80",
						Name:         "web-app-labelled",
						Tags:         []string{"production-labelled"},
						OriginalPort: 80,
						ExposedPort:  31045,
					},
				},
			}))
		})

		It("Should convert Marathon multi-port application with simple config to service group with 2 services", func() {
			// Arrange.
			client.EXPECT().Applications(gomock.Any()).Return(multiPortSimpleApplications, nil)
			resolver.EXPECT().Resolve("web.eu-west-1.internal").Return("10.10.10.20", nil).AnyTimes()
			marathonAdapter := &marathonAdapter{client: client, resolver: resolver}

			// Act.
			services, err := marathonAdapter.Services()

			// Assert.
			Ω(err).ShouldNot(HaveOccurred())
			Ω(services).Should(HaveLen(1))
			Ω(services[0].Services).Should(HaveLen(2))
			Ω(services[0]).Should(Equal(&types.ServiceGroup{
				ID: "web_app_2c033893-7993-11e5-8878-56847afe9799",
				IP: "10.10.10.20",
				Services: []*types.Service{
					&types.Service{
						ID:           "web_app_2c033893-7993-11e5-8878-56847afe9799:80",
						Name:         "web-app-80",
						Tags:         []string{"staging"},
						OriginalPort: 80,
						ExposedPort:  31045,
					},
					&types.Service{
						ID:           "web_app_2c033893-7993-11e5-8878-56847afe9799:8080",
						Name:         "web-app-8080",
						Tags:         []string{"staging"},
						OriginalPort: 8080,
						ExposedPort:  31046,
					},
				},
			}))
		})

		It("Should convert Marathon multi-port dockerized application with complex config to service group with 2 services", func() {
			// Arrange.
			client.EXPECT().Applications(gomock.Any()).Return(multiPortComplexDockerApplications, nil)
			resolver.EXPECT().Resolve("web.eu-west-1.internal").Return("10.10.10.20", nil).AnyTimes()
			marathonAdapter := &marathonAdapter{client: client, resolver: resolver}

			// Act.
			services, err := marathonAdapter.Services()

			// Assert.
			Ω(err).ShouldNot(HaveOccurred())
			Ω(services).Should(HaveLen(1))
			Ω(services[0].Services).Should(HaveLen(2))
			Ω(services[0]).Should(Equal(&types.ServiceGroup{
				ID: "web_app_2c033893-7993-11e5-8878-56847afe9799",
				IP: "10.10.10.20",
				Services: []*types.Service{
					&types.Service{
						ID:           "web_app_2c033893-7993-11e5-8878-56847afe9799:80",
						Name:         "web-app-1",
						Tags:         []string{"production"},
						OriginalPort: 80,
						ExposedPort:  31045,
					},
					&types.Service{
						ID:           "web_app_2c033893-7993-11e5-8878-56847afe9799:8080",
						Name:         "web-app-2",
						Tags:         []string{"production"},
						OriginalPort: 8080,
						ExposedPort:  31046,
					},
				},
			}))
		})
	})
})
