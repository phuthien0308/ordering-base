package simplelb

import (
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/phuthien0308/ordering-base/simplelog"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/resolver"
)

func Register(logger *simplelog.SimpleZapLogger, puller AddressPuller, interval time.Duration) {
	resolver.Register(&SimpleLBBuilder{puller: puller, interval: interval, logger: logger})
}

func NewSimpleLBBuilder(logger *simplelog.SimpleZapLogger, puller AddressPuller, interval time.Duration) *SimpleLBBuilder {
	return &SimpleLBBuilder{puller: puller, interval: interval, logger: logger}
}

type Address string

type ArrayAddress []Address

func (a ArrayAddress) MarshalLogArray(enc zapcore.ArrayEncoder) error {
	for _, v := range a {
		enc.AppendString(string(v))
	}
	return nil
}

type AddressPuller interface {
	// the pull should use a backoff timer to retry.
	Pull(ctx context.Context, serviceName string) ([]Address, error)
}

var scheme = "simplelb"

type SimpleLBBuilder struct {
	puller   AddressPuller
	interval time.Duration
	logger   *simplelog.SimpleZapLogger
}

// Build implements [resolver.Builder].
func (s *SimpleLBBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	service := target.URL.Host
	if service == "" {
		s.logger.Error(context.TODO(), "empty service name")
		return nil, errors.New("empty service name in target")
	}

	ctx, cancelFunc := context.WithCancel(context.TODO())

	spResolver := &simpleLBResolver{
		serviceName: service,
		cc:          cc,
		mutex:       sync.Mutex{},
		doneCh:      make(chan interface{}),
		ctx:         ctx,
		cancelFun:   cancelFunc,
		puller:      s.puller,
		logger:      s.logger,
		interval:    s.interval,
	}
	go spResolver.run()
	return spResolver, nil
}

// Scheme implements [resolver.Builder].
func (s *SimpleLBBuilder) Scheme() string {
	return scheme
}

type simpleLBResolver struct {
	// clientConnection
	cc resolver.ClientConn
	// service name
	serviceName string
	//make sure only one caller can update the conn's state at a time.
	mutex sync.Mutex
	// notify when the caller calls the Close method
	doneCh    chan interface{}
	cancelFun context.CancelFunc
	ctx       context.Context
	// watch the changes
	puller      AddressPuller
	lastAddress []Address
	interval    time.Duration
	logger      *simplelog.SimpleZapLogger
}

// Close implements [resolver.Resolver].

func (s *simpleLBResolver) run() {

	defer func() {
		s.doneCh <- struct{}{}
	}()

	// Initial pull
	updateAddresses := func() {
		// if watch has error, it should retry with backoff timer
		addresses, err := retryWithJitter(s.ctx, 3, time.Second, s.serviceName, s.puller)
		if err != nil {
			s.logger.Error(s.ctx, "can not pull addresses", zap.Error(err))
			s.cc.ReportError(err)
			return
		}
		diff1, diff2 := lo.Difference(s.lastAddress, addresses)
		if len(diff1) != 0 || len(diff2) != 0 {
			s.lastAddress = addresses
			resolverAddresses := lo.Map(addresses, func(ad Address, _ int) resolver.Address {
				return resolver.Address{Addr: string(ad)}
			})
			s.mutex.Lock()
			newState := resolver.State{Addresses: resolverAddresses}
			err = s.cc.UpdateState(newState)
			if err != nil {
				s.logger.Error(s.ctx, "can not update state", zap.Error(err))
				s.cc.ReportError(err)
			}
			s.logger.Info(s.ctx, "successfully update the state", zap.String("serviceName", s.serviceName),
				zap.Array("addresses", ArrayAddress(addresses)))
			s.mutex.Unlock()
		}
	}

	updateAddresses()

	ticker := time.NewTicker(s.interval)
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			updateAddresses()
		}
	}
}

// Close is called when the connection is closed, we should stop calling the service registry.
func (s *simpleLBResolver) Close() {
	s.logger.Info(context.Background(), "connection is closing")
	s.cancelFun()
	<-s.doneCh
	s.logger.Info(context.Background(), "connection is closed")
}

// ResolveNow implements [resolver.Resolver].
func (s *simpleLBResolver) ResolveNow(opts resolver.ResolveNowOptions) {}

func retryWithJitter(ctx context.Context, attempts int, initial time.Duration, serviceName string, puller AddressPuller) ([]Address, error) {

	backoff := initial
	for i := 0; i < attempts; i++ {
		addresses, err := puller.Pull(ctx, serviceName)
		if err == nil {
			return addresses, nil
		}
		jitter := rand.Int63n(int64(backoff / 2))
		wait := backoff + time.Duration(jitter)
		select {
		case <-ctx.Done():
			return []Address{}, ctx.Err()
		case <-time.After(wait):
			backoff *= 2
		}
	}
	return []Address{}, errors.New("still got error after retrying")
}
