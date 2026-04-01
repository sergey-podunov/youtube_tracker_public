package http_handler

import (
	"errors"
	"net/http"
	"strings"
)

// NewCORSHandler creates an HTTP handler to manage CORS, allowing requests from a specified origin or all origins ("*").
// allowedOrigin can be a single origin or a comma-separated list of origins.
// Returns an error if the provided allowedOrigin is empty or contains only whitespace.
// Next handlers set Access-Control-Content-Type and Access-Control-Allow-Methods headers.
func NewCORSHandler(next http.Handler, allowedOrigin string) (http.Handler, error) {
	if len(strings.TrimSpace(allowedOrigin)) == 0 {
		return nil, errors.New("allowedOrigin is empty")
	}

	allowedOrigins := strings.Split(allowedOrigin, ",")
	for i, o := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(o)
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
			for _, o := range allowedOrigins {
				if originalOrigin == o {
					w.Header().Set("Access-Control-Allow-Origin", originalOrigin)
					break
				}
			}
		}

		next.ServeHTTP(w, r)
	}), nil
}
