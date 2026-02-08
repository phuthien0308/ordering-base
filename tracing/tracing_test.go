package tracing

import (
	"context"
	"testing"
)

func TestDefaultGlobalTracer_Init(t *testing.T) {
	// Initialize with a dummy URL (won't actually connect in unit test unless we mock,
	// but ensures code path runs and dependencies are correct)
	shutdown, err := DefaultGlobalTracer("test-service", "http://localhost:9411/api/v2/spans")
	if err != nil {
		t.Fatalf("DefaultGlobalTracer failed: %v", err)
	}
	if shutdown == nil {
		t.Fatal("Expected shutdown function, got nil")
	}

	// Clean up
	if err := shutdown(context.Background()); err != nil {
		// It might fail to send because localhost:9411 isn't running, but that's expected.
		// We just want to ensure it doesn't panic.
		t.Logf("Shutdown returned error (expected as zipkin not running): %v", err)
	}
}
