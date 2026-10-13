package app

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/config"
)

// The scrape token isn't a user token, so /metrics must be checked before
// the API's own authentication would reject it.
func TestMetricsScrapeToken(t *testing.T) {
	const token = "metrics-scrape-token-at-least-32-bytes"
	a := newTestAPIWith(t, func(c *config.Config) { c.MetricsToken = token })
	scrape := func(auth string) (int, string) {
		req, err := http.NewRequest("GET", a.srv.URL+"/metrics", nil)
		require.NoError(t, err)
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		resp, err := a.srv.Client().Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(body)
	}
	status, _ := scrape("")
	assert.Equal(t, http.StatusUnauthorized, status)
	status, _ = scrape("Bearer not-the-token")
	assert.Equal(t, http.StatusUnauthorized, status)
	status, body := scrape("Bearer " + token)
	require.Equal(t, http.StatusOK, status)
	assert.Contains(t, body, "adera_http_requests_total")
}

func TestProductionHeadersAndClosedMetrics(t *testing.T) {
	a := newTestAPIWith(t, func(c *config.Config) { c.Env = config.EnvProduction })
	resp, err := a.srv.Client().Get(a.srv.URL + "/health")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, "max-age=31536000; includeSubDomains", resp.Header.Get("Strict-Transport-Security"))

	resp, err = a.srv.Client().Get(a.srv.URL + "/metrics")
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "no token configured: not served")
}
