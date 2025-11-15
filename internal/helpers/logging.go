package helpers

import (
	"context"
	"log/slog"
)

type ContextKey string
const LoggerKey = ContextKey("logger")
const RequestIDKey = ContextKey("requestID")

func LoggerFromContext(ctx context.Context, componentName string, defaultLogger *slog.Logger) *slog.Logger {
	if logger, ok := ctx.Value(LoggerKey).(*slog.Logger); ok {
		return logger.With(slog.String("component", componentName))
	}
	
	return defaultLogger.With(slog.String("component", componentName))
}

func RequestIdFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	
	return "undefined"
}

func LoggerWithRequestID(ctx context.Context, logger *slog.Logger) *slog.Logger {
	return logger.With(slog.String("requestID", RequestIdFromContext(ctx)))
}

func CreateBackgroundContext(ctx context.Context, logger *slog.Logger) context.Context {
	requestID := RequestIdFromContext(ctx)
	loggerWithRequestID := logger.With(slog.String("request_id", requestID))
	backgroundCtx := context.WithValue(
		context.WithValue(context.Background(), RequestIDKey, requestID), LoggerKey, loggerWithRequestID)
	return backgroundCtx
}
