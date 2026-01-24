package http_handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	
	corsHandler, _ := NewCORSHandler(&nextHandler, "*")
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

	corsHandler, _ := NewCORSHandler(&nextHandler, "someOrigin")
	corsHandler.ServeHTTP(responseWriter, req)

	assert.True(t, nextHandler.isCalled)
	assert.Equal(t, "someOrigin", responseWriter.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, responseWriter.Code)
}

func TestCorsHandlerShouldAllowOrigin(t *testing.T) {
	nextHandler := httpHandlerMock{}

	responseWriter := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "someOrigin")

	corsHandler, _ := NewCORSHandler(&nextHandler, "someOrigin")
	corsHandler.ServeHTTP(responseWriter, req)
	
	assert.True(t, nextHandler.isCalled)
	assert.Equal(t, "someOrigin", responseWriter.Header().Get("Access-Control-Allow-Origin"))
}

func TestCorsHandlerShouldNotAllowOrigin(t *testing.T) {
	nextHandler := httpHandlerMock{}

	responseWriter := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "someOrigin")

	corsHandler, _ := NewCORSHandler(&nextHandler, "anotherOrigin")
	corsHandler.ServeHTTP(responseWriter, req)

	assert.True(t, nextHandler.isCalled)
	assert.Empty(t, responseWriter.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, http.StatusOK, responseWriter.Code)
}

func TestCorsHandlerShouldAllowRequestWithoutOrigin(t *testing.T) {
	nextHandler := httpHandlerMock{}

	responseWriter := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	corsHandler, _ := NewCORSHandler(&nextHandler, "someOrigin")
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nextHandler := &httpHandlerMock{}

			handler, err := NewCORSHandler(nextHandler, tc.allowOrigin)

			assert.Nil(t, handler)
			assert.Error(t, err)
		})
	}
}
