package http_handler

import (
	"context"
	"fmt"
	"log/slog"
	"time"
	"youtube_tracker/internal/helpers"
	
	"github.com/google/uuid"
	"github.com/ogen-go/ogen/middleware"
)

func NewRequestLogger(logger *slog.Logger) middleware.Middleware {
	requestLogger := logger.With(slog.String("component", "RequestLogger"))

	return func(req middleware.Request, next middleware.Next) (middleware.Response, error) {
		ctx := req.Context
		loggerWithRequestID := helpers.LoggerWithRequestID(ctx, requestLogger)
		
		start := time.Now()
		resp, err := next(req)
		duration := time.Since(start)

		if err != nil {
			loggerWithRequestID.Error("Request failed",
				slog.String("method", req.Raw.Method),
				slog.String("url", req.Raw.URL.Path),
				slog.String("operation", req.OperationName),
				slog.Int64("duration_ms", duration.Milliseconds()),
				slog.Any("error", err),
			)
			return resp, err
		}

		loggerWithRequestID.Info("request",
			slog.String("method", req.Raw.Method),
			slog.String("url", req.Raw.URL.Path),
			slog.String("operation", req.OperationName),
			slog.Int64("duration_ms", duration.Milliseconds()),
			slog.String("response_type", fmt.Sprintf("%T", resp.Type)),
		)

		return resp, nil
	}
}

func RequestIDGenerator(req middleware.Request, next middleware.Next) (middleware.Response, error) {
	ctx := req.Context
	requestID := uuid.New().String()
	req.Context = context.WithValue(ctx, helpers.RequestIDKey, requestID)

	return next(req)
}
