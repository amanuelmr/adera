package app

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Moderators don't decide matters about themselves: owning a business,
// verifying their own review, or restoring it. Another moderator can.
func TestModeratorsCannotDecideTheirOwnCases(t *testing.T) {
	a := newTestAPI(t)
	ctx := context.Background()
	self := a.register("selfmod")
	a.grantRoles(&self, "moderator")
	other := a.register("othermod")
	a.grantRoles(&other, "moderator")

	t.Run("own business claim", func(t *testing.T) {
		status, res := a.do("POST", "/api/v1/businesses", map[string]any{"name": "Self Claim PLC"}, self.Access)
		require.Equal(t, http.StatusCreated, status, "%v", res)
		businessID := data(res)["id"].(string)
		status, res = a.do("POST", "/api/v1/businesses/"+businessID+"/claims",
			map[string]any{"method": "document", "message": "mine"}, self.Access)
		require.Equal(t, http.StatusCreated, status, "%v", res)
		claimID := data(res)["id"].(string)

		status, _ = a.do("POST", "/api/v1/moderation/claims/"+claimID+"/decision",
			map[string]any{"decision": "approved"}, self.Access)
		assert.Equal(t, http.StatusForbidden, status)
		var members int
		require.NoError(t, a.pool.QueryRow(ctx,
			`SELECT count(*) FROM business_members WHERE business_id = $1`, businessID).Scan(&members))
		assert.Zero(t, members, "no ownership granted")

		status, res = a.do("POST", "/api/v1/moderation/claims/"+claimID+"/decision",
			map[string]any{"decision": "approved"}, other.Access)
		require.Equal(t, http.StatusOK, status, "%v", res)
	})

	target := a.createTarget(other, "Self Moderation Cafe", catRestaurantID, "restaurant")
	review := a.review(self, target, 5, nil)

	t.Run("own review evidence", func(t *testing.T) {
		status, res := a.do("POST", "/api/v1/reviews/"+review+"/evidence",
			map[string]any{"kind": "receipt", "content_type": "image/jpeg"}, self.Access)
		require.Equal(t, http.StatusCreated, status, "%v", res)
		evidenceID := data(res)["upload_id"].(string)
		key := data(res)["upload"].(map[string]any)["key"].(string)
		payload := exifJPEG(t)
		require.NoError(t, a.store.Put(ctx, a.cfg.StoragePrivateBucket, key,
			bytes.NewReader(payload), int64(len(payload)), "image/jpeg"))
		status, _ = a.do("POST", "/api/v1/evidence/"+evidenceID+"/finalize", nil, self.Access)
		require.Equal(t, http.StatusOK, status)

		status, _ = a.do("POST", "/api/v1/moderation/evidence/"+evidenceID+"/decision",
			map[string]any{"decision": "accepted"}, self.Access)
		assert.Equal(t, http.StatusForbidden, status)
		_, res = a.do("GET", "/api/v1/reviews/"+review, nil, "")
		assert.Equal(t, "unverified", data(res)["verification_level"], "not verified")

		status, _ = a.do("POST", "/api/v1/moderation/evidence/"+evidenceID+"/decision",
			map[string]any{"decision": "accepted"}, other.Access)
		require.Equal(t, http.StatusOK, status)
	})

	t.Run("own review", func(t *testing.T) {
		status, _ := a.do("POST", "/api/v1/moderation/reviews/"+review+"/decision",
			map[string]any{"action": "remove", "note": "spam"}, other.Access)
		require.Equal(t, http.StatusOK, status)

		status, _ = a.do("POST", "/api/v1/moderation/reviews/"+review+"/decision",
			map[string]any{"action": "restore"}, self.Access)
		assert.Equal(t, http.StatusForbidden, status)
		var st string
		require.NoError(t, a.pool.QueryRow(ctx, `SELECT moderation_status FROM reviews WHERE id = $1`, review).Scan(&st))
		assert.Equal(t, "removed", st, "still removed")
	})

	t.Run("unknown review is still 404", func(t *testing.T) {
		status, _ := a.do("POST", "/api/v1/moderation/reviews/00000000-0000-4000-8000-000000000000/decision",
			map[string]any{"action": "hide"}, other.Access)
		assert.Equal(t, http.StatusNotFound, status)
	})
}
