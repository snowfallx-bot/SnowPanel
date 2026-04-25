package observability

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/snowfallx-bot/SnowPanel/backend/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

type ShutdownFunc func(context.Context) error

func InitTracing(ctx context.Context, cfg config.TracingConfig, appEnv string) (ShutdownFunc, error) {
	if !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}

	endpoint := strings.TrimSpace(cfg.OTLPEndpoint)
	if endpoint == "" {
		return func(context.Context) error { return nil }, fmt.Errorf("OTEL_EXPORTER_OTLP_ENDPOINT is required when OTEL_TRACING_ENABLED=true")
	}

	exporterOptions := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
	}
	if cfg.Insecure {
		exporterOptions = append(exporterOptions, otlptracegrpc.WithInsecure())
	}

	exportCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	exporter, err := otlptracegrpc.New(exportCtx, exporterOptions...)
	if err != nil {
		return nil, fmt.Errorf("create otlp trace exporter: %w", err)
	}

	serviceName := strings.TrimSpace(cfg.ServiceName)
	if serviceName == "" {
		serviceName = "snowpanel-backend"
	}

	resourceAttrs := []attribute.KeyValue{
		attribute.String("service.name", serviceName),
		attribute.String("deployment.environment.name", strings.TrimSpace(appEnv)),
	}
	if version := strings.TrimSpace(cfg.ServiceVersion); version != "" {
		resourceAttrs = append(resourceAttrs, attribute.String("service.version", version))
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.SampleRatio))),
		sdktrace.WithResource(resource.NewSchemaless(resourceAttrs...)),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return provider.Shutdown, nil
}
