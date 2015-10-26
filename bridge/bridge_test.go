package bridge

import (
	"testing"

	"github.com/x-cray/marathon-service-registrator/mocks"
	"github.com/x-cray/marathon-service-registrator/types"

	. "github.com/franela/goblin"
	"github.com/stretchr/testify/mock"
)

func Test(t *testing.T) {
	g := Goblin(t)
	g.Describe("Bridge", func() {
		var schedulerAdapter *mocks.SchedulerAdapter
		var registryAdapter *mocks.RegistryAdapter

		g.BeforeEach(func() {
			schedulerAdapter = new(mocks.SchedulerAdapter)
			registryAdapter = new(mocks.RegistryAdapter)
		})

		g.Describe("Sync()", func() {
			g.It("Should do nothing if service sets in scheduler and registry match", func() {
				// Arrange.
				schedulerServices := []*types.Service{
					&types.Service{
						Name: "db-server",
						IP:   "10.10.10.10",
						Port: 27017,
					},
					&types.Service{
						Name: "app-server",
						IP:   "10.10.10.10",
						Port: 3000,
					},
				}
				registryServices := []*types.Service{
					&types.Service{
						Name: "db-server",
						IP:   "10.10.10.10",
						Port: 27017,
					},
					&types.Service{
						Name: "app-server",
						IP:   "10.10.10.10",
						Port: 3000,
					},
				}
				schedulerAdapter.On("Services").Return(schedulerServices, nil)
				registryAdapter.On("Services").Return(registryServices, nil)
				registryAdapter.On("AdvertiseAddr").Return("10.10.10.10", nil)
				bridge := &Bridge{
					scheduler: schedulerAdapter,
					registry:  registryAdapter,
				}

				// Act.
				bridge.Sync()

				// Assert.
				schedulerAdapter.AssertExpectations(t)
				registryAdapter.AssertExpectations(t)
				registryAdapter.AssertNotCalled(t, "Register")
				registryAdapter.AssertNotCalled(t, "Deregister")
			})

			g.It("Should register 1 service absent in registry but present in scheduler", func() {
				// Arrange.
				schedulerServices := []*types.Service{
					&types.Service{
						Name: "db-server",
						IP:   "10.10.10.10",
						Port: 27017,
					},
					&types.Service{
						Name: "app-server",
						IP:   "10.10.10.10",
						Port: 3000,
					},
				}
				registryServices := []*types.Service{
					&types.Service{
						Name: "app-server",
						IP:   "10.10.10.10",
						Port: 3000,
					},
				}
				schedulerAdapter.On("Services").Return(schedulerServices, nil)
				registryAdapter.On("Services").Return(registryServices, nil)
				registryAdapter.On("AdvertiseAddr").Return("10.10.10.10", nil)
				registryAdapter.On("Register", mock.AnythingOfType("*types.Service")).Run(func(args mock.Arguments) {
					service := args.Get(0).(*types.Service)

					// Method call assertions.
					g.Assert(service.Name).Equal("db-server")
					g.Assert(service.IP).Equal("10.10.10.10")
					g.Assert(service.Port).Equal(27017)
				}).Return(nil)
				bridge := &Bridge{
					scheduler: schedulerAdapter,
					registry:  registryAdapter,
				}

				// Act.
				bridge.Sync()

				// Assert.
				schedulerAdapter.AssertExpectations(t)
				registryAdapter.AssertExpectations(t)
				registryAdapter.AssertNotCalled(t, "Deregister")
			})

			g.It("Should register 2 services absent in registry but present in scheduler", func() {
				// Arrange.
				schedulerServices := []*types.Service{
					&types.Service{
						Name: "db-server",
						IP:   "10.10.10.10",
						Port: 27017,
					},
					&types.Service{
						Name: "app-server",
						IP:   "10.10.10.10",
						Port: 3000,
					},
				}
				registryServices := []*types.Service{}
				schedulerAdapter.On("Services").Return(schedulerServices, nil)
				registryAdapter.On("Services").Return(registryServices, nil)
				registryAdapter.On("AdvertiseAddr").Return("10.10.10.10", nil)
				registryAdapter.On("Register", mock.AnythingOfType("*types.Service")).Run(func(args mock.Arguments) {
					service := args.Get(0).(*types.Service)

					// Method call assertions.
					if service.Name == "db-server" {
						g.Assert(service.IP).Equal("10.10.10.10")
						g.Assert(service.Port).Equal(27017)
					} else if service.Name == "app-server" {
						g.Assert(service.IP).Equal("10.10.10.10")
						g.Assert(service.Port).Equal(3000)
					} else {
						g.Fail("Tried to register an unknown service: " + service.Name)
					}
				}).Return(nil)
				bridge := &Bridge{
					scheduler: schedulerAdapter,
					registry:  registryAdapter,
				}

				// Act.
				bridge.Sync()

				// Assert.
				schedulerAdapter.AssertExpectations(t)
				registryAdapter.AssertExpectations(t)
				registryAdapter.AssertNumberOfCalls(t, "Register", 2)
				registryAdapter.AssertNotCalled(t, "Deregister")
			})
		})
	})
}
