package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

func NewTracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return otel.Tracer(name, opts...)
}

func NewSpan(tracer trace.Tracer, ctx context.Context, spanName string) (context.Context, trace.Span) {
	return tracer.Start(ctx, spanName)
}

// InitGlobalTracer wires up the global OTEL provider with the given dependencies.
// Use this if you want to bring your own Exporter (Jaeger/Stdout) or custom Resource.
func InitGlobalTracer(exporter sdktrace.SpanExporter, res *resource.Resource) (func(context.Context) error, error) {
	// 1. Create the TracerProvider with the injected Exporter and Resource
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// 2. Set Globals
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// DefaultGlobalTracer sets up the tracer with Zipkin and default resource detection.
// It simplifies identifying the service via a string name.
func DefaultGlobalTracer(serviceName string, exporter sdktrace.SpanExporter) (func(context.Context) error, error) {

	// 2. Default Resource: Service Name + OS/Container info
	res, err := resource.New(context.Background(),
		resource.WithAttributes(semconv.ServiceNameKey.String(serviceName)),
		resource.WithFromEnv(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithContainer(),
		resource.WithHost(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 3. Delegate to Core
	return InitGlobalTracer(exporter, res)
}

type NoopExporter struct{}

func (noop *NoopExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	return nil
}

func (noop *NoopExporter) Shutdown(ctx context.Context) error {
	return nil
}
