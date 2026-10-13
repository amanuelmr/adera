package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMethodLabelIsBounded(t *testing.T) {
	for _, m := range []string{"GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"} {
		assert.Equal(t, m, methodLabel(m))
	}
	for _, m := range []string{"PROPFIND", "get", "X" + string(make([]byte, 100)), ""} {
		assert.Equal(t, "OTHER", methodLabel(m))
	}
}

func TestMetricsEndpoint(t *testing.T) {
	const token = "metrics-scrape-token-at-least-32-bytes"
	fallthroughHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	call := func(mw Middleware, method, path string, headers ...string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, nil)
		for i := 0; i+1 < len(headers); i += 2 {
			r.Header.Set(headers[i], headers[i+1])
		}
		rec := httptest.NewRecorder()
		mw(fallthroughHandler).ServeHTTP(rec, r)
		return rec
	}

	guarded := MetricsEndpoint(token, false)
	rec := call(guarded, "GET", "/metrics")
	assert.Equal(t, http.StatusUnauthorized, rec.Code, "no token")
	assert.Contains(t, rec.Header().Get("WWW-Authenticate"), "Bearer")
	assert.Equal(t, http.StatusUnauthorized, call(guarded, "GET", "/metrics", "Authorization", "Bearer wrong").Code)
	assert.Equal(t, http.StatusUnauthorized, call(guarded, "GET", "/metrics", "Authorization", token).Code, "scheme required")
	rec = call(guarded, "GET", "/metrics", "Authorization", "Bearer "+token)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "adera_http_requests_in_flight")
	assert.Equal(t, http.StatusTeapot, call(guarded, "GET", "/health").Code, "other paths pass through")
	assert.Equal(t, http.StatusTeapot, call(guarded, "POST", "/metrics").Code, "other methods are the router's business")

	closed := MetricsEndpoint("", false)
	assert.Equal(t, http.StatusTeapot, call(closed, "GET", "/metrics").Code, "not served without a token in production")

	open := MetricsEndpoint("", true)
	assert.Equal(t, http.StatusOK, call(open, "GET", "/metrics").Code, "open in development")
}

func TestSecureHeadersHSTS(t *testing.T) {
	ok := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	rec := httptest.NewRecorder()
	SecureHeaders(true)(ok).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	assert.Equal(t, "max-age=31536000; includeSubDomains", rec.Header().Get("Strict-Transport-Security"))
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))

	rec = httptest.NewRecorder()
	SecureHeaders(false)(ok).ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	assert.Empty(t, rec.Header().Get("Strict-Transport-Security"))
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
}
