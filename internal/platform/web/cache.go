package web

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
)

var publicCachePolicies = map[string]string{
	"GET /api/v1/categories":                     "public, max-age=3600, stale-while-revalidate=86400",
	"GET /api/v1/categories/{idOrCode}":          "public, max-age=3600, stale-while-revalidate=86400",
	"GET /api/v1/categories/{idOrCode}/criteria": "public, max-age=3600, stale-while-revalidate=86400",
	"GET /api/v1/locations/cities":               "public, max-age=3600, stale-while-revalidate=86400",
	"GET /api/v1/locations/cities/{id}/areas":    "public, max-age=3600, stale-while-revalidate=86400",
	"GET /api/v1/targets":                        "public, max-age=30, stale-while-revalidate=300",
	"GET /api/v1/targets/top-rated":              "public, max-age=30, stale-while-revalidate=300",
	"GET /api/v1/targets/trending":               "public, max-age=30, stale-while-revalidate=300",
	"GET /api/v1/targets/{idOrSlug}":             "public, max-age=30, stale-while-revalidate=300",
	"GET /api/v1/targets/{id}/reviews":           "public, max-age=30, stale-while-revalidate=300",
	"GET /api/v1/reviews/{id}":                   "public, max-age=30, stale-while-revalidate=300",
	"GET /api/v1/targets/{id}/stats":             "public, max-age=30, stale-while-revalidate=300",
	"GET /api/v1/targets/{id}/reality-check":     "public, max-age=30, stale-while-revalidate=300",
	"GET /api/v1/search/targets":                 "public, max-age=30, stale-while-revalidate=300",
	"GET /api/v1/businesses/{id}":                "public, max-age=30, stale-while-revalidate=300",
}

type bufferedResponseWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (w *bufferedResponseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *bufferedResponseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(p)
}

// PublicCache adds conditional caching only to anonymous public reads. Other
// responses retain the no-store default established by SecureHeaders.
func PublicCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (r.Method != http.MethodGet && r.Method != http.MethodHead) ||
			r.Header.Get("Authorization") != "" || !strings.HasPrefix(r.URL.Path, "/api/v1/") {
			next.ServeHTTP(w, r)
			return
		}

		buffered := &bufferedResponseWriter{ResponseWriter: w}
		next.ServeHTTP(buffered, r)
		status := buffered.status
		if status == 0 {
			status = http.StatusOK
		}
		policy, cacheable := publicCachePolicies[r.Pattern]
		if status != http.StatusOK || !cacheable {
			w.WriteHeader(status)
			_, _ = w.Write(buffered.body.Bytes())
			return
		}

		sum := sha256.Sum256(buffered.body.Bytes())
		etag := `"` + hex.EncodeToString(sum[:16]) + `"`
		w.Header().Set("Cache-Control", policy)
		w.Header().Set("ETag", etag)
		w.Header().Add("Vary", "Authorization")
		if etagMatches(r.Header.Get("If-None-Match"), etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.WriteHeader(status)
		if r.Method != http.MethodHead {
			_, _ = w.Write(buffered.body.Bytes())
		}
	})
}

func etagMatches(header, etag string) bool {
	for value := range strings.SplitSeq(header, ",") {
		value = strings.TrimSpace(value)
		if value == "*" || value == etag || strings.TrimPrefix(value, "W/") == etag {
			return true
		}
	}
	return false
}
