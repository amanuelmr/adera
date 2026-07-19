package web

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// envelope is the uniform success response body.
type envelope struct {
	Data any   `json:"data"`
	Meta *Meta `json:"meta,omitempty"`
}

// Meta carries pagination metadata for collection responses.
type Meta struct {
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

type errorBody struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Details   map[string]string `json:"details,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

// Respond writes v inside the standard envelope.
func Respond(w http.ResponseWriter, status int, v any) {
	respondJSON(w, status, envelope{Data: v})
}

// RespondPage writes a collection with pagination metadata.
func RespondPage(w http.ResponseWriter, status int, v any, meta Meta) {
	respondJSON(w, status, envelope{Data: v, Meta: &meta})
}

// RespondError normalizes err, logs internals, and writes the JSON error body.
func RespondError(w http.ResponseWriter, r *http.Request, err error) {
	appErr := AsError(err)
	if appErr.Internal != nil || appErr.Status >= 500 {
		slog.ErrorContext(r.Context(), "request failed",
			"code", appErr.Code,
			"status", appErr.Status,
			"error", appErr.Error(),
			"method", r.Method,
			"path", r.URL.Path,
		)
	}
	if appErr.Status == http.StatusTooManyRequests {
		w.Header().Set("Retry-After", "60")
	}
	respondJSON(w, appErr.Status, errorBody{Error: errorPayload{
		Code:      appErr.Code,
		Message:   appErr.Message,
		Details:   appErr.Details,
		RequestID: RequestIDFromContext(r.Context()),
	}})
}

func respondJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// Headers are gone; nothing safe left to do but log.
		slog.Error("encoding response failed", "error", err)
	}
}

// DefaultMaxBodyBytes bounds JSON request bodies (uploads go straight to
// object storage, never through the API).
const DefaultMaxBodyBytes = 64 << 10 // 64 KiB

// DecodeJSON strictly decodes the request body into dst: size-capped,
// unknown fields rejected (mass-assignment defense), single JSON value only.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, DefaultMaxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var maxBytesErr *http.MaxBytesError
		switch {
		case errors.As(err, &maxBytesErr):
			return &Error{Status: http.StatusRequestEntityTooLarge, Code: CodePayloadTooLarge, Message: "request body too large"}
		case errors.Is(err, io.EOF):
			return ErrInvalidJSON(errors.New("empty body"))
		default:
			return ErrInvalidJSON(err)
		}
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return ErrInvalidJSON(errors.New("body must contain a single JSON value"))
	}
	return nil
}

// ClientIP extracts the caller IP. X-Forwarded-For is only honored when
// trustProxy is set (i.e., we run behind our own reverse proxy) and then only
// the rightmost hop is used.
func ClientIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			return strings.TrimSpace(parts[len(parts)-1])
		}
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		host = host[:i]
	}
	return host
}

type ctxKey int

const (
	ctxKeyRequestID ctxKey = iota
	ctxKeyPrincipal
)

// RequestIDFromContext returns the request ID set by the RequestID middleware.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyRequestID).(string); ok {
		return v
	}
	return ""
}
