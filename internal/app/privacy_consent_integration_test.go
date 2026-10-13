package app

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/users"
)

func TestPrivacyConsent(t *testing.T) {
	a := newTestAPI(t)
	notice := func(u testUser) map[string]any {
		status, res := a.do("GET", "/api/v1/users/me", nil, u.Access)
		require.Equal(t, http.StatusOK, status)
		return data(res)["privacy_notice"].(map[string]any)
	}

	t.Run("registration records the notice accepted", func(t *testing.T) {
		u := a.register("consent-new")
		n := notice(u)
		assert.Equal(t, users.PrivacyNoticeVersion, n["accepted_version"])
		assert.Equal(t, users.PrivacyNoticeVersion, n["current_version"])
		assert.NotEmpty(t, n["accepted_at"])
	})

	t.Run("an older account is asked, and can accept", func(t *testing.T) {
		u := a.register("consent-old")
		_, err := a.pool.Exec(context.Background(), `
			UPDATE users SET privacy_notice_version = NULL, privacy_accepted_at = NULL WHERE id = $1`, u.ID)
		require.NoError(t, err)
		n := notice(u)
		assert.NotContains(t, n, "accepted_version")
		assert.Equal(t, users.PrivacyNoticeVersion, n["current_version"])

		status, _ := a.do("POST", "/api/v1/users/me/privacy-consent", map[string]any{"version": "2020-01-01"}, u.Access)
		assert.Equal(t, http.StatusUnprocessableEntity, status, "only the current notice can be accepted")
		status, res := a.do("POST", "/api/v1/users/me/privacy-consent",
			map[string]any{"version": users.PrivacyNoticeVersion}, u.Access)
		require.Equal(t, http.StatusOK, status, "%v", res)
		assert.Equal(t, users.PrivacyNoticeVersion, data(res)["privacy_notice"].(map[string]any)["accepted_version"])
	})

	t.Run("the notice is published", func(t *testing.T) {
		res, body := a.page("/privacy")
		require.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, body, "Privacy notice")
		assert.Contains(t, body, "Frankfurt")
		res, body = a.page("/privacy?lang=am")
		require.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, body, "የግላዊነት ማሳወቂያ")
		_, home := a.page("/trust")
		assert.Contains(t, home, `href="/privacy"`, "linked from every page's footer")
	})
}
