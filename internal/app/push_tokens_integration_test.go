package app

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/notifications"
)

func pushToken(name string) string { return "push-token-" + name + strings.Repeat("x", 40) }

func (a *testAPI) registerPush(u testUser, token string) {
	a.t.Helper()
	status, res := a.do("POST", "/api/v1/users/me/devices",
		map[string]any{"token": token, "platform": "android"}, u.Access)
	require.Equal(a.t, http.StatusOK, status, "%v", res)
}

// A push token belongs to the sign-in it was registered under: once that
// sign-in ends, the phone stops getting the account's notifications, even
// if it never managed to unregister.
func TestPushStopsWhenTheSignInEnds(t *testing.T) {
	a := newTestAPI(t)
	ctx := context.Background()
	svc := notifications.NewService(a.pool)
	tokensFor := func(u testUser) []string {
		tokens, err := svc.DeviceTokensFor(ctx, uuid.MustParse(u.ID))
		require.NoError(t, err)
		return tokens
	}

	t.Run("logout", func(t *testing.T) {
		u := a.register("push-logout")
		a.registerPush(u, pushToken("logout"))
		assert.Equal(t, []string{pushToken("logout")}, tokensFor(u))

		// The session refreshes: same sign-in, still delivered.
		status, res := a.do("POST", "/api/v1/auth/refresh", map[string]any{"refresh_token": u.Refresh}, "")
		require.Equal(t, http.StatusOK, status, "%v", res)
		u.Access = data(res)["access_token"].(string)
		assert.Len(t, tokensFor(u), 1, "a refreshed sign-in is the same sign-in")

		// Signed out without unregistering (e.g. offline at the time).
		status, _ = a.do("POST", "/api/v1/auth/logout", nil, u.Access)
		require.Equal(t, http.StatusOK, status)
		assert.Empty(t, tokensFor(u))
	})

	t.Run("one phone signing out leaves the other", func(t *testing.T) {
		u := a.register("push-two-phones")
		a.registerPush(u, pushToken("phone-a"))
		status, res := a.do("POST", "/api/v1/auth/login", map[string]any{"identifier": u.Email, "password": "password123"}, "")
		require.Equal(t, http.StatusOK, status)
		other := u
		other.Access = data(res)["tokens"].(map[string]any)["access_token"].(string)
		a.registerPush(other, pushToken("phone-b"))
		assert.Len(t, tokensFor(u), 2)

		status, _ = a.do("POST", "/api/v1/auth/logout", nil, u.Access)
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, []string{pushToken("phone-b")}, tokensFor(u))
	})

	t.Run("sign out everywhere removes tokens, old ones included", func(t *testing.T) {
		u := a.register("push-logout-all")
		a.registerPush(u, pushToken("current"))
		// A token registered before tokens were tied to sign-ins.
		_, err := a.pool.Exec(ctx, `
			INSERT INTO device_tokens (id, user_id, token, platform) VALUES (gen_random_uuid(), $1, $2, 'android')`,
			u.ID, pushToken("legacy"))
		require.NoError(t, err)
		assert.Len(t, tokensFor(u), 2)

		status, _ := a.do("POST", "/api/v1/auth/logout-all", nil, u.Access)
		require.Equal(t, http.StatusOK, status)
		var n int
		require.NoError(t, a.pool.QueryRow(ctx, `SELECT count(*) FROM device_tokens WHERE user_id = $1`, u.ID).Scan(&n))
		assert.Zero(t, n)
	})

	t.Run("deactivated account", func(t *testing.T) {
		u := a.register("push-deactivated")
		a.registerPush(u, pushToken("deactivated"))
		status, _ := a.do("DELETE", "/api/v1/users/me", nil, u.Access)
		require.Equal(t, http.StatusOK, status)
		assert.Empty(t, tokensFor(u))
	})
}
