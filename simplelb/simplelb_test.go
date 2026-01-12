package simplelb

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/phuthien0308/ordering-base/simplelog"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"google.golang.org/grpc/resolver"
)

type MockAddressPuller struct {
	mock.Mock
}

func (m *MockAddressPuller) Pull(ctx context.Context, serviceName string) ([]Address, error) {
	args := m.Called(ctx, serviceName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]Address), args.Error(1)
}

func TestSimpleLB(t *testing.T) {
	logger := simplelog.NewSimpleZapLogger(zap.NewNop())
	mockAddressPuller := &MockAddressPuller{}
	simpleLbBuilder := NewSimpleLBBuilder(logger, mockAddressPuller, 0)
	_, err := simpleLbBuilder.Build(resolver.Target{URL: url.URL{Host: "test-service"}}, nil, resolver.BuildOptions{})
	assert.NoError(t, err)
}

type DynamicPuller struct {
	addresses []Address
}

func (p *DynamicPuller) Pull(ctx context.Context, serviceName string) ([]Address, error) {
	return p.addresses, nil
}

func TestSimpleLBRun(t *testing.T) {
	logger := simplelog.NewSimpleZapLogger(zap.NewNop())
	puller := &DynamicPuller{}
	simpleLbBuilder := NewSimpleLBBuilder(logger, puller, 10*time.Millisecond)
	mockClientConn := &MockClientConn{}
	address1 := []Address{"127.0.0.1:8080", "127.0.0.1:8081"}
	resolverAddresses1 := lo.Map(address1, func(ad Address, _ int) resolver.Address {
		return resolver.Address{Addr: string(ad)}
	})
	expectedState1 := resolver.State{
		Addresses: resolverAddresses1,
	}

	address2 := []Address{"127.0.0.1:8080", "127.0.0.1:8081", "127.0.0.1:8082"}
	resolverAddresses2 := lo.Map(address2, func(ad Address, _ int) resolver.Address {
		return resolver.Address{Addr: string(ad)}
	})
	expectedState2 := resolver.State{
		Addresses: resolverAddresses2,
	}

	puller.addresses = address1
	_, err := simpleLbBuilder.Build(resolver.Target{URL: url.URL{Host: "test-service"}}, mockClientConn, resolver.BuildOptions{})

	assert.NoError(t, err)

	mockClientConn.On("UpdateState", expectedState1).Return(nil).Once()
	mockClientConn.On("UpdateState", expectedState2).Return(nil).Once()

	// Wait for the first address to be processed
	time.Sleep(100 * time.Millisecond)

	// Update the addresses returned by the mock
	puller.addresses = address2

	// Wait for the second address to be processed
	time.Sleep(100 * time.Millisecond)

	mockClientConn.AssertExpectations(t)
}
