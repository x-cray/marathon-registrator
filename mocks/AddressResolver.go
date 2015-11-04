package mocks

import "github.com/stretchr/testify/mock"

type AddressResolver struct {
	mock.Mock
}

func (_m *AddressResolver) Resolve(hostname string) (string, error) {
	ret := _m.Called(hostname)

	var r0 string
	if rf, ok := ret.Get(0).(func(string) string); ok {
		r0 = rf(hostname)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(hostname)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
