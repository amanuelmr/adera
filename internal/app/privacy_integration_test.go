package app

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrivacyExportAndErasureWorkflow(t *testing.T) {
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
	assert.Equal(t, "1", export["schema_version"])
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

	status, res = a.do("POST", "/api/v1/users/me/erasure-requests",
		map[string]any{"reason": "I no longer plan to use this account."}, user.Access)
	require.Equal(t, http.StatusCreated, status, "%v", res)
	requestID := data(res)["id"].(string)
	assert.Equal(t, "pending", data(res)["status"])

	status, _ = a.do("POST", "/api/v1/users/me/erasure-requests", map[string]any{}, user.Access)
	assert.Equal(t, http.StatusConflict, status, "only one active request is allowed")

	status, res = a.do("GET", "/api/v1/users/me/erasure-requests", nil, user.Access)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, dataList(res), 1)
	assert.Equal(t, requestID, dataList(res)[0].(map[string]any)["id"])

	stranger := a.register("privacy-stranger")
	status, _ = a.do("DELETE", "/api/v1/users/me/erasure-requests/"+requestID, nil, stranger.Access)
	assert.Equal(t, http.StatusNotFound, status, "foreign requests must look nonexistent")

	status, res = a.do("GET", "/api/v1/admin/privacy/erasure-requests?status=pending", nil, admin.Access)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, dataList(res), 1)

	status, _ = a.do("POST", "/api/v1/admin/privacy/erasure-requests/"+requestID+"/decision",
		map[string]any{"status": "approved"}, admin.Access)
	assert.Equal(t, http.StatusConflict, status, "approval requires an explicit review step")

	status, res = a.do("POST", "/api/v1/admin/privacy/erasure-requests/"+requestID+"/decision",
		map[string]any{"status": "in_review", "note": "Identity and retention scope checked."}, admin.Access)
	require.Equal(t, http.StatusOK, status, "%v", res)
	assert.Equal(t, "in_review", data(res)["status"])

	status, res = a.do("GET", "/api/v1/users/me/notifications", nil, user.Access)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, dataList(res), 1)
	assert.Equal(t, "privacy.erasure_in_review", dataList(res)[0].(map[string]any)["event_type"])

	status, res = a.do("POST", "/api/v1/admin/privacy/erasure-requests/"+requestID+"/decision",
		map[string]any{"status": "approved", "note": "Accepted for privacy-officer fulfillment."}, admin.Access)
	require.Equal(t, http.StatusOK, status, "%v", res)
	assert.Equal(t, "approved", data(res)["status"])
	assert.NotNil(t, data(res)["decided_at"])

	status, _ = a.do("DELETE", "/api/v1/users/me/erasure-requests/"+requestID, nil, user.Access)
	assert.Equal(t, http.StatusConflict, status, "approved requests are terminal")

	var eventCount int
	err = a.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM data_erasure_request_events WHERE request_id = $1`, requestID).Scan(&eventCount)
	require.NoError(t, err)
	assert.Equal(t, 3, eventCount, "request, review, and decision events must be retained")

	cancellingUser := a.register("privacy-cancel")
	status, res = a.do("POST", "/api/v1/users/me/erasure-requests", map[string]any{}, cancellingUser.Access)
	require.Equal(t, http.StatusCreated, status)
	cancelID := data(res)["id"].(string)
	status, res = a.do("DELETE", "/api/v1/users/me/erasure-requests/"+cancelID, nil, cancellingUser.Access)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "cancelled", data(res)["status"])
	status, _ = a.do("POST", "/api/v1/admin/privacy/erasure-requests/"+cancelID+"/decision",
		map[string]any{"status": "in_review"}, admin.Access)
	assert.Equal(t, http.StatusConflict, status)

	status, res = a.do("GET", "/api/v1/users/me/data-export", nil, user.Access)
	require.Equal(t, http.StatusOK, status)
	export = data(res)
	require.Len(t, export["erasure_requests"].([]any), 1)
	assert.Equal(t, "approved", export["erasure_requests"].([]any)[0].(map[string]any)["status"])
	require.Len(t, export["notifications"].([]any), 2)
}
