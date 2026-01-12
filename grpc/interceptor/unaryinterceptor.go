package interceptor

import (
	"context"
	"time"

	"github.com/phuthien0308/ordering-base/simplelog"
	"github.com/phuthien0308/ordering-base/simplelog/tags"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func RequestInterceptor(logger *simplelog.SimpleZapLogger, env string) grpc.UnaryServerInterceptor {

	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()

		fields := []tags.T{tags.String(
			"rpc.system", "grpc"),
			tags.String("grpc.method", info.FullMethod),
			tags.String("service", "config"),
			tags.String("environment", env),
		}

		var requestID string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if v := md.Get("x-request-id"); len(v) > 0 {
				requestID = v[0]
			}
		}
		if requestID != "" {
			fields = append(fields, tags.String("request-id", requestID))
		}
		logger.Info(ctx, "grpc request", fields...)
		ctx = context.WithValue(ctx, simplelog.SimpleLogKeyCtx, fields)
		resp, err := handler(ctx, req)
		duration := time.Since(start)
		code := status.Convert(err)
		fields = append(fields, tags.String("grpc.code", code.String()), tags.Duration("duration", duration))

		if err == nil {
			logger.Info(ctx,
				"grpc request succeeded",
				append(fields, zap.Error(err))...,
			)
		} else {
			logger.Error(ctx,
				"grpc request failed",
				append(fields, zap.Error(err))...,
			)
		}
		return resp, err
	}
}
