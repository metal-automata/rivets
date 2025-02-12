// Code generated by mockery v2.42.1. DO NOT EDIT.

package events

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockStream is an autogenerated mock type for the Stream type
type MockStream struct {
	mock.Mock
}

type MockStream_Expecter struct {
	mock *mock.Mock
}

func (_m *MockStream) EXPECT() *MockStream_Expecter {
	return &MockStream_Expecter{mock: &_m.Mock}
}

// Close provides a mock function with given fields:
func (_m *MockStream) Close() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Close")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockStream_Close_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Close'
type MockStream_Close_Call struct {
	*mock.Call
}

// Close is a helper method to define mock.On call
func (_e *MockStream_Expecter) Close() *MockStream_Close_Call {
	return &MockStream_Close_Call{Call: _e.mock.On("Close")}
}

func (_c *MockStream_Close_Call) Run(run func()) *MockStream_Close_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockStream_Close_Call) Return(_a0 error) *MockStream_Close_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockStream_Close_Call) RunAndReturn(run func() error) *MockStream_Close_Call {
	_c.Call.Return(run)
	return _c
}

// Open provides a mock function with given fields:
func (_m *MockStream) Open() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Open")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockStream_Open_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Open'
type MockStream_Open_Call struct {
	*mock.Call
}

// Open is a helper method to define mock.On call
func (_e *MockStream_Expecter) Open() *MockStream_Open_Call {
	return &MockStream_Open_Call{Call: _e.mock.On("Open")}
}

func (_c *MockStream_Open_Call) Run(run func()) *MockStream_Open_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockStream_Open_Call) Return(_a0 error) *MockStream_Open_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockStream_Open_Call) RunAndReturn(run func() error) *MockStream_Open_Call {
	_c.Call.Return(run)
	return _c
}

// Publish provides a mock function with given fields: ctx, subject, msg
func (_m *MockStream) Publish(ctx context.Context, subject string, msg []byte) error {
	ret := _m.Called(ctx, subject, msg)

	if len(ret) == 0 {
		panic("no return value specified for Publish")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, []byte) error); ok {
		r0 = rf(ctx, subject, msg)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockStream_Publish_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Publish'
type MockStream_Publish_Call struct {
	*mock.Call
}

// Publish is a helper method to define mock.On call
//   - ctx context.Context
//   - subject string
//   - msg []byte
func (_e *MockStream_Expecter) Publish(ctx interface{}, subject interface{}, msg interface{}) *MockStream_Publish_Call {
	return &MockStream_Publish_Call{Call: _e.mock.On("Publish", ctx, subject, msg)}
}

func (_c *MockStream_Publish_Call) Run(run func(ctx context.Context, subject string, msg []byte)) *MockStream_Publish_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].([]byte))
	})
	return _c
}

func (_c *MockStream_Publish_Call) Return(_a0 error) *MockStream_Publish_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockStream_Publish_Call) RunAndReturn(run func(context.Context, string, []byte) error) *MockStream_Publish_Call {
	_c.Call.Return(run)
	return _c
}

// PublishOverwrite provides a mock function with given fields: ctx, subject, msg
func (_m *MockStream) PublishOverwrite(ctx context.Context, subject string, msg []byte) error {
	ret := _m.Called(ctx, subject, msg)

	if len(ret) == 0 {
		panic("no return value specified for PublishOverwrite")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, []byte) error); ok {
		r0 = rf(ctx, subject, msg)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockStream_PublishOverwrite_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'PublishOverwrite'
type MockStream_PublishOverwrite_Call struct {
	*mock.Call
}

// PublishOverwrite is a helper method to define mock.On call
//   - ctx context.Context
//   - subject string
//   - msg []byte
func (_e *MockStream_Expecter) PublishOverwrite(ctx interface{}, subject interface{}, msg interface{}) *MockStream_PublishOverwrite_Call {
	return &MockStream_PublishOverwrite_Call{Call: _e.mock.On("PublishOverwrite", ctx, subject, msg)}
}

func (_c *MockStream_PublishOverwrite_Call) Run(run func(ctx context.Context, subject string, msg []byte)) *MockStream_PublishOverwrite_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string), args[2].([]byte))
	})
	return _c
}

func (_c *MockStream_PublishOverwrite_Call) Return(_a0 error) *MockStream_PublishOverwrite_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockStream_PublishOverwrite_Call) RunAndReturn(run func(context.Context, string, []byte) error) *MockStream_PublishOverwrite_Call {
	_c.Call.Return(run)
	return _c
}

// PullOneMsg provides a mock function with given fields: ctx, subject
func (_m *MockStream) PullOneMsg(ctx context.Context, subject string) (Message, error) {
	ret := _m.Called(ctx, subject)

	if len(ret) == 0 {
		panic("no return value specified for PullOneMsg")
	}

	var r0 Message
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (Message, error)); ok {
		return rf(ctx, subject)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) Message); ok {
		r0 = rf(ctx, subject)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(Message)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, subject)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockStream_PullOneMsg_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'PullOneMsg'
type MockStream_PullOneMsg_Call struct {
	*mock.Call
}

// PullOneMsg is a helper method to define mock.On call
//   - ctx context.Context
//   - subject string
func (_e *MockStream_Expecter) PullOneMsg(ctx interface{}, subject interface{}) *MockStream_PullOneMsg_Call {
	return &MockStream_PullOneMsg_Call{Call: _e.mock.On("PullOneMsg", ctx, subject)}
}

func (_c *MockStream_PullOneMsg_Call) Run(run func(ctx context.Context, subject string)) *MockStream_PullOneMsg_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(string))
	})
	return _c
}

func (_c *MockStream_PullOneMsg_Call) Return(_a0 Message, _a1 error) *MockStream_PullOneMsg_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStream_PullOneMsg_Call) RunAndReturn(run func(context.Context, string) (Message, error)) *MockStream_PullOneMsg_Call {
	_c.Call.Return(run)
	return _c
}

// Subscribe provides a mock function with given fields: ctx
func (_m *MockStream) Subscribe(ctx context.Context) (MsgCh, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Subscribe")
	}

	var r0 MsgCh
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (MsgCh, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) MsgCh); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(MsgCh)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockStream_Subscribe_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Subscribe'
type MockStream_Subscribe_Call struct {
	*mock.Call
}

// Subscribe is a helper method to define mock.On call
//   - ctx context.Context
func (_e *MockStream_Expecter) Subscribe(ctx interface{}) *MockStream_Subscribe_Call {
	return &MockStream_Subscribe_Call{Call: _e.mock.On("Subscribe", ctx)}
}

func (_c *MockStream_Subscribe_Call) Run(run func(ctx context.Context)) *MockStream_Subscribe_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context))
	})
	return _c
}

func (_c *MockStream_Subscribe_Call) Return(_a0 MsgCh, _a1 error) *MockStream_Subscribe_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockStream_Subscribe_Call) RunAndReturn(run func(context.Context) (MsgCh, error)) *MockStream_Subscribe_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockStream creates a new instance of MockStream. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockStream(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockStream {
	mock := &MockStream{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
