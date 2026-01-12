package simplelog

import (
	"context"

	"github.com/phuthien0308/ordering-base/simplelog/tags"
	"go.uber.org/zap"
)

type SimpleLogKey struct{}

var SimpleLogKeyCtx = SimpleLogKey{}

type SimpleZapLogger struct {
	*zap.Logger
}

func NewSimpleZapLogger(logger *zap.Logger) *SimpleZapLogger {
	return &SimpleZapLogger{logger}
}

func (logger *SimpleZapLogger) withContext(ctx context.Context) *SimpleZapLogger {
	if fields, ok := ctx.Value(SimpleLogKeyCtx).([]zap.Field); ok {
		return &SimpleZapLogger{logger.With(fields...)}
	}
	return logger
}

func (logger *SimpleZapLogger) Debug(ctx context.Context, msg string, fields ...tags.T) {
	logger.withContext(ctx).Logger.Debug(msg, fields...)
}

func (logger *SimpleZapLogger) Info(ctx context.Context, msg string, fields ...tags.T) {
	logger.withContext(ctx).Logger.Info(msg, fields...)
}

func (logger *SimpleZapLogger) Warn(ctx context.Context, msg string, fields ...tags.T) {
	logger.withContext(ctx).Logger.Warn(msg, fields...)
}

func (logger *SimpleZapLogger) Error(ctx context.Context, msg string, fields ...tags.T) {
	logger.withContext(ctx).Logger.Error(msg, fields...)
}

func (logger *SimpleZapLogger) DPanic(ctx context.Context, msg string, fields ...tags.T) {
	logger.withContext(ctx).Logger.DPanic(msg, fields...)
}

func (logger *SimpleZapLogger) Panic(ctx context.Context, msg string, fields ...tags.T) {
	logger.withContext(ctx).Logger.Panic(msg, fields...)
}

func (logger *SimpleZapLogger) Fatal(ctx context.Context, msg string, fields ...tags.T) {
	logger.withContext(ctx).Logger.Fatal(msg, fields...)
}
