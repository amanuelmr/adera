package app

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/auth"
	"github.com/adera-platform/backend/internal/platform/storage"
)

func TestDocsRoutes(t *testing.T) {
	srv := httptest.NewServer(BuildAPI(testConfig(), nil, storage.NewMemory(), auth.NoopProvider{}))
	t.Cleanup(srv.Close)

	resp, err := srv.Client().Get(srv.URL + "/openapi.yaml")
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Content-Type"), "application/yaml")
	require.Contains(t, string(raw), "openapi: 3.0.3")
	require.Contains(t, string(raw), "  - url: /\n")
	require.NotContains(t, string(raw), "url: http://localhost")

	resp, err = srv.Client().Get(srv.URL + "/docs")
	require.NoError(t, err)
	defer resp.Body.Close()
	raw, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Content-Type"), "text/html")
	require.Contains(t, string(raw), `url: "/openapi.yaml"`)
	csp := resp.Header.Get("Content-Security-Policy")
	require.Contains(t, csp, "https://unpkg.com")

	// The CDN files are pinned and integrity-checked, and the one inline
	// script is allowed by its hash rather than by 'unsafe-inline'.
	html := string(raw)
	require.Contains(t, html, "swagger-ui-dist@5.33.0/swagger-ui-bundle.js\" integrity=\"sha384-")
	require.Contains(t, html, "swagger-ui-dist@5.33.0/swagger-ui.css\" integrity=\"sha384-")
	require.NotContains(t, html, "swagger-ui-dist@5/")
	scriptSrc, _, _ := strings.Cut(strings.SplitN(csp, "script-src ", 2)[1], ";")
	require.NotContains(t, scriptSrc, "unsafe-inline")
	inline := html[strings.Index(html, "<script>")+len("<script>") : strings.LastIndex(html, "</script>")]
	sum := sha256.Sum256([]byte(inline))
	require.Contains(t, scriptSrc, "'sha256-"+base64.StdEncoding.EncodeToString(sum[:])+"'")
}
