package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCursorRoundTrip(t *testing.T) {
	now := time.Now().UTC()
	id := uuid.New()
	c := TimeCursor(now, id)

	req := httptest.NewRequest("GET", "/?cursor="+c.Encode(), nil)
	parsed, err := ParseCursor(req)
	require.NoError(t, err)
	require.NotNil(t, parsed)
	assert.Equal(t, id, parsed.ID)
	ts, ok := parsed.Time()
	assert.True(t, ok)
	assert.True(t, ts.Equal(now))
}

func TestParseCursorRejectsGarbage(t *testing.T) {
	for _, raw := range []string{"garbage!!", "bm90LWpzb24", "e30"} { // junk, "not-json", "{}"
		req := httptest.NewRequest("GET", "/?cursor="+raw, nil)
		_, err := ParseCursor(req)
		assert.Error(t, err, raw)
	}
	req := httptest.NewRequest("GET", "/", nil)
	parsed, err := ParseCursor(req)
	require.NoError(t, err)
	assert.Nil(t, parsed)
}

func TestParseLimit(t *testing.T) {
	for raw, want := range map[string]int{"": 20, "5": 5, "500": 50, "0": 20, "-3": 20, "abc": 20} {
		req := httptest.NewRequest("GET", "/?limit="+raw, nil)
		assert.Equal(t, want, ParseLimit(req), "limit=%q", raw)
	}
}

func TestHasRoleAdminImpliesModerator(t *testing.T) {
	p := Principal{Roles: []string{RoleAdmin}}
	assert.True(t, p.HasRole(RoleAdmin))
	assert.True(t, p.HasRole(RoleModerator), "admin implies moderator")
	assert.False(t, p.HasRole(RoleBusinessOwner))

	c := Principal{Roles: []string{RoleCustomer}}
	assert.False(t, c.HasRole(RoleModerator))
}

func TestDecodeJSONStrict(t *testing.T) {
	type dst struct {
		Name string `json:"name"`
	}
	call := func(body string) error {
		req := httptest.NewRequest("POST", "/", strings.NewReader(body))
		var d dst
		return DecodeJSON(httptest.NewRecorder(), req, &d)
	}
	assert.NoError(t, call(`{"name":"ok"}`))
	// Unknown fields rejected (mass-assignment defense).
	assert.Error(t, call(`{"name":"ok","is_admin":true}`))
	// Trailing data rejected.
	assert.Error(t, call(`{"name":"ok"}{"again":1}`))
	// Empty body rejected.
	assert.Error(t, call(""))
	// Oversized body rejected.
	err := call(`{"name":"` + strings.Repeat("x", DefaultMaxBodyBytes+100) + `"}`)
	require.Error(t, err)
	appErr := AsError(err)
	assert.Equal(t, CodePayloadTooLarge, appErr.Code)
}

func TestAuthenticateMiddleware(t *testing.T) {
	verify := func(_ context.Context, token string) (Principal, error) {
		if token == "good" {
			return Principal{UserID: uuid.New(), Roles: []string{RoleCustomer}}, nil
		}
		return Principal{}, assertAnError
	}
	handler := Authenticate(verify)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := PrincipalFromContext(r.Context())
		if ok {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}))

	// No header: passes through unauthenticated.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	assert.Equal(t, http.StatusNoContent, rec.Code)

	// Valid bearer token.
	rec = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer good")
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Invalid token rejected outright.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer bad")
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	// Malformed header rejected.
	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwdw==")
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

var assertAnError = &Error{Status: 401, Code: CodeUnauthorized, Message: "nope"}

func TestCORSAllowlist(t *testing.T) {
	handler := CORS([]string{"https://app.adera.et"})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://app.adera.et")
	handler.ServeHTTP(rec, req)
	assert.Equal(t, "https://app.adera.et", rec.Header().Get("Access-Control-Allow-Origin"))

	rec = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	handler.ServeHTTP(rec, req)
	assert.Empty(t, rec.Header().Get("Access-Control-Allow-Origin"))
}

func TestRecoverMiddleware(t *testing.T) {
	handler := Recover(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "boom", "panic details must not leak to clients")
}

func TestRespondErrorHidesInternals(t *testing.T) {
	rec := httptest.NewRecorder()
	RespondError(rec, httptest.NewRequest("GET", "/", nil), ErrInternal(assertAnError))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "nope")
	assert.Contains(t, rec.Body.String(), CodeInternal)
}
