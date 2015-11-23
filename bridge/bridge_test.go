package bridge

import (
	"errors"
	"testing"

	"github.com/x-cray/marathon-registrator/types"

	log "github.com/Sirupsen/logrus"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestBridge(t *testing.T) {
	log.SetLevel(log.FatalLevel)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bridge Suite")
}

var _ = Describe("Bridge", func() {
	var (
		mockCtrl         *gomock.Controller
		schedulerAdapter *types.MockSchedulerAdapter
		registryAdapter  *types.MockRegistryAdapter
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		schedulerAdapter = types.NewMockSchedulerAdapter(mockCtrl)
		registryAdapter = types.NewMockRegistryAdapter(mockCtrl)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("Sync()", func() {
		It("Should forward errors received from RegistryAdapter.Services()", func() {
			// Arrange.
			registryAdapter.EXPECT().Services().Return([]*types.ServiceGroup{}, errors.New("registry-error"))
			bridge := &Bridge{
				scheduler: schedulerAdapter,
				registry:  registryAdapter,
			}

			// Act.
			err := bridge.Sync()

			// Assert.
			Ω(err).Should(HaveOccurred())
		})

		It("Should forward errors received from RegistryAdapter.AdvertiseAddr()", func() {
			// Arrange.
			registryAdapter.EXPECT().Services().Return([]*types.ServiceGroup{}, nil)
			registryAdapter.EXPECT().AdvertiseAddr().Return("", errors.New("registry-error"))
			bridge := &Bridge{
				scheduler: schedulerAdapter,
				registry:  registryAdapter,
			}

			// Act.
			err := bridge.Sync()

			// Assert.
			Ω(err).Should(HaveOccurred())
		})

		It("Should forward errors received from SchedulerAdapter.Services()", func() {
			// Arrange.
			schedulerAdapter.EXPECT().Services().Return([]*types.ServiceGroup{}, errors.New("scheduler-error"))
			registryAdapter.EXPECT().Services().Return([]*types.ServiceGroup{}, nil)
			registryAdapter.EXPECT().AdvertiseAddr().Return("", nil)
			bridge := &Bridge{
				scheduler: schedulerAdapter,
				registry:  registryAdapter,
			}

			// Act.
			err := bridge.Sync()

			// Assert.
			Ω(err).Should(HaveOccurred())
		})

		It("Should do nothing if service sets in scheduler and registry are empty", func() {
			// Arrange.
			schedulerAdapter.EXPECT().Services().Return([]*types.ServiceGroup{}, nil)
			registryAdapter.EXPECT().Services().Return([]*types.ServiceGroup{}, nil)
			registryAdapter.EXPECT().AdvertiseAddr().Return("10.10.10.10", nil)

			// Call assertions
			registryAdapter.EXPECT().Register(gomock.Any()).Times(0)
			registryAdapter.EXPECT().Deregister(gomock.Any()).Times(0)

			bridge := &Bridge{
				scheduler: schedulerAdapter,
				registry:  registryAdapter,
			}

			// Act.
			bridge.Sync()
		})

		It("Should do nothing if service sets in scheduler and registry match", func() {
			// Arrange.
			schedulerServices := []*types.ServiceGroup{
				&types.ServiceGroup{
					ID: "db_server_2c033893-7993-11e5-8878-56847afe9799",
					IP: "10.10.10.10",
					Services: []*types.Service{
						&types.Service{
							ID:           "db_server_2c033893-7993-11e5-8878-56847afe9799:27017",
							Name:         "db-server",
							Healthy:      true,
							OriginalPort: 27017,
							ExposedPort:  31045,
						},
					},
				},
				&types.ServiceGroup{
					ID: "app_server_5877d4d2-7b4b-11e5-b945-56847afe9799",
					IP: "10.10.10.10",
					Services: []*types.Service{
						&types.Service{
							ID:           "app_server_5877d4d2-7b4b-11e5-b945-56847afe9799:3000",
							Name:         "app-server",
							Healthy:      true,
							OriginalPort: 3000,
							ExposedPort:  31046,
						},
					},
				},
			}
			registryServices := []*types.ServiceGroup{
				&types.ServiceGroup{
					ID: "db_server_2c033893-7993-11e5-8878-56847afe9799",
					IP: "10.10.10.10",
					Services: []*types.Service{
						&types.Service{
							ID:          "db_server_2c033893-7993-11e5-8878-56847afe9799:27017",
							Name:        "db-server",
							ExposedPort: 31045,
						},
					},
				},
				&types.ServiceGroup{
					ID: "app_server_5877d4d2-7b4b-11e5-b945-56847afe9799",
					IP: "10.10.10.10",
					Services: []*types.Service{
						&types.Service{
							ID:          "app_server_5877d4d2-7b4b-11e5-b945-56847afe9799:3000",
							Name:        "app-server",
							ExposedPort: 31046,
						},
					},
				},
			}
			schedulerAdapter.EXPECT().Services().Return(schedulerServices, nil)
			registryAdapter.EXPECT().Services().Return(registryServices, nil)
			registryAdapter.EXPECT().AdvertiseAddr().Return("10.10.10.10", nil)

			// Call assertions
			registryAdapter.EXPECT().Register(gomock.Any()).Times(0)
			registryAdapter.EXPECT().Deregister(gomock.Any()).Times(0)

			bridge := &Bridge{
				scheduler: schedulerAdapter,
				registry:  registryAdapter,
			}

			// Act.
			bridge.Sync()
		})

		It("Should register 1 service absent from registry but present in scheduler", func() {
			// Arrange.
			schedulerServices := []*types.ServiceGroup{
				&types.ServiceGroup{
					ID: "db_server_2c033893-7993-11e5-8878-56847afe9799",
					IP: "10.10.10.10",
					Services: []*types.Service{
						&types.Service{
							ID:           "db_server_2c033893-7993-11e5-8878-56847afe9799:27017",
							Name:         "db-server",
							Healthy:      true,
							OriginalPort: 27017,
							ExposedPort:  31045,
						},
					},
				},
				&types.ServiceGroup{
					ID: "app_server_5877d4d2-7b4b-11e5-b945-56847afe9799",
					IP: "10.10.10.10",
					Services: []*types.Service{
						&types.Service{
							ID:           "app_server_5877d4d2-7b4b-11e5-b945-56847afe9799:3000",
							Name:         "app-server",
							Healthy:      true,
							OriginalPort: 3000,
							ExposedPort:  31046,
						},
					},
				},
			}
			registryServices := []*types.ServiceGroup{
				&types.ServiceGroup{
					ID: "app_server_5877d4d2-7b4b-11e5-b945-56847afe9799",
					IP: "10.10.10.10",
					Services: []*types.Service{
						&types.Service{
							ID:          "app_server_5877d4d2-7b4b-11e5-b945-56847afe9799:3000",
							Name:        "app-server",
							ExposedPort: 31046,
						},
					},
				},
			}
			schedulerAdapter.EXPECT().Services().Return(schedulerServices, nil)
			registryAdapter.EXPECT().Services().Return(registryServices, nil)
			registryAdapter.EXPECT().AdvertiseAddr().Return("10.10.10.10", nil)
			registryAdapter.EXPECT().Register(gomock.Any()).Do(func(group *types.ServiceGroup) {
				Ω(group.IP).Should(Equal("10.10.10.10"))
				Ω(group.Services).Should(HaveLen(1))
				service := group.Services[0]

				Ω(service.ID).Should(Equal("db_server_2c033893-7993-11e5-8878-56847afe9799:27017"))
				Ω(service.Name).Should(Equal("db-server"))
				Ω(service.ExposedPort).Should(Equal(31045))
			}).Return(nil).Times(1)
			registryAdapter.EXPECT().Deregister(gomock.Any()).Times(0)

			bridge := &Bridge{
				scheduler: schedulerAdapter,
				registry:  registryAdapter,
			}

			// Act.
			bridge.Sync()
		})

		It("Should deregister 1 service absent from scheduler but present in registry", func() {
			// Arrange.
			schedulerServices := []*types.ServiceGroup{
				&types.ServiceGroup{
					ID: "app_server_5877d4d2-7b4b-11e5-b945-56847afe9799",
					IP: "10.10.10.10",
					Services: []*types.Service{
						&types.Service{
							ID:           "app_server_5877d4d2-7b4b-11e5-b945-56847afe9799:3000",
							Name:         "app-server",
							Healthy:      true,
							OriginalPort: 3000,
							ExposedPort:  31046,
						},
					},
				},
			}
			registryServices := []*types.ServiceGroup{
				&types.ServiceGroup{
					ID: "db_server_2c033893-7993-11e5-8878-56847afe9799",
					IP: "10.10.10.10",
					Services: []*types.Service{
						&types.Service{
							ID:          "db_server_2c033893-7993-11e5-8878-56847afe9799:27017",
							Name:        "db-server",
							ExposedPort: 31045,
						},
					},
				},
				&types.ServiceGroup{
					ID: "app_server_5877d4d2-7b4b-11e5-b945-56847afe9799",
					IP: "10.10.10.10",
					Services: []*types.Service{
						&types.Service{
							ID:          "app_server_5877d4d2-7b4b-11e5-b945-56847afe9799:3000",
							Name:        "app-server",
							ExposedPort: 31046,
						},
					},
				},
			}
			schedulerAdapter.EXPECT().Services().Return(schedulerServices, nil)
			registryAdapter.EXPECT().Services().Return(registryServices, nil)
			registryAdapter.EXPECT().AdvertiseAddr().Return("10.10.10.10", nil)
			registryAdapter.EXPECT().Deregister(gomock.Any()).Do(func(group *types.ServiceGroup) {
				Ω(group.IP).Should(Equal("10.10.10.10"))
				Ω(group.Services).Should(HaveLen(1))
				service := group.Services[0]

				Ω(service.ID).Should(Equal("db_server_2c033893-7993-11e5-8878-56847afe9799:27017"))
				Ω(service.Name).Should(Equal("db-server"))
				Ω(service.ExposedPort).Should(Equal(31045))
			}).Return(nil).Times(1)
			registryAdapter.EXPECT().Register(gomock.Any()).Times(0)

			bridge := &Bridge{
				scheduler: schedulerAdapter,
				registry:  registryAdapter,
			}

			// Act.
			bridge.Sync()
		})

		It("Should register 2 services absent from registry but present in scheduler", func() {
			// Arrange.
			schedulerServices := []*types.ServiceGroup{
				&types.ServiceGroup{
					ID: "db_server_2c033893-7993-11e5-8878-56847afe9799",
					IP: "10.10.10.10",
					Services: []*types.Service{
						&types.Service{
							ID:           "db_server_2c033893-7993-11e5-8878-56847afe9799:27017",
							Name:         "db-server",
							Healthy:      true,
							OriginalPort: 27017,
							ExposedPort:  31045,
						},
					},
				},
				&types.ServiceGroup{
					ID: "app_server_5877d4d2-7b4b-11e5-b945-56847afe9799",
					IP: "10.10.10.10",
					Services: []*types.Service{
						&types.Service{
							ID:           "app_server_5877d4d2-7b4b-11e5-b945-56847afe9799:3000",
							Name:         "app-server",
							Healthy:      true,
							OriginalPort: 3000,
							ExposedPort:  31046,
						},
					},
				},
			}
			registryServices := []*types.ServiceGroup{}
			schedulerAdapter.EXPECT().Services().Return(schedulerServices, nil)
			registryAdapter.EXPECT().Services().Return(registryServices, nil)
			registryAdapter.EXPECT().AdvertiseAddr().Return("10.10.10.10", nil)
			registryAdapter.EXPECT().Register(gomock.Any()).Do(func(group *types.ServiceGroup) {
				Ω(group.IP).Should(Equal("10.10.10.10"))
				Ω(group.Services).Should(HaveLen(1))
				service := group.Services[0]

				if service.Name == "db-server" {
					Ω(service.ExposedPort).Should(Equal(31045))
				} else if service.Name == "app-server" {
					Ω(service.ExposedPort).Should(Equal(31046))
				} else {
					Fail("Tried to register an unknown service: " + service.Name)
				}
			}).Return(nil).Times(2)
			registryAdapter.EXPECT().Deregister(gomock.Any()).Times(0)

			bridge := &Bridge{
				scheduler: schedulerAdapter,
				registry:  registryAdapter,
			}

			// Act.
			bridge.Sync()
		})

		It("Should not try to register services from address different than registry advertized one", func() {
			// Arrange.
			schedulerServices := []*types.ServiceGroup{
				&types.ServiceGroup{
					ID: "db_server_2c033893-7993-11e5-8878-56847afe9799",
					IP: "10.10.10.10",
					Services: []*types.Service{
						&types.Service{
							ID:           "db_server_2c033893-7993-11e5-8878-56847afe9799:27017",
							Name:         "db-server",
							Healthy:      true,
							OriginalPort: 27017,
							ExposedPort:  31045,
						},
					},
				},
				&types.ServiceGroup{
					ID: "app_server_5877d4d2-7b4b-11e5-b945-56847afe9799",
					IP: "10.10.10.20",
					Services: []*types.Service{
						&types.Service{
							ID:           "app_server_5877d4d2-7b4b-11e5-b945-56847afe9799:3000",
							Name:         "app-server",
							Healthy:      true,
							OriginalPort: 3000,
							ExposedPort:  31046,
						},
					},
				},
			}
			registryServices := []*types.ServiceGroup{}
			schedulerAdapter.EXPECT().Services().Return(schedulerServices, nil)
			registryAdapter.EXPECT().Services().Return(registryServices, nil)
			registryAdapter.EXPECT().AdvertiseAddr().Return("10.10.10.10", nil)
			registryAdapter.EXPECT().Register(gomock.Any()).Do(func(group *types.ServiceGroup) {
				Ω(group.IP).Should(Equal("10.10.10.10"))
				Ω(group.Services).Should(HaveLen(1))
				service := group.Services[0]

				// Method call assertions.
				Ω(service.Name).Should(Equal("db-server"))
				Ω(service.ExposedPort).Should(Equal(31045))
			}).Return(nil).Times(1)
			registryAdapter.EXPECT().Deregister(gomock.Any()).Times(0)

			bridge := &Bridge{
				scheduler: schedulerAdapter,
				registry:  registryAdapter,
			}

			// Act.
			bridge.Sync()
		})

		It("Should not try to register unhealthy services", func() {
			// Arrange.
			schedulerServices := []*types.ServiceGroup{
				&types.ServiceGroup{
					ID: "db_server_2c033893-7993-11e5-8878-56847afe9799",
					IP: "10.10.10.10",
					Services: []*types.Service{
						&types.Service{
							ID:           "db_server_2c033893-7993-11e5-8878-56847afe9799:27017",
							Name:         "db-server",
							Healthy:      false,
							OriginalPort: 27017,
							ExposedPort:  31045,
						},
					},
				},
			}
			registryServices := []*types.ServiceGroup{}
			schedulerAdapter.EXPECT().Services().Return(schedulerServices, nil)
			registryAdapter.EXPECT().Services().Return(registryServices, nil)
			registryAdapter.EXPECT().AdvertiseAddr().Return("10.10.10.10", nil)
			registryAdapter.EXPECT().Register(gomock.Any()).Times(0)
			registryAdapter.EXPECT().Deregister(gomock.Any()).Times(0)

			bridge := &Bridge{
				scheduler: schedulerAdapter,
				registry:  registryAdapter,
			}

			// Act.
			bridge.Sync()
		})
	})

	Describe("ProcessSchedulerEvents()", func() {
		It("Should forward errors received from SchedulerAdapter.ListenForEvents()", func() {
			// Arrange.
			schedulerAdapter.EXPECT().ListenForEvents(gomock.Any()).Return(errors.New("scheduler-error"))
			bridge := &Bridge{
				scheduler: schedulerAdapter,
				registry:  registryAdapter,
			}

			// Act.
			err := bridge.ProcessSchedulerEvents()

			// Assert.
			Ω(err).Should(HaveOccurred())
		})

		It("Should process event and exit on scheduler channel closure", func() {
			// Arrange.
			schedulerAdapter.EXPECT().ListenForEvents(gomock.Any()).Do(func(channel types.EventsChannel) {
				channel <- &types.ServiceEvent{}
				close(channel)
			}).Return(nil)
			bridge := &Bridge{
				scheduler: schedulerAdapter,
				registry:  registryAdapter,
			}

			// Act.
			err := bridge.ProcessSchedulerEvents()

			// Assert.
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("Should refresh service lists on ServiceStarted event", func() {
			// Arrange.
			registryAdapter.EXPECT().AdvertiseAddr().Return("10.10.10.10", nil)
			schedulerAdapter.EXPECT().ListenForEvents(gomock.Any()).Do(func(channel types.EventsChannel) {
				channel <- &types.ServiceEvent{
					ServiceID: "db_server_2c033893-7993-11e5-8878-56847afe9799:27017",
					IP:        "10.10.10.10",
					Action:    types.ServiceStarted,
				}
				close(channel)
			}).Return(nil)
			schedulerServices := []*types.ServiceGroup{}
			schedulerAdapter.EXPECT().Services().Return(schedulerServices, nil).Times(1)
			bridge := &Bridge{
				scheduler: schedulerAdapter,
				registry:  registryAdapter,
			}

			// Act.
			err := bridge.ProcessSchedulerEvents()

			// Assert.
			Ω(err).ShouldNot(HaveOccurred())
		})
	})
})
