package simplelog

import (
	"context"

	"github.com/phuthien0308/ordering-base/simplelog/tags"
	"go.uber.org/zap"
)

type SimpleLogKey struct{}

var SimpleLogKeyCtx = SimpleLogKey{}

type SimpleLogger struct {
	*zap.Logger
}

var zapLogger, _ = zap.NewProduction(zap.AddCaller(),
	zap.AddStacktrace(zap.ErrorLevel))

var Logger = NewSimpleLogger(zapLogger)

func NewSimpleLogger(logger *zap.Logger) *SimpleLogger {
	return &SimpleLogger{logger}
}

func (logger *SimpleLogger) withContext(ctx context.Context) *SimpleLogger {
	if fields, ok := ctx.Value(SimpleLogKeyCtx).([]zap.Field); ok {
		return &SimpleLogger{logger.With(fields...)}
	}
	return logger
}

func (logger *SimpleLogger) Debug(ctx context.Context, msg string, fields ...tags.T) {
	logger.withContext(ctx).Logger.Debug(msg, fields...)
}

func (logger *SimpleLogger) Info(ctx context.Context, msg string, fields ...tags.T) {
	logger.withContext(ctx).Logger.Info(msg, fields...)
}

func (logger *SimpleLogger) Warn(ctx context.Context, msg string, fields ...tags.T) {
	logger.withContext(ctx).Logger.Warn(msg, fields...)
}

func (logger *SimpleLogger) Error(ctx context.Context, msg string, fields ...tags.T) {
	logger.withContext(ctx).Logger.Error(msg, fields...)
}

func (logger *SimpleLogger) DPanic(ctx context.Context, msg string, fields ...tags.T) {
	logger.withContext(ctx).Logger.DPanic(msg, fields...)
}

func (logger *SimpleLogger) Panic(ctx context.Context, msg string, fields ...tags.T) {
	logger.withContext(ctx).Logger.Panic(msg, fields...)
}

func (logger *SimpleLogger) Fatal(ctx context.Context, msg string, fields ...tags.T) {
	logger.withContext(ctx).Logger.Fatal(msg, fields...)
}
