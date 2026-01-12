package simplelb

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/phuthien0308/ordering-base/simplelog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"google.golang.org/grpc/resolver"
	"google.golang.org/grpc/serviceconfig"
)

// MockClientConn mocks the resolver.ClientConn interface
type MockClientConn struct {
	mock.Mock
}

func (m *MockClientConn) UpdateState(state resolver.State) error {
	// The implementation calls UpdateState.
	// We just record the call.
	args := m.Called(state)
	return args.Error(0)
}

func (m *MockClientConn) ReportError(err error) {
	m.Called(err)
}

func (m *MockClientConn) NewAddress(addresses []resolver.Address) {
	m.Called(addresses)
}

func (m *MockClientConn) NewServiceConfig(serviceConfig string) {
	m.Called(serviceConfig)
}

func (m *MockClientConn) ParseServiceConfig(serviceConfigJSON string) *serviceconfig.ParseResult {
	return nil
}

// MockAddressWatcher mocks the AddressWatcher interface
type MockAddressWatcher struct {
	mock.Mock
}

func (m *MockAddressWatcher) Watch(ctx context.Context, serviceName string) (<-chan []Address, error) {
	args := m.Called(ctx, serviceName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	// Correctly cast the return channel
	return args.Get(0).(chan []Address), args.Error(1)
}

func TestSimpleLBWatcherBuilder_Scheme(t *testing.T) {
	b := &SimpleLBWatcherBuilder{}
	assert.Equal(t, "simplelb_watcher", b.Scheme())
}

func TestSimpleLBWatcherBuilder_Build_Error(t *testing.T) {
	logger := simplelog.NewSimpleZapLogger(zap.NewNop())
	b := &SimpleLBWatcherBuilder{logger: logger}

	// Target with empty host
	target := resolver.Target{URL: url.URL{Host: ""}}

	res, err := b.Build(target, nil, resolver.BuildOptions{})
	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Equal(t, "empty service name in target", err.Error())
}

func TestSimpleLBWatcherBuilder_Build_Success_And_Run(t *testing.T) {
	// This test integrates Build and the running of the resolver logic
	logger := simplelog.NewSimpleZapLogger(zap.NewNop())
	mockWatcher := new(MockAddressWatcher)
	mockCC := new(MockClientConn)
	interval := time.Millisecond * 10

	b := &SimpleLBWatcherBuilder{
		watcher:  mockWatcher,
		interval: interval,
		logger:   logger,
	}

	serviceName := "test-service"
	target := resolver.Target{URL: url.URL{Host: serviceName}}

	// Channel to control address updates
	addrCh := make(chan []Address)

	// Setup Expectation for Watch
	mockWatcher.On("Watch", mock.Anything, serviceName).Return(addrCh, nil)

	// Build
	res, err := b.Build(target, mockCC, resolver.BuildOptions{})

	// Verify Build success
	assert.NoError(t, err)
	assert.NotNil(t, res)

	// Now we need to verify the loop behavior.
	// Since run() is called in a goroutine, we need to wait or rely on side effects (mock calls).

	// 1. Send update
	addresses := []Address{"1.2.3.4:80"}
	expectedState := resolver.State{
		Addresses: []resolver.Address{{Addr: "1.2.3.4:80"}},
	}

	// Expect UpdateState to be called
	mockCC.On("UpdateState", expectedState).Return(nil).Run(func(args mock.Arguments) {
		// Signal that call happened if needed, or just let assert verify later
	}).Once()

	addrCh <- addresses

	// Give some time for the goroutine to process
	time.Sleep(50 * time.Millisecond)

	// 2. Send same update (should filter diff if using lo.Difference properly)
	// simplelb_watcher.go logic: diff1, diff2 := lo.Difference(s.lastAddress, addresses)
	// if len(diff1) != 0 || len(diff2) != 0
	// Same address -> should NOT call UpdateState
	addrCh <- addresses
	time.Sleep(20 * time.Millisecond)

	// 3. Send new update
	newAddresses := []Address{"5.6.7.8:90"}
	expectedState2 := resolver.State{
		Addresses: []resolver.Address{{Addr: "5.6.7.8:90"}},
	}
	mockCC.On("UpdateState", expectedState2).Return(nil).Once()

	addrCh <- newAddresses
	time.Sleep(50 * time.Millisecond)

	// Close the resolver
	res.Close()
	// Verify all expectations
	mockCC.AssertExpectations(t)
	mockWatcher.AssertExpectations(t)
}

func TestSimpleLBWatcherResolver_WatchError(t *testing.T) {
	logger := simplelog.NewSimpleZapLogger(zap.NewNop())
	mockWatcher := new(MockAddressWatcher)
	mockCC := new(MockClientConn)
	interval := time.Millisecond * 10

	b := &SimpleLBWatcherBuilder{
		watcher:  mockWatcher,
		interval: interval,
		logger:   logger,
	}

	serviceName := "error-service"
	target := resolver.Target{URL: url.URL{Host: serviceName}}

	// Setup Expectation for Watch to fail FIRST time, then succeed?
	// The code retries every 1 second (hardcoded) + has consumption loop.
	// "Wait before retrying -> time.Sleep(time.Second)"
	// To test retry, we might need to wait > 1s, which is slow.
	// We can just test that it reports error on failure.

	mockWatcher.On("Watch", mock.Anything, serviceName).Return(nil, errors.New("watch failed")).Once()
	mockCC.On("ReportError", mock.MatchedBy(func(err error) bool {
		return err.Error() == "watch failed"
	})).Once()

	// We can't easily wait for the retry unless we patch time or wait 1s.
	// Let's just verify the initial error report.

	res, err := b.Build(target, mockCC, resolver.BuildOptions{})
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)
	res.Close()

	mockCC.AssertExpectations(t)
	mockWatcher.AssertExpectations(t)
}
