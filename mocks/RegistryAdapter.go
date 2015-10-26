package mocks

import "github.com/x-cray/marathon-service-registrator/types"
import "github.com/stretchr/testify/mock"

type RegistryAdapter struct {
	mock.Mock
}

func (_m *RegistryAdapter) Services() ([]*types.Service, error) {
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
func (_m *RegistryAdapter) Ping() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *RegistryAdapter) Register(service *types.Service) error {
	ret := _m.Called(service)

	var r0 error
	if rf, ok := ret.Get(0).(func(*types.Service) error); ok {
		r0 = rf(service)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *RegistryAdapter) Deregister(service *types.Service) error {
	ret := _m.Called(service)

	var r0 error
	if rf, ok := ret.Get(0).(func(*types.Service) error); ok {
		r0 = rf(service)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *RegistryAdapter) AdvertiseAddr() (string, error) {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
