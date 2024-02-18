package otel

import (
	"context"
	"errors"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"time"

	"github.com/shoppinglist/config"
	"github.com/shoppinglist/log"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"

	metric2 "go.opentelemetry.io/otel/metric"
	trace2 "go.opentelemetry.io/otel/trace"
)

// SetupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func SetupOTelSDK(ctx context.Context, config *config.Config,
) (tracer trace2.Tracer, meter metric2.Meter, shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// Set up trace provider.
	tracerProvider, err := newTraceProvider(ctx, config)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// Set up meter provider.
	meterProvider, err := newMeterProvider(ctx, config)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	tracer = otel.GetTracerProvider().Tracer(config.ServiceName)
	meter = otel.Meter(config.ServiceName)

	return
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTraceProvider(ctx context.Context, cfg *config.Config) (*trace.TracerProvider, error) {
	var conn *grpc.ClientConn
	var err error
	if cfg.OtelCollectorHost != "" {
		log.Logger(ctx).Info().Str("address", cfg.OtelCollectorHost+":4317").Msg("connecting to trace collector")
		conn, err = grpc.DialContext(ctx, cfg.OtelCollectorHost+":4317",
			// Note the use of insecure transport here. TLS is recommended in production.
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			log.Logger(ctx).Err(err).Msg("failed to create gRPC connection to collector")
			cfg.UseStdout = true
		}
	} else {
		cfg.UseStdout = true
	}

	// Set up a trace exporter
	var traceExporter trace.SpanExporter
	if !cfg.UseStdout {
		traceExporter, err = otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
		if err != nil {
			log.Logger(ctx).Err(err).Msg("failed to create trace exporter")
			cfg.UseStdout = true
		}
	}
	if traceExporter == nil {
		log.Logger(ctx).Info().Msg("falling back to stdout trace")
		traceExporter, err = stdouttrace.New(
			stdouttrace.WithPrettyPrint())
		if err != nil {
			log.Logger(ctx).Err(err).Msg("failed to create stdout trace exporter")
			return nil, err
		}
	}

	r, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(config.Get(ctx).ServiceName)),
		resource.WithAttributes(semconv.ServiceVersion(config.Get(ctx).ServiceVersion)),
	)
	if err != nil {
		log.Logger(ctx).Err(err).Msg("resource.New")
		return nil, err
	}
	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(r),
	)
	return traceProvider, nil
}

func newMeterProvider(ctx context.Context, cfg *config.Config) (*metric.MeterProvider, error) {
	var metricExporter metric.Exporter
	var err error
	if cfg.OtelCollectorHost != "" {
		log.Logger(ctx).Info().Str("address", cfg.OtelCollectorHost+":4317").Msg("connecting to metric collector")
		metricExporter, err = otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithInsecure(),
			otlpmetricgrpc.WithEndpoint(cfg.OtelCollectorHost+":4317"),
		)
		if err != nil {
			log.Logger(ctx).Err(err).Msg("failed to create metric exporter")
			cfg.UseStdout = true
		}
	} else {
		cfg.UseStdout = true
	}
	if cfg.UseStdout == true || metricExporter == nil {
		log.Logger(ctx).Info().Msg("falling back to stdout metric")
		metricExporter, err = stdoutmetric.New()
		if err != nil {
			log.Logger(ctx).Err(err).Msg("failed to create stdout metric exporter")
			return nil, err
		}
	}

	r, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(config.Get(ctx).ServiceName)),
		resource.WithAttributes(semconv.ServiceVersion(config.Get(ctx).ServiceVersion)),
	)
	if err != nil {
		log.Logger(ctx).Err(err).Msg("resource.New")
		return nil, err
	}
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			// Default is 1m. Set to 3s for demonstrative purposes.
			metric.WithInterval(3*time.Second))),
		metric.WithResource(r),
	)
	return meterProvider, nil
}
