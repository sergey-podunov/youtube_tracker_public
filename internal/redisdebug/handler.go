// Package redisdebug exposes unauthenticated HTTP endpoints that read and
// write Redis directly. It exists to support manual/end-to-end testing
// (including against prod) and is intended to be removed once no longer
// needed. Do not build production features on top of it.
package redisdebug

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/redis/go-redis/v9"
)

// Handler serves the /redis/* debug endpoints backed by a Redis client.
type Handler struct {
	logger *slog.Logger
	client *redis.Client
}

// stringPayload is the body for the string endpoints: {"value": "..."}.
type stringPayload struct {
	Value string `json:"value"`
}

// listPayload is the body for the list endpoints: {"values": ["a", "b"]}.
type listPayload struct {
	Values []string `json:"values"`
}

// Hash endpoints use a bare JSON object of field->value, e.g. {"f1": "v1"}.

// NewHandler returns a Handler that operates on the given Redis client.
func NewHandler(logger *slog.Logger, client *redis.Client) *Handler {
	return &Handler{logger: logger, client: client}
}

// Register mounts the debug routes onto mux. Routing uses Go 1.22 method +
// wildcard patterns, so unmatched methods yield 405 automatically.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("PUT /redis/string/{key}", h.putString)
	mux.HandleFunc("GET /redis/string/{key}", h.getString)
	mux.HandleFunc("PUT /redis/hash/{key}", h.putHash)
	mux.HandleFunc("GET /redis/hash/{key}", h.getHash)
	mux.HandleFunc("POST /redis/list/{key}", h.pushList)
	mux.HandleFunc("GET /redis/list/{key}", h.getList)
}

// putString sets key to the payload's value (SET).
func (h *Handler) putString(w http.ResponseWriter, r *http.Request) {
	var payload stringPayload
	if !decodeBody(w, r, &payload) {
		return
	}

	if err := h.client.Set(r.Context(), r.PathValue("key"), payload.Value, 0).Err(); err != nil {
		h.writeError(w, r, http.StatusInternalServerError, "set failed", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// getString returns the value stored at key (GET); 404 if it does not exist.
func (h *Handler) getString(w http.ResponseWriter, r *http.Request) {
	value, err := h.client.Get(r.Context(), r.PathValue("key")).Result()
	if errors.Is(err, redis.Nil) {
		http.NotFound(w, r)
		return
	}
	if err != nil {
		h.writeError(w, r, http.StatusInternalServerError, "get failed", err)
		return
	}

	h.writeJSON(w, r, stringPayload{Value: value})
}

// putHash sets the payload's fields on the hash at key (HSET).
func (h *Handler) putHash(w http.ResponseWriter, r *http.Request) {
	var fields map[string]string
	if !decodeBody(w, r, &fields) {
		return
	}
	if len(fields) == 0 {
		h.writeError(w, r, http.StatusBadRequest, "at least one field is required", nil)
		return
	}

	if err := h.client.HSet(r.Context(), r.PathValue("key"), fields).Err(); err != nil {
		h.writeError(w, r, http.StatusInternalServerError, "hset failed", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// getHash returns all fields of the hash at key (HGETALL); 404 if empty/missing.
func (h *Handler) getHash(w http.ResponseWriter, r *http.Request) {
	fields, err := h.client.HGetAll(r.Context(), r.PathValue("key")).Result()
	if err != nil {
		h.writeError(w, r, http.StatusInternalServerError, "hgetall failed", err)
		return
	}
	if len(fields) == 0 {
		http.NotFound(w, r)
		return
	}

	h.writeJSON(w, r, fields)
}

// pushList appends the payload's values to the list at key (RPUSH).
func (h *Handler) pushList(w http.ResponseWriter, r *http.Request) {
	var payload listPayload
	if !decodeBody(w, r, &payload) {
		return
	}
	if len(payload.Values) == 0 {
		h.writeError(w, r, http.StatusBadRequest, "at least one value is required", nil)
		return
	}

	values := make([]any, len(payload.Values))
	for i, v := range payload.Values {
		values[i] = v
	}

	if err := h.client.RPush(r.Context(), r.PathValue("key"), values...).Err(); err != nil {
		h.writeError(w, r, http.StatusInternalServerError, "rpush failed", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// getList returns all elements of the list at key (LRANGE); 404 if empty/missing.
func (h *Handler) getList(w http.ResponseWriter, r *http.Request) {
	values, err := h.client.LRange(r.Context(), r.PathValue("key"), 0, -1).Result()
	if err != nil {
		h.writeError(w, r, http.StatusInternalServerError, "lrange failed", err)
		return
	}
	if len(values) == 0 {
		http.NotFound(w, r)
		return
	}

	h.writeJSON(w, r, listPayload{Values: values})
}

// decodeBody decodes the request body into dst, rejecting unknown fields.
// On failure it writes a 400 and returns false.
func decodeBody(w http.ResponseWriter, r *http.Request, dst any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return false
	}

	return true
}

// writeJSON serializes body as JSON with a 200 status.
func (h *Handler) writeJSON(w http.ResponseWriter, r *http.Request, body any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(body); err != nil {
		h.logger.WarnContext(r.Context(), "Failed to encode response", "error", err)
	}
}

// writeError logs err (when non-nil) and writes msg with the given status.
func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, status int, msg string, err error) {
	if err != nil {
		h.logger.ErrorContext(r.Context(), "Redis debug request failed", "msg", msg, "error", err)
	}
	http.Error(w, msg, status)
}
