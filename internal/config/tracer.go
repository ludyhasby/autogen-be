package config

import (
	"context"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"time"
)

func NewTracer(env *Env) (func(context.Context) error, error) {

	exp, err := otlptracehttp.New(
		context.Background(),
		otlptracehttp.WithEndpoint(env.OTelEndpoint),
		otlptracehttp.WithInsecure(),
		otlptracehttp.WithURLPath("/v1/traces"),
	)
	if err != nil {
		return nil, err
	}

	res := resource.NewSchemaless(
		semconv.ServiceNameKey.String(env.AppName),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(1.0))),
		sdktrace.WithBatcher(
			exp,
			sdktrace.WithMaxQueueSize(4096),
			sdktrace.WithMaxExportBatchSize(512),
			sdktrace.WithExportTimeout(2*time.Second),
			sdktrace.WithBatchTimeout(200*time.Millisecond),
		),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{},
		),
	)

	return tp.Shutdown, nil
}
