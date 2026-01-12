package http_handler

import (
	"errors"
	"net/http"
	"strings"
)

// NewCORSHandler creates an HTTP handler to manage CORS, allowing requests from a specified origin or all origins ("*").
// Returns an error if the provided allowedOrigin is empty or contains only whitespace.
// Next handlers set Access-Control-Content-Type and Access-Control-Allow-Methods headers.
func NewCORSHandler(next http.Handler, allowedOrigin string) (http.Handler, error) {
	if len(strings.TrimSpace(allowedOrigin)) == 0 {
		return nil, errors.New("allowedOrigin is empty")
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		originalOrigin := r.Header.Get("Origin")
		if originalOrigin == "" {
			next.ServeHTTP(w, r)
			return
		}

		if allowedOrigin == "*" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		} else {
			if originalOrigin == allowedOrigin {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			}
		}

		next.ServeHTTP(w, r)
	}), nil
}
