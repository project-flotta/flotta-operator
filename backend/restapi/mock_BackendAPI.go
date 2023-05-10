// Code generated by mockery v1.0.0. DO NOT EDIT.

package restapi

import backend "github.com/project-flotta/flotta-operator/backend/restapi/operations/backend"
import context "context"
import middleware "github.com/go-openapi/runtime/middleware"
import mock "github.com/stretchr/testify/mock"

// MockBackendAPI is an autogenerated mock type for the BackendAPI type
type MockBackendAPI struct {
	mock.Mock
}

// EnrolDevice provides a mock function with given fields: ctx, params
func (_m *MockBackendAPI) EnrolDevice(ctx context.Context, params backend.EnrolDeviceParams) middleware.Responder {
	ret := _m.Called(ctx, params)

	var r0 middleware.Responder
	if rf, ok := ret.Get(0).(func(context.Context, backend.EnrolDeviceParams) middleware.Responder); ok {
		r0 = rf(ctx, params)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(middleware.Responder)
		}
	}

	return r0
}

// GetDeviceConfiguration provides a mock function with given fields: ctx, params
func (_m *MockBackendAPI) GetDeviceConfiguration(ctx context.Context, params backend.GetDeviceConfigurationParams) middleware.Responder {
	ret := _m.Called(ctx, params)

	var r0 middleware.Responder
	if rf, ok := ret.Get(0).(func(context.Context, backend.GetDeviceConfigurationParams) middleware.Responder); ok {
		r0 = rf(ctx, params)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(middleware.Responder)
		}
	}

	return r0
}

// GetPlaybookExecutions provides a mock function with given fields: ctx, params
func (_m *MockBackendAPI) GetPlaybookExecutions(ctx context.Context, params backend.GetPlaybookExecutionsParams) middleware.Responder {
	ret := _m.Called(ctx, params)

	var r0 middleware.Responder
	if rf, ok := ret.Get(0).(func(context.Context, backend.GetPlaybookExecutionsParams) middleware.Responder); ok {
		r0 = rf(ctx, params)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(middleware.Responder)
		}
	}

	return r0
}

// GetRegistrationStatus provides a mock function with given fields: ctx, params
func (_m *MockBackendAPI) GetRegistrationStatus(ctx context.Context, params backend.GetRegistrationStatusParams) middleware.Responder {
	ret := _m.Called(ctx, params)

	var r0 middleware.Responder
	if rf, ok := ret.Get(0).(func(context.Context, backend.GetRegistrationStatusParams) middleware.Responder); ok {
		r0 = rf(ctx, params)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(middleware.Responder)
		}
	}

	return r0
}

// RegisterDevice provides a mock function with given fields: ctx, params
func (_m *MockBackendAPI) RegisterDevice(ctx context.Context, params backend.RegisterDeviceParams) middleware.Responder {
	ret := _m.Called(ctx, params)

	var r0 middleware.Responder
	if rf, ok := ret.Get(0).(func(context.Context, backend.RegisterDeviceParams) middleware.Responder); ok {
		r0 = rf(ctx, params)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(middleware.Responder)
		}
	}

	return r0
}

// UpdateHeartBeat provides a mock function with given fields: ctx, params
func (_m *MockBackendAPI) UpdateHeartBeat(ctx context.Context, params backend.UpdateHeartBeatParams) middleware.Responder {
	ret := _m.Called(ctx, params)

	var r0 middleware.Responder
	if rf, ok := ret.Get(0).(func(context.Context, backend.UpdateHeartBeatParams) middleware.Responder); ok {
		r0 = rf(ctx, params)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(middleware.Responder)
		}
	}

	return r0
}
