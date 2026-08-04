package app

import (
	"io"
	"net/http"
	"net/http/httptest"
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
	require.Contains(t, resp.Header.Get("Content-Security-Policy"), "https://unpkg.com")
}
