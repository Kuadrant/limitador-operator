/*
Copyright 2025 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package observability

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"
	"k8s.io/utils/env"
)

const (
	// Default values for OpenTelemetry configuration
	defaultServiceName = "limitador-operator"
	defaultEndpoint    = ""

	// Environment variable names
	envServiceName        = "OTEL_SERVICE_NAME"
	envOTLPEndpoint       = "OTEL_EXPORTER_OTLP_ENDPOINT"
	envOTLPInsecure       = "OTEL_EXPORTER_OTLP_INSECURE"
	envResourceAttributes = "OTEL_RESOURCE_ATTRIBUTES"
)

// Config holds the OpenTelemetry configuration
// Endpoint URL scheme determines the protocol:
//   - rpc://host:port  → gRPC OTLP
//   - http://host:port → HTTP OTLP (insecure)
//   - https://host:port → HTTP OTLP (secure)
//   - "" (empty)       → Tracing disabled (no-op)
type Config struct {
	ServiceName        string
	ServiceVersion     string
	Endpoint           string
	Insecure           bool
	ResourceAttributes map[string]string
}

// Provider holds the OpenTelemetry providers and cleanup function
type Provider struct {
	TracerProvider *trace.TracerProvider
	Shutdown       func(context.Context) error
}

// NewConfig creates a new Config from environment variables
func NewConfig(version string) *Config {
	serviceName := env.GetString(envServiceName, defaultServiceName)
	endpoint := env.GetString(envOTLPEndpoint, defaultEndpoint)
	insecure, _ := strconv.ParseBool(env.GetString(envOTLPInsecure, "false"))

	// Parse resource attributes (key=value,key=value format)
	resourceAttrs := make(map[string]string)
	if attrStr := env.GetString(envResourceAttributes, ""); attrStr != "" {
		for _, pair := range strings.Split(attrStr, ",") {
			if kv := strings.SplitN(pair, "=", 2); len(kv) == 2 {
				resourceAttrs[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}

	return &Config{
		ServiceName:        serviceName,
		ServiceVersion:     version,
		Endpoint:           endpoint,
		Insecure:           insecure,
		ResourceAttributes: resourceAttrs,
	}
}

// InitProvider initializes OpenTelemetry providers based on the configuration
// This focuses on distributed tracing only - metrics are handled separately by controller-runtime
func InitProvider(ctx context.Context, cfg *Config) (*Provider, error) {
	// Create resource
	res, err := newResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Initialize trace provider
	tracerProvider, traceShutdown, err := initTracerProvider(ctx, cfg, res)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracer provider: %w", err)
	}

	// Set global tracer provider
	otel.SetTracerProvider(tracerProvider)

	// Set global propagator for distributed tracing
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &Provider{
		TracerProvider: tracerProvider,
		Shutdown:       traceShutdown,
	}, nil
}

// newResource creates a resource with service information and custom attributes
func newResource(cfg *Config) (*resource.Resource, error) {
	// Start with service attributes
	serviceAttrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(cfg.ServiceName),
		semconv.ServiceVersionKey.String(cfg.ServiceVersion),
	}

	// Add custom resource attributes from config
	for k, v := range cfg.ResourceAttributes {
		serviceAttrs = append(serviceAttrs, attribute.String(k, v))
	}

	// Create resource with service attributes
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(serviceAttrs...),
	)
	if err != nil {
		return nil, err
	}

	// Merge with default resource
	return resource.Merge(
		resource.Default(),
		res,
	)
}

// initTracerProvider initializes the trace provider with the configured exporter
func initTracerProvider(ctx context.Context, cfg *Config, res *resource.Resource) (*trace.TracerProvider, func(context.Context) error, error) {
	// If endpoint is empty, create a no-op provider (tracing disabled)
	if cfg.Endpoint == "" {
		tp := trace.NewTracerProvider(trace.WithResource(res))
		return tp, func(ctx context.Context) error { return nil }, nil
	}

	// Create exporter based on endpoint URL
	exporter, err := newExporter(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Create tracer provider with batch span processor
	tp := trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithBatcher(exporter),
	)

	shutdown := func(ctx context.Context) error {
		if err := tp.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown tracer provider: %w", err)
		}
		return nil
	}

	return tp, shutdown, nil
}

// newExporter creates an OTLP trace exporter based on endpoint URL scheme
// Following the Authorino pattern:
//   - rpc://host:port  → gRPC exporter
//   - http://host:port → HTTP exporter (insecure)
//   - https://host:port → HTTP exporter (secure)
func newExporter(ctx context.Context, cfg *Config) (trace.SpanExporter, error) {
	u, err := url.Parse(cfg.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	var client otlptrace.Client

	switch u.Scheme {
	case "rpc":
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(u.Host),
		}
		if cfg.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		client = otlptracegrpc.NewClient(opts...)

	case "http", "https":
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(u.Host),
		}
		if path := u.Path; path != "" {
			opts = append(opts, otlptracehttp.WithURLPath(path))
		}
		if cfg.Insecure || u.Scheme == "http" {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		client = otlptracehttp.NewClient(opts...)

	default:
		return nil, fmt.Errorf("unsupported endpoint scheme: %s (use 'rpc', 'http', or 'https')", u.Scheme)
	}

	return otlptrace.New(ctx, client)
}
