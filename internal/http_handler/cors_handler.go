package http_handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
)

// NewCORSHandler creates an HTTP handler to manage CORS, allowing requests from a specified origin or all origins ("*").
// allowedOrigin can be a single origin or a comma-separated list of origins.
// Returns an error if the provided allowedOrigin is empty or contains only whitespace.
// Next handlers set Access-Control-Content-Type and Access-Control-Allow-Methods headers.
func NewCORSHandler(next http.Handler, allowedOrigin string, logger *slog.Logger) (http.Handler, error) {
	if len(strings.TrimSpace(allowedOrigin)) == 0 {
		return nil, errors.New("allowedOrigin is empty")
	}

	allowedOrigins := strings.Split(allowedOrigin, ",")
	for i, o := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(o)
		if len(allowedOrigins[i]) == 0 {
			return nil, errors.New("allowedOrigin contains empty entry")
		}
	}
	if len(allowedOrigins) > 1 {
		for _, o := range allowedOrigins {
			if o == "*" {
				return nil, errors.New("wildcard '*' cannot be mixed with other origins")
			}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originalOrigin := r.Header.Get("Origin")
		if originalOrigin == "" {
			next.ServeHTTP(w, r)
			return
		}

		if allowedOrigins[0] == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else {
			allowed := false
			for _, o := range allowedOrigins {
				if originalOrigin == o {
					w.Header().Set("Access-Control-Allow-Origin", originalOrigin)
					allowed = true
					break
				}
			}
			if !allowed {
				logger.Warn("CORS: origin not allowed", "origin", originalOrigin)
			}
		}

		next.ServeHTTP(w, r)
	}), nil
}
