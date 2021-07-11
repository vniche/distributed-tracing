module github.com/vniche/distributed-tracing/tracing

go 1.16

require (
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/exporters/otlp v0.20.0
	go.opentelemetry.io/otel/sdk v0.20.0
	go.opentelemetry.io/otel/trace v0.20.0
	google.golang.org/grpc v1.39.0
)

// replacements for security vulnerabilities fixes found by aquasec's trivy
replace (
	golang.org/x/crypto => golang.org/x/crypto v0.0.0-20210711020723-a769d52b0f97
	gopkg.in/yaml.v2 => gopkg.in/yaml.v2 v2.4.0
)
