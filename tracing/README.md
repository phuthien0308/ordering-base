# Tracing Module

This module provides a unified way to initialize distributed tracing for the `ordering-base` project, using OpenTelemetry and Zipkin.

## Initialization

You must initialize the global tracer **once** at the startup of your application (e.g., in `main.go`).

### Quick Start (Zipkin)

Use `DefaultGlobalTracer` for a pre-configured Zipkin setup.

```go
package main

import (
	"context"
	"log"

	"github.com/phuthien0308/ordering-base/tracing"
)

func main() {
	// 1. Initialize
	// Zipkin URL example: "http://localhost:9411/api/v2/spans"
	shutdown, err := tracing.DefaultGlobalTracer("my-service-name", "http://localhost:9411/api/v2/spans")
	if err != nil {
		log.Fatalf("failed to init tracing: %v", err)
	}
	
	// 2. Schedule Shutdown
	// This flushes any buffered traces before the app exits.
	defer shutdown(context.Background())

	// 3. Start your app...
}
```

### Custom Setup (Advanced)

Use `InitGlobalTracer` if you want to bring your own Exporter (e.g., Jaeger, Stdout) or custom Resource.

```go
import (
	"github.com/phuthien0308/ordering-base/tracing"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
)

func main() {
	exporter, _ := stdouttrace.New(stdouttrace.WithPrettyPrint())
	res, _ := resource.New(context.Background()) // ... attributes

	shutdown, _ := tracing.InitGlobalTracer(exporter, res)
	defer shutdown(context.Background())
}
```

## Instrumentation Guide

Once initialized, you need to instrument your HTTP or gRPC handlers to automatically generate spans and propagate context.

### HTTP Instrumentation

Using `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp`:

**Server (Middleware):**
```go
import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

func main() {
    handler := http.HandlerFunc(myHandler)
    
    // Wrap your handler
    wrappedHandler := otelhttp.NewHandler(handler, "operation-name")
    
    http.ListenAndServe(":8080", wrappedHandler)
}
```

**Client:**
```go
import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

func callService() {
    // Use an instrumented HTTP client
    client := http.Client{
        Transport: otelhttp.NewTransport(http.DefaultTransport),
    }

    // Ensure 'ctx' contains the parent trace info if available
    req, _ := http.NewRequestWithContext(ctx, "GET", "http://...", nil)
    
    client.Do(req)
}
```

### gRPC Instrumentation

Using `go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc`:

**Server (Interceptor):**
```go
import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

func main() {
    s := grpc.NewServer(
        grpc.StatsHandler(otelgrpc.NewServerHandler()),
    )
    // ... register services ...
}
```

**Client (Interceptor):**
```go
import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

func main() {
    conn, err := grpc.Dial(
        "address", 
        grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
    )
    // ...
}
```

## Manual Instrumentation

For internal logic that isn't HTTP/gRPC, use the standard OTEL API:

```go
import "go.opentelemetry.io/otel"

func internalWork(ctx context.Context) {
    tracer := otel.Tracer("my-package")
    
    ctx, span := tracer.Start(ctx, "work-operation")
    defer span.End()

    // Do work...
}
```
