package app

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/config"
	"github.com/adera-platform/backend/internal/platform/web"
)

// Behind a proxy every request arrives from the proxy's address; the limiter
// must still give each caller their own bucket.
func TestRateLimitPerClientBehindProxy(t *testing.T) {
	a := newTestAPIWith(t, func(c *config.Config) {
		c.RateLimitEnabled = true
		c.ClientIPSource = web.IPSource{Header: "Cf-Connecting-Ip"}
	})
	login := func(ip string, n int) int {
		status, _ := a.do("POST", "/api/v1/auth/login", map[string]any{
			"identifier": fmt.Sprintf("nobody-%s-%d@example.com", ip, n), "password": "wrongwrong",
		}, "", "CF-Connecting-IP", ip)
		return status
	}
	var last int
	for i := range 8 {
		last = login("196.188.1.2", i)
	}
	require.Equal(t, http.StatusTooManyRequests, last, "one caller is limited")
	assert.Equal(t, http.StatusUnauthorized, login("196.188.9.9", 0), "another caller behind the same proxy is not")
}

// Refresh is keyed by the token: many phones share a carrier address, and
// one user refreshing repeatedly must not lock out the rest.
func TestRefreshIsNotLimitedByAddress(t *testing.T) {
	a := newTestAPIWith(t, func(c *config.Config) { c.RateLimitEnabled = true })
	u := a.register("refresher")
	token := u.Refresh
	for i := range 8 {
		status, res := a.do("POST", "/api/v1/auth/refresh", map[string]any{"refresh_token": token}, "")
		require.Equal(t, http.StatusOK, status, "refresh %d: %v", i, res)
		token = data(res)["refresh_token"].(string)
	}
}

func TestAdminRequestInfo(t *testing.T) {
	a := newTestAPIWith(t, func(c *config.Config) { c.ClientIPSource = web.IPSource{Hops: 1} })
	admin := a.register("reqinfoadmin")
	a.grantRoles(&admin, "admin")
	user := a.register("reqinfouser")

	status, _ := a.do("GET", "/api/v1/admin/request-info", nil, user.Access)
	assert.Equal(t, http.StatusForbidden, status, "admins only")

	status, res := a.do("GET", "/api/v1/admin/request-info", nil, admin.Access,
		"X-Forwarded-For", "6.6.6.6, 196.188.1.2", "CF-Connecting-IP", "196.188.1.2")
	require.Equal(t, http.StatusOK, status, "%v", res)
	d := data(res)
	assert.Equal(t, "xff:1", d["client_ip_source"])
	assert.Equal(t, "196.188.1.2", d["client_ip"])
	assert.Equal(t, []any{"6.6.6.6, 196.188.1.2"}, d["x_forwarded_for"])
	assert.Equal(t, map[string]any{"CF-Connecting-IP": "196.188.1.2"}, d["proxy_headers"])
	assert.NotEmpty(t, d["remote_addr"])
}
