package http_handler

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

var discardLogger = slog.New(slog.NewTextHandler(io.Discard, nil))

type httpHandlerMock struct {
	isCalled bool
}

func (h *httpHandlerMock) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.isCalled = true
	w.WriteHeader(http.StatusOK)
}

func TestCorsHandlerCallNext(t *testing.T) {
	nextHandler := httpHandlerMock{}

	responseWriter := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "someOrigin")

	corsHandler, _ := NewCORSHandler(&nextHandler, "*", discardLogger)
	corsHandler.ServeHTTP(responseWriter, req)

	assert.True(t, nextHandler.isCalled)
	assert.Equal(t, "*", responseWriter.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, responseWriter.Code)
}

func TestCorsHandlerShouldCallNextWhenOptions(t *testing.T) {
	nextHandler := httpHandlerMock{}

	responseWriter := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "someOrigin")

	corsHandler, _ := NewCORSHandler(&nextHandler, "someOrigin", discardLogger)
	corsHandler.ServeHTTP(responseWriter, req)

	assert.True(t, nextHandler.isCalled)
	assert.Equal(t, "someOrigin", responseWriter.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, responseWriter.Code)
}

func TestCorsHandlerShouldAllowOrigin(t *testing.T) {
	testCases := []struct {
		name        string
		allowOrigin string
	}{
		{name: "one origin", allowOrigin: "someOrigin"},
		{name: "list of origins - matching last", allowOrigin: "allowOrigin, someOrigin"},
		{name: "list of origins - matching first", allowOrigin: "someOrigin, allowOrigin"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nextHandler := httpHandlerMock{}

			responseWriter := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", "someOrigin")

			corsHandler, _ := NewCORSHandler(&nextHandler, tc.allowOrigin, discardLogger)
			corsHandler.ServeHTTP(responseWriter, req)

			assert.True(t, nextHandler.isCalled)
			assert.Equal(t, "someOrigin", responseWriter.Header().Get("Access-Control-Allow-Origin"))
		})
	}
}

func TestCorsHandlerShouldAllowOriginWhenOriginIsCommaList(t *testing.T) {
	nextHandler := httpHandlerMock{}

	responseWriter := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "someOrigin")

	corsHandler, _ := NewCORSHandler(&nextHandler, "allowOrigin, someOrigin", discardLogger)
	corsHandler.ServeHTTP(responseWriter, req)

	assert.True(t, nextHandler.isCalled)
	assert.Equal(t, "someOrigin", responseWriter.Header().Get("Access-Control-Allow-Origin"))
}

func TestCorsHandlerShouldNotAllowOrigin(t *testing.T) {
	testCases := []struct {
		name        string
		allowOrigin string
	}{
		{name: "one origin", allowOrigin: "anotherOrigin"},
		{name: "list of origins", allowOrigin: "anotherOrigin, alosAnotherOrigin"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nextHandler := httpHandlerMock{}

			responseWriter := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set("Origin", "someOrigin")

			corsHandler, _ := NewCORSHandler(&nextHandler, tc.allowOrigin, discardLogger)
			corsHandler.ServeHTTP(responseWriter, req)

			assert.True(t, nextHandler.isCalled)
			assert.Empty(t, responseWriter.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(t, http.StatusOK, responseWriter.Code)
		})
	}
}

func TestCorsHandlerShouldAllowRequestWithoutOrigin(t *testing.T) {
	nextHandler := httpHandlerMock{}

	responseWriter := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	corsHandler, _ := NewCORSHandler(&nextHandler, "someOrigin", discardLogger)
	corsHandler.ServeHTTP(responseWriter, req)

	assert.True(t, nextHandler.isCalled)
	assert.Empty(t, responseWriter.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, responseWriter.Code)
}

func TestNewCORSHandlerShouldRejectInvalidAllowOrigin(t *testing.T) {
	testCases := []struct {
		name        string
		allowOrigin string
	}{
		{name: "empty string", allowOrigin: ""},
		{name: "single space", allowOrigin: " "},
		{name: "multiple spaces", allowOrigin: "   "},
		{name: "tab character", allowOrigin: "\t"},
		{name: "newline character", allowOrigin: "\n"},
		{name: "mixed whitespace", allowOrigin: " \t\n "},
		{name: "empty strings in list", allowOrigin: "origin1,,origin2"},
		{name: "trailing comma", allowOrigin: "origin1,"},
		{name: "leading comma", allowOrigin: ",origin2"},
		{name: "white space list entries", allowOrigin: "origin1, , origin2"},
		{name: "wildcard mixed", allowOrigin: "*, https://example.com"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nextHandler := &httpHandlerMock{}

			handler, err := NewCORSHandler(nextHandler, tc.allowOrigin, discardLogger)

			assert.Nil(t, handler)
			assert.Error(t, err)
		})
	}
}
