// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	domain "github.com/odpf/guardian/domain"
	mock "github.com/stretchr/testify/mock"
)

// AppealService is an autogenerated mock type for the AppealService type
type AppealService struct {
	mock.Mock
}

// Create provides a mock function with given fields: user, resourceIDs
func (_m *AppealService) Create(user string, resourceIDs []uint) ([]*domain.Appeal, error) {
	ret := _m.Called(user, resourceIDs)

	var r0 []*domain.Appeal
	if rf, ok := ret.Get(0).(func(string, []uint) []*domain.Appeal); ok {
		r0 = rf(user, resourceIDs)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*domain.Appeal)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, []uint) error); ok {
		r1 = rf(user, resourceIDs)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}