package mocks

import "github.com/stretchr/testify/mock"

import "net/url"
import marathonClient "github.com/x-cray/go-marathon"

type MarathonClient struct {
	mock.Mock
}

func (_m *MarathonClient) Applications(_a0 url.Values) (*marathonClient.Applications, error) {
	ret := _m.Called(_a0)

	var r0 *marathonClient.Applications
	if rf, ok := ret.Get(0).(func(url.Values) *marathonClient.Applications); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*marathonClient.Applications)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(url.Values) error); ok {
		r1 = rf(_a0)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
func (_m *MarathonClient) AddEventsListener(channel marathonClient.EventsChannel, filter int) error {
	ret := _m.Called(channel, filter)

	var r0 error
	if rf, ok := ret.Get(0).(func(marathonClient.EventsChannel, int) error); ok {
		r0 = rf(channel, filter)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
func (_m *MarathonClient) RemoveEventsListener(channel marathonClient.EventsChannel) {
	_m.Called(channel)
}
