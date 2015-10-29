package mocks

import "github.com/x-cray/marathon-service-registrator/types"
import "github.com/stretchr/testify/mock"

type SchedulerAdapter struct {
	mock.Mock
}

func (_m *SchedulerAdapter) Services() ([]*types.Service, error) {
	ret := _m.Called()

	var r0 []*types.Service
	if rf, ok := ret.Get(0).(func() []*types.Service); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*types.Service)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *SchedulerAdapter) ListenForEvents(channel types.EventsChannel) error {
	ret := _m.Called(channel)

	var r0 error
	if rf, ok := ret.Get(0).(func(types.EventsChannel) error); ok {
		r0 = rf(channel)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
