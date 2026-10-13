package app

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/media"
	"github.com/adera-platform/backend/internal/platform/config"
	"github.com/adera-platform/backend/internal/privacy"
)

func TestPrivacyExport(t *testing.T) {
	a := newTestAPI(t)
	admin := a.register("privacy-admin")
	a.grantRoles(&admin, "admin")
	user := a.register("privacy-user")
	target := a.createTarget(admin, "Privacy Export Cafe", catRestaurantID, "cafe")
	reviewID := a.review(user, target, 4, map[string]any{
		"incentive_type":     "discount",
		"disclosure_details": "A launch discount was applied.",
	})

	status, res := a.do("GET", "/api/v1/users/me/data-export", nil, user.Access)
	require.Equal(t, http.StatusOK, status, "%v", res)
	export := data(res)
	assert.Equal(t, "2", export["schema_version"])
	profile := export["profile"].(map[string]any)
	assert.Equal(t, user.ID, profile["id"])
	assert.Equal(t, user.Email, profile["email"])
	reviews := export["reviews"].([]any)
	require.Len(t, reviews, 1)
	assert.Equal(t, reviewID, reviews[0].(map[string]any)["id"])
	assert.Equal(t, "discount", reviews[0].(map[string]any)["incentive_type"])

	raw, err := json.Marshal(export)
	require.NoError(t, err)
	for _, secret := range []string{"password_hash", "refresh_token_hash", "code_hash", "object_key"} {
		assert.NotContains(t, string(raw), secret, "exports must not expose internal secret %s", secret)
	}
}

// The export covers what the platform records about someone, including the
// things they didn't type: devices, fraud signals, moderation decisions about
// their content, and their files.
func TestPrivacyExportCoversRecordedData(t *testing.T) {
	a := newTestAPIWith(t, func(c *config.Config) {
		c.SignalHashKey = "export-test-signal-key-at-least-32-bytes"
	})
	ctx := context.Background()
	mod := a.register("export-mod")
	a.grantRoles(&mod, "moderator")
	user := a.register("export-user")
	target := a.createTarget(mod, "Export Coverage Cafe", catRestaurantID, "cafe")
	review := a.reviewFrom(user, target, "196.188.4.5", "4b0c3c8e-56a8-4b77-9d0e-6a3c1f2b9e10")
	photo, _ := a.uploadPhoto(user, review)
	status, res := a.do("POST", "/api/v1/reviews/"+review+"/evidence",
		map[string]any{"kind": "receipt", "content_type": "image/jpeg"}, user.Access)
	require.Equal(t, http.StatusCreated, status, "%v", res)
	evidenceID := data(res)["upload_id"].(string)
	receipt := exifJPEG(t)
	require.NoError(t, a.store.Put(ctx, a.cfg.StoragePrivateBucket, data(res)["upload"].(map[string]any)["key"].(string),
		bytes.NewReader(receipt), int64(len(receipt)), "image/jpeg"))
	status, _ = a.do("POST", "/api/v1/evidence/"+evidenceID+"/finalize", nil, user.Access)
	require.Equal(t, http.StatusOK, status)
	a.registerPush(user, pushToken("export"))
	status, _ = a.do("POST", "/api/v1/moderation/reviews/"+review+"/decision",
		map[string]any{"action": "under_review", "note": "internal: checking the receipt"}, mod.Access)
	require.Equal(t, http.StatusOK, status)

	status, res = a.do("GET", "/api/v1/users/me/data-export", nil, user.Access)
	require.Equal(t, http.StatusOK, status, "%v", res)
	export := data(res)
	assert.Equal(t, "2", export["schema_version"])

	devices := export["devices"].([]any)
	require.Len(t, devices, 1)
	assert.Equal(t, "android", devices[0].(map[string]any)["platform"])
	assert.NotContains(t, devices[0], "token", "the push token is a send credential")

	signals := export["fraud_signals"].([]any)
	require.Len(t, signals, 1)
	assert.Equal(t, true, signals[0].(map[string]any)["network_recorded"])
	assert.Equal(t, true, signals[0].(map[string]any)["device_recorded"])

	decisions := export["moderation_decisions"].([]any)
	require.Len(t, decisions, 1)
	assert.Equal(t, "review_under_review", decisions[0].(map[string]any)["action"])
	assert.NotContains(t, decisions[0], "note", "moderators' notes stay internal")
	assert.NotContains(t, decisions[0], "actor_id")

	media := export["review_media_metadata"].([]any)
	require.Len(t, media, 1)
	assert.Equal(t, a.cfg.StoragePublicBaseURL+"/"+photo, media[0].(map[string]any)["url"])
	evidence := export["review_evidence_metadata"].([]any)
	require.Len(t, evidence, 1)
	assert.Contains(t, evidence[0].(map[string]any)["download_url"], "signed=1")
}

func (a *testAPI) login(u *testUser) int {
	a.t.Helper()
	status, res := a.do("POST", "/api/v1/auth/login", map[string]any{"identifier": u.Email, "password": "password123"}, "")
	if status == http.StatusOK {
		tokens := data(res)["tokens"].(map[string]any)
		u.Access, u.Refresh = tokens["access_token"].(string), tokens["refresh_token"].(string)
	}
	return status
}

func TestAccountErasure(t *testing.T) {
	a := newTestAPI(t)
	ctx := context.Background()
	svc := privacy.NewService(a.pool, nil, privacy.Files{Store: a.store, PrivateBucket: a.cfg.StoragePrivateBucket, PublicBucket: a.cfg.StoragePublicBucket, PublicBaseURL: a.cfg.StoragePublicBaseURL})
	admin := a.register("erasure-admin")
	a.grantRoles(&admin, "admin")
	user := a.register("erasure-user")
	reader := a.register("erasure-reader")
	target := a.createTarget(admin, "Erasure Cafe", catRestaurantID, "cafe")
	review := a.review(user, target, 2, map[string]any{"social_media_url": "https://www.tiktok.com/@erasure-user"})
	photo, thumb := a.uploadPhoto(user, review)
	status, res := a.do("POST", "/api/v1/reviews/"+review+"/evidence",
		map[string]any{"kind": "receipt", "content_type": "image/jpeg"}, user.Access)
	require.Equal(t, http.StatusCreated, status, "%v", res)
	evidenceKey := data(res)["upload"].(map[string]any)["key"].(string)
	receipt := exifJPEG(t)
	require.NoError(t, a.store.Put(ctx, a.cfg.StoragePrivateBucket, evidenceKey,
		bytes.NewReader(receipt), int64(len(receipt)), "image/jpeg"))
	status, _ = a.do("POST", "/api/v1/evidence/"+data(res)["upload_id"].(string)+"/finalize", nil, user.Access)
	require.Equal(t, http.StatusOK, status)
	status, _ = a.do("PUT", "/api/v1/reviews/"+review+"/helpful", nil, reader.Access)
	require.Equal(t, http.StatusOK, status)
	a.registerPush(user, pushToken("erasure"))
	statsBefore := a.stats(target)
	count := func(query string, args ...any) int {
		var n int
		require.NoError(t, a.pool.QueryRow(ctx, query, args...).Scan(&n))
		return n
	}

	t.Run("a request locks the account at once", func(t *testing.T) {
		status, res := a.do("POST", "/api/v1/users/me/erasure-requests",
			map[string]any{"reason": "Moving away."}, user.Access)
		require.Equal(t, http.StatusCreated, status, "%v", res)
		assert.Equal(t, "scheduled", data(res)["status"])
		when, err := time.Parse(time.RFC3339Nano, data(res)["scheduled_for"].(string))
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now().Add(7*24*time.Hour), when, time.Minute)

		status, _ = a.do("GET", "/api/v1/users/me", nil, user.Access)
		assert.Equal(t, http.StatusUnauthorized, status, "signed out everywhere")
		assert.Zero(t, count(`SELECT count(*) FROM device_tokens WHERE user_id = $1`, user.ID), "no more notifications")
		_, list := a.do("GET", "/api/v1/targets/"+target+"/reviews", nil, "")
		assert.Len(t, dataList(list), 1, "reviews stay up during the grace period")
	})

	t.Run("signing in during the grace period cancels it", func(t *testing.T) {
		require.Equal(t, http.StatusOK, a.login(&user))
		status, res := a.do("GET", "/api/v1/users/me/erasure-requests", nil, user.Access)
		require.Equal(t, http.StatusOK, status)
		require.Len(t, dataList(res), 1)
		assert.Equal(t, "cancelled", dataList(res)[0].(map[string]any)["status"])
		n, err := svc.RunDueErasures(ctx, 10)
		require.NoError(t, err)
		assert.Zero(t, n)
	})

	t.Run("only one active request", func(t *testing.T) {
		status, _ := a.do("POST", "/api/v1/users/me/erasure-requests", map[string]any{}, user.Access)
		require.Equal(t, http.StatusCreated, status)
		require.Equal(t, http.StatusOK, a.login(&user)) // cancels it again
		status, _ = a.do("POST", "/api/v1/users/me/erasure-requests", map[string]any{"reason": "For real."}, user.Access)
		require.Equal(t, http.StatusCreated, status)
		_, err := a.pool.Exec(ctx, `INSERT INTO data_erasure_requests (id, user_id, status, scheduled_for)
			VALUES (gen_random_uuid(), $1, 'scheduled', now())`, user.ID)
		assert.Error(t, err, "the database allows one active request")
	})

	t.Run("nothing runs before the grace period ends", func(t *testing.T) {
		n, err := svc.RunDueErasures(ctx, 10)
		require.NoError(t, err)
		assert.Zero(t, n)
	})

	t.Run("after the grace period the account is erased", func(t *testing.T) {
		_, err := a.pool.Exec(ctx, `UPDATE data_erasure_requests SET scheduled_for = now() - interval '1 minute'
			WHERE user_id = $1 AND status = 'scheduled'`, user.ID)
		require.NoError(t, err)
		n, err := svc.RunDueErasures(ctx, 10)
		require.NoError(t, err)
		require.Equal(t, 1, n)

		var name, status string
		var email, phone, hash *string
		require.NoError(t, a.pool.QueryRow(ctx, `SELECT display_name, email, phone, password_hash, status
			FROM users WHERE id = $1`, user.ID).Scan(&name, &email, &phone, &hash, &status))
		assert.Equal(t, "erased", status)
		assert.Equal(t, privacy.FormerMemberName, name)
		assert.Nil(t, email)
		assert.Nil(t, phone)
		assert.Nil(t, hash)
		assert.Equal(t, http.StatusUnauthorized, a.login(&user), "the account can't be signed into")

		for table, query := range map[string]string{
			"sessions":      `SELECT count(*) FROM auth_sessions WHERE user_id = $1`,
			"notifications": `SELECT count(*) FROM notifications WHERE user_id = $1`,
			"push tokens":   `SELECT count(*) FROM device_tokens WHERE user_id = $1`,
			"photos":        `SELECT count(*) FROM review_media m JOIN reviews r ON r.id = m.review_id WHERE r.user_id = $1`,
			"evidence":      `SELECT count(*) FROM review_evidence e JOIN reviews r ON r.id = e.review_id WHERE r.user_id = $1`,
			"profile links": `SELECT count(*) FROM reviews WHERE user_id = $1 AND social_media_url <> ''`,
			"reasons":       `SELECT count(*) FROM data_erasure_requests WHERE user_id = $1 AND reason <> ''`,
		} {
			assert.Zero(t, count(query, user.ID), table)
		}
		assert.Equal(t, 1, count(`SELECT count(*) FROM data_erasure_request_events e
			JOIN data_erasure_requests r ON r.id = e.request_id
			WHERE r.user_id = $1 AND e.action = 'completed' AND e.actor_id IS NULL`, user.ID), "recorded as done by the system")
	})

	t.Run("the review stays, by a former member", func(t *testing.T) {
		_, list := a.do("GET", "/api/v1/targets/"+target+"/reviews", nil, "")
		require.Len(t, dataList(list), 1)
		rv := dataList(list)[0].(map[string]any)
		assert.Equal(t, review, rv["id"])
		assert.Equal(t, true, rv["reviewer_erased"])
		assert.Nil(t, rv["media"])
		assert.NotContains(t, rv, "social_media_url")
		assert.EqualValues(t, 1, rv["helpful_count"], "votes on it are kept")
		assert.Equal(t, statsBefore["review_count"], a.stats(target)["review_count"], "ratings unchanged")

		_, body := a.page("/t/" + a.slugOf(target))
		assert.Contains(t, body, "Former member")
		assert.NotContains(t, body, "Test erasure-user")
	})

	t.Run("its files are deleted", func(t *testing.T) {
		n, err := svc.PurgeQueuedObjects(ctx, 100)
		require.NoError(t, err)
		assert.Equal(t, 3, n, "photo, thumbnail, receipt")
		assert.False(t, a.objectIn(a.cfg.StoragePublicBucket, photo))
		assert.False(t, a.objectIn(a.cfg.StoragePublicBucket, thumb))
		assert.Zero(t, count(`SELECT count(*) FROM pending_object_deletions`))
		assert.False(t, a.objectIn(a.cfg.StoragePrivateBucket, evidenceKey))
	})

	t.Run("admins can see it was done", func(t *testing.T) {
		status, res := a.do("GET", "/api/v1/admin/privacy/erasure-requests?status=completed", nil, admin.Access)
		require.Equal(t, http.StatusOK, status, "%v", res)
		require.Len(t, dataList(res), 1)
		assert.NotNil(t, dataList(res)[0].(map[string]any)["completed_at"])
	})
}

// A photo already moved out of public view (its review was removed) is
// deleted from where it actually is.
func TestErasureFindsWithheldPhotos(t *testing.T) {
	a := newTestAPI(t)
	ctx := context.Background()
	svc := privacy.NewService(a.pool, nil, privacy.Files{Store: a.store, PrivateBucket: a.cfg.StoragePrivateBucket, PublicBucket: a.cfg.StoragePublicBucket, PublicBaseURL: a.cfg.StoragePublicBaseURL})
	mod := a.register("withheld-mod")
	a.grantRoles(&mod, "moderator")
	user := a.register("withheld-user")
	target := a.createTarget(mod, "Withheld Erasure Cafe", catRestaurantID, "cafe")
	review := a.review(user, target, 3, nil)
	photo, thumb := a.uploadPhoto(user, review)
	status, _ := a.do("DELETE", "/api/v1/reviews/"+review, nil, user.Access)
	require.Equal(t, http.StatusOK, status)
	_, err := media.NewService(a.pool, a.store, nil, a.cfg.StoragePrivateBucket, a.cfg.StoragePublicBucket).
		SyncVisibility(ctx, 10)
	require.NoError(t, err)
	require.True(t, a.objectIn(a.cfg.StoragePrivateBucket, "withheld/"+photo))

	status, _ = a.do("POST", "/api/v1/users/me/erasure-requests", map[string]any{}, user.Access)
	require.Equal(t, http.StatusCreated, status)
	_, err = a.pool.Exec(ctx, `UPDATE data_erasure_requests SET scheduled_for = now() WHERE user_id = $1`, user.ID)
	require.NoError(t, err)
	n, err := svc.RunDueErasures(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	n, err = svc.PurgeQueuedObjects(ctx, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, n)
	assert.False(t, a.objectIn(a.cfg.StoragePrivateBucket, "withheld/"+photo))
	assert.False(t, a.objectIn(a.cfg.StoragePrivateBucket, "withheld/"+thumb))
}
