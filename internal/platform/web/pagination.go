package web

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// Cursor is an opaque keyset-pagination cursor: the sort key value and row ID
// of the last item on the previous page. It is base64-encoded JSON; clients
// must treat it as opaque.
type Cursor struct {
	// K is the primary sort key of the last row: an RFC3339Nano timestamp,
	// or a numeric string for score-ordered listings.
	K string `json:"k"`
	// ID breaks ties deterministically.
	ID uuid.UUID `json:"id"`
}

// Encode serializes the cursor for transport.
func (c Cursor) Encode() string {
	b, _ := json.Marshal(c) // struct of string+uuid cannot fail to marshal
	return base64.RawURLEncoding.EncodeToString(b)
}

// Time parses the sort key as a timestamp.
func (c Cursor) Time() (time.Time, bool) {
	t, err := time.Parse(time.RFC3339Nano, c.K)
	return t, err == nil
}

// Float parses the sort key as a number.
func (c Cursor) Float() (float64, bool) {
	f, err := strconv.ParseFloat(c.K, 64)
	return f, err == nil
}

// TimeCursor builds a cursor from a timestamp sort key.
func TimeCursor(t time.Time, id uuid.UUID) Cursor {
	return Cursor{K: t.UTC().Format(time.RFC3339Nano), ID: id}
}

// FloatCursor builds a cursor from a numeric sort key.
func FloatCursor(f float64, id uuid.UUID) Cursor {
	return Cursor{K: strconv.FormatFloat(f, 'f', -1, 64), ID: id}
}

// ParseCursor decodes the "cursor" query parameter, if present.
func ParseCursor(r *http.Request) (*Cursor, error) {
	raw := r.URL.Query().Get("cursor")
	if raw == "" {
		return nil, nil
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, ErrValidation("invalid cursor").WithDetail("cursor", "malformed")
	}
	var c Cursor
	if err := json.Unmarshal(b, &c); err != nil || c.ID == uuid.Nil {
		return nil, ErrValidation("invalid cursor").WithDetail("cursor", "malformed")
	}
	return &c, nil
}

const (
	DefaultPageSize = 20
	MaxPageSize     = 50
)

// ParseLimit reads the "limit" query parameter, clamped to [1, MaxPageSize].
func ParseLimit(r *http.Request) int {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return DefaultPageSize
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return DefaultPageSize
	}
	return min(n, MaxPageSize)
}

// ParseUUID reads a path value as a UUID.
func ParseUUID(r *http.Request, name string) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue(name))
	if err != nil {
		return uuid.Nil, ErrValidation("invalid identifier").WithDetail(name, "must be a UUID")
	}
	return id, nil
}
