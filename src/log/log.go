package log

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/rs/zerolog"
)

func Logger(ctx context.Context) *zerolog.Logger {
	fileName := os.Getenv("LOG_FILE_NAME")
	if fileName == "" {
		fileName = "var/log/shoppinglist/item-service.log"
	}

	fileLogger := &lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    5, //
		MaxBackups: 10,
		MaxAge:     14,
		Compress:   false,
	}

	output := zerolog.MultiLevelWriter(os.Stderr, fileLogger)
	logger := zerolog.New(output).With().Timestamp().Caller().Str("app", os.Getenv("SERVICE_NAME")).Logger()

	logger = logger.Hook(zerologTraceHook(ctx))

	return &logger
}

// zerologTraceHook is a hook that;
// (a) adds TraceIds & spanIds to logs of all LogLevels
// (b) adds logs to the active span as events.
func zerologTraceHook(ctx context.Context) zerolog.HookFunc {
	return func(e *zerolog.Event, level zerolog.Level, message string) {
		if level == zerolog.NoLevel {
			return
		}
		if !e.Enabled() {
			return
		}

		if ctx == nil {
			return
		}

		span := trace.SpanFromContext(ctx)
		if !span.IsRecording() {
			return
		}

		{ // (a) adds TraceIds & spanIds to logs.
			//
			// TODO: (komuw) add stackTraces maybe.
			//
			sCtx := span.SpanContext()
			if sCtx.HasTraceID() {
				e.Str("traceId", sCtx.TraceID().String())
			}
			if sCtx.HasSpanID() {
				e.Str("spanId", sCtx.SpanID().String())
			}
		}

		{ // (b) adds logs to the active span as events.

			// code from: https://github.com/uptrace/opentelemetry-go-extra/tree/main/otellogrus
			// whose license(BSD 2-Clause) can be found at: https://github.com/uptrace/opentelemetry-go-extra/blob/v0.1.18/LICENSE

			// Unlike logrus or exp/slog, zerolog does not give hooks the ability to get the whole event/message with all its key-values
			// see: https://github.com/rs/zerolog/issues/300

			attrs := make([]attribute.KeyValue, 0)

			logSeverityKey := attribute.Key("log.severity")
			logMessageKey := attribute.Key("log.message")
			attrs = append(attrs, logSeverityKey.String(level.String()))
			attrs = append(attrs, logMessageKey.String(message))

			// todo: add caller info.

			span.AddEvent("log", trace.WithAttributes(attrs...))
			if level >= zerolog.ErrorLevel {
				span.SetStatus(codes.Error, message)
			}
		}
	}
}
