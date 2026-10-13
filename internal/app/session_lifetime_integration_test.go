package app

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/security"
)

// Refreshing keeps a session going, but only up to SessionMaxLifetime from
// the original sign-in.
func TestSessionMaxLifetime(t *testing.T) {
	a := newTestAPI(t)
	ctx := context.Background()
	maxLifetime := a.cfg.SessionMaxLifetime

	refresh := func(token string) (int, string) {
		status, res := a.do("POST", "/api/v1/auth/refresh", map[string]any{"refresh_token": token}, "")
		if status != http.StatusOK {
			return status, ""
		}
		return status, data(res)["refresh_token"].(string)
	}
	// backdate moves the whole session family's sign-in into the past.
	backdate := func(token string, age time.Duration) {
		_, err := a.pool.Exec(ctx, `
			UPDATE auth_sessions SET family_created_at = now() - make_interval(secs => $2)
			WHERE family_id = (SELECT family_id FROM auth_sessions WHERE refresh_token_hash = $1)`,
			security.HashToken(token), age.Seconds())
		require.NoError(t, err)
	}

	t.Run("rotation carries the sign-in time forward", func(t *testing.T) {
		u := a.register("lifetime-carry")
		var first, next time.Time
		require.NoError(t, a.pool.QueryRow(ctx, `SELECT family_created_at FROM auth_sessions WHERE refresh_token_hash = $1`,
			security.HashToken(u.Refresh)).Scan(&first))
		status, token := refresh(u.Refresh)
		require.Equal(t, http.StatusOK, status)
		require.NoError(t, a.pool.QueryRow(ctx, `SELECT family_created_at FROM auth_sessions WHERE refresh_token_hash = $1`,
			security.HashToken(token)).Scan(&next))
		assert.True(t, first.Equal(next), "%v vs %v", first, next)
	})

	t.Run("near the limit, the new token expires at the limit", func(t *testing.T) {
		u := a.register("lifetime-near")
		backdate(u.Refresh, maxLifetime-24*time.Hour)
		status, token := refresh(u.Refresh)
		require.Equal(t, http.StatusOK, status)
		var expires time.Time
		require.NoError(t, a.pool.QueryRow(ctx, `SELECT expires_at FROM auth_sessions WHERE refresh_token_hash = $1`,
			security.HashToken(token)).Scan(&expires))
		assert.WithinDuration(t, time.Now().Add(24*time.Hour), expires, time.Minute,
			"capped at the lifetime, not a full refresh TTL")
	})

	t.Run("past the limit, refresh fails", func(t *testing.T) {
		u := a.register("lifetime-past")
		backdate(u.Refresh, maxLifetime+time.Minute)
		status, _ := refresh(u.Refresh)
		assert.Equal(t, http.StatusUnauthorized, status)
	})
}
