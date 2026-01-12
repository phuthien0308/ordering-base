module github.com/phuthien0308/ordering-base/ratelimiter

go 1.25.5

require (
	github.com/go-redis/redis/v8 v8.11.5
	github.com/go-redis/redismock/v8 v8.11.5
	github.com/phuthien0308/ordering-base/simplelog v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.27.1
)

require (
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	go.uber.org/multierr v1.11.0 // indirect
)

replace github.com/phuthien0308/ordering-base/simplelog => ../simplelog
