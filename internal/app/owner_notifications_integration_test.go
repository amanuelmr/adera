package app

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// notificationsFor returns the event types in the user's inbox.
func (a *testAPI) notificationsFor(u testUser) []map[string]any {
	a.t.Helper()
	status, res := a.do("GET", "/api/v1/users/me/notifications", nil, u.Access)
	require.Equal(a.t, http.StatusOK, status, "%v", res)
	var out []map[string]any
	for _, item := range res["data"].([]any) {
		out = append(out, item.(map[string]any))
	}
	return out
}

func TestOwnersAreNotifiedOfNewReviews(t *testing.T) {
	a := newTestAPI(t)
	mod := a.register("ownernotifmod")
	a.grantRoles(&mod, "moderator")
	owner := a.register("ownernotif")
	reviewer := a.register("ownernotifreviewer")

	// A business the owner manages, through the real claim flow.
	status, res := a.do("POST", "/api/v1/businesses", map[string]any{"name": "Notify Bistro PLC"}, owner.Access)
	require.Equal(t, http.StatusCreated, status, "%v", res)
	businessID := data(res)["id"].(string)
	status, res = a.do("POST", "/api/v1/businesses/"+businessID+"/claims",
		map[string]any{"method": "document", "message": "trade license attached"}, owner.Access)
	require.Equal(t, http.StatusCreated, status, "%v", res)
	status, res = a.do("POST", "/api/v1/moderation/claims/"+data(res)["id"].(string)+"/decision",
		map[string]any{"decision": "approved"}, mod.Access)
	require.Equal(t, http.StatusOK, status, "%v", res)
	status, res = a.do("POST", "/api/v1/targets", map[string]any{
		"target_type": "restaurant", "category_id": catRestaurantID, "name": "Notify Bistro Bole", "business_id": businessID,
	}, mod.Access)
	require.Equal(t, http.StatusCreated, status, "%v", res)
	targetID := data(res)["id"].(string)

	reviewID := a.review(reviewer, targetID, 2, nil)

	t.Run("the owner gets a review.received item with what the app needs to open it", func(t *testing.T) {
		var received []map[string]any
		for _, n := range a.notificationsFor(owner) {
			if n["event_type"] == "review.received" {
				received = append(received, n)
			}
		}
		require.Len(t, received, 1)
		d := received[0]["data"].(map[string]any)
		assert.Equal(t, reviewID, received[0]["subject_id"])
		assert.Equal(t, targetID, d["target_id"])
		assert.Equal(t, businessID, d["business_id"])
		assert.Equal(t, "2", d["rating"])
	})

	t.Run("the reviewer is not notified about their own review", func(t *testing.T) {
		for _, n := range a.notificationsFor(reviewer) {
			assert.NotEqual(t, "review.received", n["event_type"])
		}
	})

	t.Run("an owner reviewing their own business isn't notified about it", func(t *testing.T) {
		before := len(a.notificationsFor(owner))
		a.review(owner, targetID, 5, nil)
		assert.Len(t, a.notificationsFor(owner), before)
	})

	t.Run("a place with no business notifies no one", func(t *testing.T) {
		plain := a.createTarget(mod, "No Business Café", catRestaurantID, "restaurant")
		before := len(a.notificationsFor(owner))
		a.review(a.register("plainreviewer"), plain, 4, nil)
		assert.Len(t, a.notificationsFor(owner), before)
	})
}
