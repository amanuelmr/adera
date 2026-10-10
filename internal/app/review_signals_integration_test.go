package app

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/config"
	"github.com/adera-platform/backend/internal/reviews"
)

// reviewFrom posts a review as if sent from the given address and app install.
func (a *testAPI) reviewFrom(u testUser, targetID, ip, installID string) string {
	a.t.Helper()
	headers := []string{"X-Forwarded-For", ip}
	if installID != "" {
		headers = append(headers, reviews.InstallIDHeader, installID)
	}
	status, res := a.do("POST", "/api/v1/reviews", map[string]any{
		"target_id":      targetID,
		"overall_rating": 5,
		"body":           "An integration test review body that is long enough.",
	}, u.Access, headers...)
	require.Equal(a.t, http.StatusCreated, status, "create review: %v", res)
	return data(res)["id"].(string)
}

type signalGroup struct {
	kind  string
	users []string
}

func (a *testAPI) signalGroups(mod testUser, targetID string) []signalGroup {
	a.t.Helper()
	status, res := a.do("GET", "/api/v1/moderation/targets/"+targetID+"/review-signals", nil, mod.Access)
	require.Equal(a.t, http.StatusOK, status, "%v", res)
	var out []signalGroup
	for _, item := range dataList(res) {
		g := item.(map[string]any)
		assert.Len(a.t, g["key"], 12)
		sg := signalGroup{kind: g["kind"].(string)}
		for _, rv := range g["reviews"].([]any) {
			r := rv.(map[string]any)
			assert.NotEmpty(a.t, r["account_created_at"])
			assert.NotEmpty(a.t, r["reviewer_name"])
			sg.users = append(sg.users, r["user_id"].(string))
		}
		out = append(out, sg)
	}
	return out
}

func TestReviewSignals(t *testing.T) {
	a := newTestAPIWith(t, func(c *config.Config) {
		c.TrustProxyHeaders = true
		c.SignalHashKey = "integration-test-signal-key-32-bytes!!"
	})
	mod := a.register("signalsmod")
	a.grantRoles(&mod, "moderator")
	target := a.createTarget(mod, "Signals Cafe", catRestaurantID, "restaurant")
	alice, bob := a.register("signalsalice"), a.register("signalsbob")
	carol, dan := a.register("signalscarol"), a.register("signalsdan")
	erin := a.register("signalserin")

	const phone = "6f1c2b9e-8d43-4a57-9b0e-2f6c1d3a4b5c"
	a.reviewFrom(alice, target, "10.0.1.5", phone)
	a.reviewFrom(bob, target, "10.0.2.9", phone) // same phone, different network
	a.reviewFrom(carol, target, "10.0.3.7", "0d3e5f7a-1b2c-4d5e-8f90-a1b2c3d4e5f6")
	a.reviewFrom(dan, target, "10.0.3.200", "9a8b7c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d") // same /24 as carol
	a.reviewFrom(erin, target, "10.0.9.1", "not-a-uuid")                            // malformed install ID ignored

	groups := a.signalGroups(mod, target)
	require.Len(t, groups, 2, "%v", groups)
	assert.Equal(t, signalGroup{kind: "device", users: []string{alice.ID, bob.ID}}, groups[0])
	assert.Equal(t, signalGroup{kind: "network", users: []string{carol.ID, dan.ID}}, groups[1])

	t.Run("only moderators can see signals", func(t *testing.T) {
		status, _ := a.do("GET", "/api/v1/moderation/targets/"+target+"/review-signals", nil, alice.Access)
		assert.Equal(t, http.StatusForbidden, status)
	})

	t.Run("signals are hashes, never addresses", func(t *testing.T) {
		var rows, hashed int
		require.NoError(t, a.pool.QueryRow(context.Background(), `
			SELECT count(*), count(*) FILTER (WHERE octet_length(network_hash) = 32)
			FROM review_signals WHERE target_id = $1`, target).Scan(&rows, &hashed))
		assert.Equal(t, 5, rows)
		assert.Equal(t, 5, hashed)
		var withInstall int
		require.NoError(t, a.pool.QueryRow(context.Background(), `
			SELECT count(*) FROM review_signals WHERE target_id = $1 AND install_hash IS NOT NULL`, target).
			Scan(&withInstall))
		assert.Equal(t, 4, withInstall, "erin's malformed install ID is not stored")
	})

	t.Run("expired signals are purged", func(t *testing.T) {
		_, err := a.pool.Exec(context.Background(), `
			UPDATE review_signals SET created_at = now() - interval '91 days'
			WHERE review_id IN (SELECT id FROM reviews WHERE user_id = $1)`, alice.ID)
		require.NoError(t, err)
		purged, err := reviews.NewRepo(a.pool, "", nil).PurgeExpiredSignals(context.Background())
		require.NoError(t, err)
		assert.Equal(t, int64(1), purged)

		groups := a.signalGroups(mod, target)
		require.Len(t, groups, 1, "bob alone no longer forms a device group")
		assert.Equal(t, "network", groups[0].kind)
	})
}

func TestReviewSignalsOffWithoutKey(t *testing.T) {
	a := newTestAPI(t)
	mod := a.register("nosignalsmod")
	a.grantRoles(&mod, "moderator")
	target := a.createTarget(mod, "No Signals Cafe", catRestaurantID, "restaurant")
	a.reviewFrom(a.register("nosignalsa"), target, "10.0.1.5", "6f1c2b9e-8d43-4a57-9b0e-2f6c1d3a4b5c")

	var rows int
	require.NoError(t, a.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM review_signals WHERE target_id = $1`, target).Scan(&rows))
	assert.Zero(t, rows)
	assert.Empty(t, a.signalGroups(mod, target))
}
