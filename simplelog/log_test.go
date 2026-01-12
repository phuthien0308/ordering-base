package simplelog

import (
	"context"
	"testing"

	"github.com/phuthien0308/ordering-base/simplelog/tags"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestLogger(t *testing.T) {
	zapCore, observerLogs := observer.New(zap.InfoLevel)
	zapLogger := zap.New(zapCore)
	logger := NewSimpleZapLogger(zapLogger)
	ctx := context.WithValue(context.TODO(), SimpleLogKeyCtx, []tags.T{tags.String("trace_id", "1")})
	logger.Info(ctx, "hello world")
	result := false
	for _, entry := range observerLogs.All() {
		if _, ok := entry.ContextMap()["trace_id"]; ok {
			result = true
		}
	}
	if !result {
		t.Fail()
	}

}
