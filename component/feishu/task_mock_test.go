// Code generated by mockery v2.53.2. DO NOT EDIT.

package feishu

import mock "github.com/stretchr/testify/mock"

// MockTask is an autogenerated mock type for the Task type
type MockTask struct {
	mock.Mock
}

type MockTask_Expecter struct {
	mock *mock.Mock
}

func (_m *MockTask) EXPECT() *MockTask_Expecter {
	return &MockTask_Expecter{mock: &_m.Mock}
}

// Close provides a mock function with no fields
func (_m *MockTask) Close() {
	_m.Called()
}

// MockTask_Close_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Close'
type MockTask_Close_Call struct {
	*mock.Call
}

// Close is a helper method to define mock.On call
func (_e *MockTask_Expecter) Close() *MockTask_Close_Call {
	return &MockTask_Close_Call{Call: _e.mock.On("Close")}
}

func (_c *MockTask_Close_Call) Run(run func()) *MockTask_Close_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockTask_Close_Call) Return() *MockTask_Close_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockTask_Close_Call) RunAndReturn(run func()) *MockTask_Close_Call {
	_c.Run(run)
	return _c
}

// Complete provides a mock function with no fields
func (_m *MockTask) Complete() {
	_m.Called()
}

// MockTask_Complete_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Complete'
type MockTask_Complete_Call struct {
	*mock.Call
}

// Complete is a helper method to define mock.On call
func (_e *MockTask_Expecter) Complete() *MockTask_Complete_Call {
	return &MockTask_Complete_Call{Call: _e.mock.On("Complete")}
}

func (_c *MockTask_Complete_Call) Run(run func()) *MockTask_Complete_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockTask_Complete_Call) Return() *MockTask_Complete_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockTask_Complete_Call) RunAndReturn(run func()) *MockTask_Complete_Call {
	_c.Run(run)
	return _c
}

// Interrupt provides a mock function with no fields
func (_m *MockTask) Interrupt() {
	_m.Called()
}

// MockTask_Interrupt_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Interrupt'
type MockTask_Interrupt_Call struct {
	*mock.Call
}

// Interrupt is a helper method to define mock.On call
func (_e *MockTask_Expecter) Interrupt() *MockTask_Interrupt_Call {
	return &MockTask_Interrupt_Call{Call: _e.mock.On("Interrupt")}
}

func (_c *MockTask_Interrupt_Call) Run(run func()) *MockTask_Interrupt_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockTask_Interrupt_Call) Return() *MockTask_Interrupt_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockTask_Interrupt_Call) RunAndReturn(run func()) *MockTask_Interrupt_Call {
	_c.Run(run)
	return _c
}

// Run provides a mock function with no fields
func (_m *MockTask) Run() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Run")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockTask_Run_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Run'
type MockTask_Run_Call struct {
	*mock.Call
}

// Run is a helper method to define mock.On call
func (_e *MockTask_Expecter) Run() *MockTask_Run_Call {
	return &MockTask_Run_Call{Call: _e.mock.On("Run")}
}

func (_c *MockTask_Run_Call) Run(run func()) *MockTask_Run_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockTask_Run_Call) Return(_a0 error) *MockTask_Run_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockTask_Run_Call) RunAndReturn(run func() error) *MockTask_Run_Call {
	_c.Call.Return(run)
	return _c
}

// Validate provides a mock function with no fields
func (_m *MockTask) Validate() error {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Validate")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockTask_Validate_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Validate'
type MockTask_Validate_Call struct {
	*mock.Call
}

// Validate is a helper method to define mock.On call
func (_e *MockTask_Expecter) Validate() *MockTask_Validate_Call {
	return &MockTask_Validate_Call{Call: _e.mock.On("Validate")}
}

func (_c *MockTask_Validate_Call) Run(run func()) *MockTask_Validate_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockTask_Validate_Call) Return(_a0 error) *MockTask_Validate_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockTask_Validate_Call) RunAndReturn(run func() error) *MockTask_Validate_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockTask creates a new instance of MockTask. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockTask(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockTask {
	mock := &MockTask{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
