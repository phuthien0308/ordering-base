package simplelb

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/phuthien0308/ordering-base/simplelog"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"google.golang.org/grpc/resolver"
)

var watcherScheme = "simplelb_watcher"

func RegisterWatcher(logger *simplelog.SimpleZapLogger, watcher AddressWatcher, interval time.Duration) {
	resolver.Register(&SimpleLBWatcherBuilder{watcher: watcher, interval: interval, logger: logger})
}

type AddressWatcher interface {
	// Watch should return a channel that sends address updates
	Watch(ctx context.Context, serviceName string) (<-chan []Address, error)
}

type SimpleLBWatcherBuilder struct {
	watcher  AddressWatcher
	interval time.Duration
	logger   *simplelog.SimpleZapLogger
}

// Build implements [resolver.Builder].
func (s *SimpleLBWatcherBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	service := target.URL.Host
	if service == "" {
		s.logger.Error(context.TODO(), "empty service name")
		return nil, errors.New("empty service name in target")
	}

	ctx, cancelFunc := context.WithCancel(context.TODO())

	spResolver := &simpleLBWatcherResolver{
		serviceName: service,
		cc:          cc,
		mutex:       sync.Mutex{},
		doneCh:      make(chan interface{}),
		ctx:         ctx,
		cancelFun:   cancelFunc,
		watcher:     s.watcher,
		logger:      s.logger,
		interval:    s.interval,
	}
	go spResolver.run()
	return spResolver, nil
}

// Scheme implements [resolver.Builder].
func (s *SimpleLBWatcherBuilder) Scheme() string {
	return watcherScheme
}

type simpleLBWatcherResolver struct {
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
	watcher     AddressWatcher
	lastAddress []Address
	interval    time.Duration
	logger      *simplelog.SimpleZapLogger
}

// Close implements [resolver.Resolver].

func (s *simpleLBWatcherResolver) run() {

	defer func() {
		s.doneCh <- struct{}{}
	}()

	// Reconnection loop
	for {
		// Check context before trying to connecting
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		// Start watching for address changes
		addressCh, err := s.watcher.Watch(s.ctx, s.serviceName)
		if err != nil {
			s.logger.Error(s.ctx, "failed to start watching addresses", zap.Error(err))
			s.cc.ReportError(err)

			// Wait before retrying
			select {
			case <-s.ctx.Done():
				return
			case <-time.After(time.Second):
				continue
			}
		}

		// Consumption loop
	reconnectLoop:
		for {
			select {
			case <-s.ctx.Done():
				return
			case addresses, ok := <-addressCh:
				if !ok {
					// Channel closed, try to reconnect
					s.logger.Warn(s.ctx, "address channel closed, attempting to reconnect...")
					break reconnectLoop
				}

				diff1, diff2 := lo.Difference(s.lastAddress, addresses)
				if len(diff1) != 0 || len(diff2) != 0 {
					s.lastAddress = addresses // Update lastAddress
					resolverAddresses := lo.Map(addresses, func(ad Address, _ int) resolver.Address {
						return resolver.Address{Addr: string(ad)}
					})
					newState := resolver.State{Addresses: resolverAddresses}
					s.mutex.Lock()
					err = s.cc.UpdateState(newState)
					if err != nil {
						s.logger.Error(s.ctx, "can not update state", zap.Error(err))
						s.cc.ReportError(err)
					} else {
						s.logger.Info(s.ctx, "successfully update the state", zap.String("serviceName", s.serviceName),
							zap.Array("addresses", ArrayAddress(addresses)))
					}
					s.mutex.Unlock()
				}
			}
		}

		// Optional: wait a bit before reconnecting immediately to avoid tight loops if it fails instantly
		time.Sleep(time.Second)
	}

}

// Close is called when the connection is closed, we should stop calling the service registry.
func (s *simpleLBWatcherResolver) Close() {
	s.logger.Info(context.Background(), "connection is closing")
	s.cancelFun()
	<-s.doneCh
	s.logger.Info(context.Background(), "connection is closed")
}

// ResolveNow implements [resolver.Resolver].
func (s *simpleLBWatcherResolver) ResolveNow(opts resolver.ResolveNowOptions) {}
