//go:build database

package redisdebug

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"youtube_tracker/internal/helpers"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// drain reads and returns the full body of a response.
func drain(t *testing.T, r io.Reader) string {
	t.Helper()

	b, err := io.ReadAll(r)
	require.NoError(t, err)

	return string(b)
}

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	ctx := context.Background()
	container, err := helpers.CreateRedisContainer(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	client := redis.NewClient(&redis.Options{Addr: container.Addr})
	t.Cleanup(func() { _ = client.Close() })

	mux := http.NewServeMux()
	NewHandler(slog.Default(), client).Register(mux)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return srv
}

func put(t *testing.T, srv *httptest.Server, path, body string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(http.MethodPut, srv.URL+path, strings.NewReader(body))
	require.NoError(t, err)
	resp, err := srv.Client().Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	return resp
}

func post(t *testing.T, srv *httptest.Server, path, body string) *http.Response {
	t.Helper()

	resp, err := srv.Client().Post(srv.URL+path, "application/json", strings.NewReader(body))
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	return resp
}

func get(t *testing.T, srv *httptest.Server, path string) *http.Response {
	t.Helper()

	resp, err := srv.Client().Get(srv.URL + path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	return resp
}

func TestStringRoundTrip(t *testing.T) {
	srv := newTestServer(t)

	putResp := put(t, srv, "/redis/string/greeting", `{"value":"hello"}`)
	assert.Equal(t, http.StatusNoContent, putResp.StatusCode)

	getResp := get(t, srv, "/redis/string/greeting")
	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	assert.JSONEq(t, `{"value":"hello"}`, drain(t, getResp.Body))
}

func TestStringUpsertOverwrites(t *testing.T) {
	srv := newTestServer(t)

	put(t, srv, "/redis/string/k", `{"value":"first"}`)
	put(t, srv, "/redis/string/k", `{"value":"second"}`)

	getResp := get(t, srv, "/redis/string/k")
	assert.JSONEq(t, `{"value":"second"}`, drain(t, getResp.Body))
}

func TestStringGetMissingKeyYields404(t *testing.T) {
	srv := newTestServer(t)

	getResp := get(t, srv, "/redis/string/does-not-exist")
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
}

func TestHashRoundTrip(t *testing.T) {
	srv := newTestServer(t)

	putResp := put(t, srv, "/redis/hash/user", `{"name":"alice","role":"admin"}`)
	assert.Equal(t, http.StatusNoContent, putResp.StatusCode)

	getResp := get(t, srv, "/redis/hash/user")
	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	assert.JSONEq(t, `{"name":"alice","role":"admin"}`, drain(t, getResp.Body))
}

func TestHashPutMergesFields(t *testing.T) {
	srv := newTestServer(t)

	put(t, srv, "/redis/hash/user", `{"name":"alice"}`)
	put(t, srv, "/redis/hash/user", `{"role":"admin"}`)

	getResp := get(t, srv, "/redis/hash/user")
	assert.JSONEq(t, `{"name":"alice","role":"admin"}`, drain(t, getResp.Body))
}

func TestHashGetMissingKeyYields404(t *testing.T) {
	srv := newTestServer(t)

	getResp := get(t, srv, "/redis/hash/nope")
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
}

func TestListRoundTrip(t *testing.T) {
	srv := newTestServer(t)

	postResp := post(t, srv, "/redis/list/queue", `{"values":["a","b"]}`)
	assert.Equal(t, http.StatusNoContent, postResp.StatusCode)

	getResp := get(t, srv, "/redis/list/queue")
	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	assert.JSONEq(t, `{"values":["a","b"]}`, drain(t, getResp.Body))
}

func TestListPushAppends(t *testing.T) {
	srv := newTestServer(t)

	post(t, srv, "/redis/list/queue", `{"values":["a"]}`)
	post(t, srv, "/redis/list/queue", `{"values":["b","c"]}`)

	getResp := get(t, srv, "/redis/list/queue")
	assert.JSONEq(t, `{"values":["a","b","c"]}`, drain(t, getResp.Body))
}

func TestListGetMissingKeyYields404(t *testing.T) {
	srv := newTestServer(t)

	getResp := get(t, srv, "/redis/list/nope")
	assert.Equal(t, http.StatusNotFound, getResp.StatusCode)
}
