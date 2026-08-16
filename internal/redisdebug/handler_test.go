package redisdebug

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// newMuxWithHandler builds a mux backed by a Redis client that is never
// dialed. It is only valid for cases that fail before touching Redis
// (malformed or empty payloads), so no server is required.
func newMuxWithHandler(t *testing.T) *http.ServeMux {
	t.Helper()

	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:0"})
	t.Cleanup(func() { _ = client.Close() })

	mux := http.NewServeMux()
	NewHandler(slog.Default(), client).Register(mux)

	return mux
}

func doRequest(t *testing.T, mux http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	return rec
}

func TestPutStringRejectsMalformedJSON(t *testing.T) {
	rec := doRequest(t, newMuxWithHandler(t), http.MethodPut, "/redis/string/foo", "{not json")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPutHashRejectsMalformedJSON(t *testing.T) {
	rec := doRequest(t, newMuxWithHandler(t), http.MethodPut, "/redis/hash/foo", "{not json")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPutHashRejectsEmptyObject(t *testing.T) {
	rec := doRequest(t, newMuxWithHandler(t), http.MethodPut, "/redis/hash/foo", "{}")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestPushListRejectsEmptyValues(t *testing.T) {
	rec := doRequest(t, newMuxWithHandler(t), http.MethodPost, "/redis/list/foo", `{"values":[]}`)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUnsupportedMethodYields405(t *testing.T) {
	rec := doRequest(t, newMuxWithHandler(t), http.MethodDelete, "/redis/string/foo", "")

	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestUnknownPathYields404(t *testing.T) {
	rec := doRequest(t, newMuxWithHandler(t), http.MethodGet, "/redis/unknown/foo", "")

	assert.Equal(t, http.StatusNotFound, rec.Code)
}
