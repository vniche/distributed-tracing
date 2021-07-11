package tracing

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/semconv"

	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func Tracer(name string) trace.Tracer {
	return otel.Tracer(name)
}

// tracerProvider returns an OpenTelemetry TracerProvider configured to use
// the Jaeger exporter that will send spans to the provided url. The returned
// TracerProvider will also use a Resource configured with all the information
// about the application.
func tracerProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	// Create an OTLP exporter, passing in Honeycomb credentials as environment variables.
	exp, err := otlp.NewExporter(
		ctx,
		// Set up a trace exporter
		otlpgrpc.NewDriver(
			otlpgrpc.WithEndpoint("api.honeycomb.io:443"),
			otlpgrpc.WithDialOption(grpc.WithBlock()),
			otlpgrpc.WithHeaders(map[string]string{
				"x-honeycomb-team":    os.Getenv("HONEYCOMB_API_KEY"),
				"x-honeycomb-dataset": os.Getenv("HONEYCOMB_DATASET"),
			}),
			otlpgrpc.WithTLSCredentials(credentials.NewClientTLSFromCert(nil, "")),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize exporter: %v", err)
	}

	options := []sdktrace.TracerProviderOption{
		sdktrace.WithSpanProcessor(
			sdktrace.NewBatchSpanProcessor(exp),
		),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.ServiceNameKey.String(os.Getenv("SERVICE_NAME")),
			semconv.DeploymentEnvironmentKey.String(os.Getenv("ENVIRONMENT")),
		)),
	}
	if os.Getenv("DEBUG_TRACING") != "true" {
		options = append(options, sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.0001)))
	}

	return sdktrace.NewTracerProvider(options...), nil
}

func InitTracerProvider(ctx context.Context) (*sdktrace.TracerProvider, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var (
		tp  *sdktrace.TracerProvider
		err error
	)
	tp, err = tracerProvider(ctx)
	if err != nil {
		return nil, err
	}

	// Register our TracerProvider as the global so any imported
	// instrumentation in the future will default to using it.
	otel.SetTracerProvider(tp)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	return tp, nil
}
