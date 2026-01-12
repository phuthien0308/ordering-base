module github.com/phuthien0308/ordering-base/grpc

go 1.25.5

replace github.com/phuthien0308/ordering-base/simplelog => ../simplelog

require (
	github.com/phuthien0308/ordering-base/simplelog v0.0.1
	go.uber.org/zap v1.27.1
	google.golang.org/grpc v1.78.0
)

require (
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/net v0.47.0 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251029180050-ab9386a59fda // indirect
	google.golang.org/protobuf v1.36.10 // indirect
)
