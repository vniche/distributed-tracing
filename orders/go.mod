module github.com/vniche/distributed-tracing/orders

go 1.16

require (
	github.com/google/uuid v1.2.0
	github.com/vniche/distributed-tracing/common v0.0.1-alpha
	github.com/vniche/distributed-tracing/tracing v0.0.1-alpha
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.20.0
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/trace v0.20.0
	google.golang.org/grpc v1.39.0
	google.golang.org/protobuf v1.27.1
)

// replacements for security vulnerabilities fixes found by aquasec's trivy
replace (
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.4.0
)
