package otel

import (
	"context"
	"errors"
	"github.com/shoppinglist/log"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

// SetupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func SetupOTelSDK(ctx context.Context, otelCollectorHost string) (shutdown func(context.Context) error, err error) {
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
	tracerProvider, err := newTraceProvider(ctx, otelCollectorHost)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// Set up meter provider.
	meterProvider, err := newMeterProvider(ctx, otelCollectorHost)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	return
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTraceProvider(ctx context.Context, otelCollectorHost string) (*trace.TracerProvider, error) {
	useStdout := false
	var conn *grpc.ClientConn
	var err error
	if otelCollectorHost != "" {
		log.Logger().Info().Str("address", otelCollectorHost+":4317").Msg("connecting to trace collector")
		conn, err = grpc.DialContext(ctx, otelCollectorHost+":4317",
			// Note the use of insecure transport here. TLS is recommended in production.
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
		)
		if err != nil {
			log.Logger().Err(err).Msg("failed to create gRPC connection to collector")
			useStdout = true
		}
	} else {
		useStdout = true
	}

	// Set up a trace exporter
	var traceExporter trace.SpanExporter
	if !useStdout {
		traceExporter, err = otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
		if err != nil {
			log.Logger().Err(err).Msg("failed to create trace exporter")
			useStdout = true
		}
	}
	if traceExporter == nil {
		log.Logger().Info().Msg("falling back to stdout trace")
		traceExporter, err = stdouttrace.New(
			stdouttrace.WithPrettyPrint())
		if err != nil {
			log.Logger().Err(err).Msg("failed to create stdout trace exporter")
			return nil, err
		}
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
	)
	return traceProvider, nil
}

func newMeterProvider(ctx context.Context, otelCollectorHost string) (*metric.MeterProvider, error) {
	useStdout := false

	var metricExporter metric.Exporter
	var err error
	if otelCollectorHost != "" {
		log.Logger().Info().Str("address", otelCollectorHost+":4317").Msg("connecting to metric collector")
		metricExporter, err = otlpmetricgrpc.New(ctx,
			otlpmetricgrpc.WithInsecure(),
			otlpmetricgrpc.WithEndpoint(otelCollectorHost+":4317"),
		)
		if err != nil {
			log.Logger().Err(err).Msg("failed to create metric exporter")
			useStdout = true
		}
	} else {
		useStdout = true
	}
	if useStdout == true || metricExporter == nil {
		log.Logger().Info().Msg("falling back to stdout metric")
		metricExporter, err = stdoutmetric.New()
		if err != nil {
			log.Logger().Err(err).Msg("failed to create stdout metric exporter")
			return nil, err
		}
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			// Default is 1m. Set to 3s for demonstrative purposes.
			metric.WithInterval(3*time.Second))),
	)
	return meterProvider, nil
}
