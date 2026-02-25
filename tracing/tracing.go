package tracing

import (
	"context"
	"paymentfc/config"
	"paymentfc/log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer trace.Tracer

func InitTracer(cfg config.TracingConfig) (func(context.Context) error, error) {
	if !cfg.Enabled {
		log.Logger.Info().Msg("Tracing is disabled")
		return func(ctx context.Context) error { return nil }, nil
	}

	ctx := context.Background()

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(cfg.Endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to create OTLP trace exporter")
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			attribute.String("environment", "development"),
		),
	)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to create resource")
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer = tp.Tracer(cfg.ServiceName)

	log.Logger.Info().Str("endpoint", cfg.Endpoint).Str("service", cfg.ServiceName).Msg("Tracing initialized")

	return tp.Shutdown, nil
}

func GetTracer() trace.Tracer {
	return tracer
}

func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return tracer.Start(ctx, name, opts...)
}
