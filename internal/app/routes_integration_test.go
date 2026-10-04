package app

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/config"
)

// HTTP-level coverage for routes added for the mobile client: auth wiring,
// status codes and response shapes as documented in api/openapi.yaml.

func TestDeviceRegistrationRoutes(t *testing.T) {
	a := newTestAPI(t)
	alice := a.register("devicealice")
	bob := a.register("devicebob")
	body := map[string]any{"token": "fcm-token-alice-0123456789abcdefghijklmnop", "platform": "android", "app_version": "1.0.0"}

	status, _ := a.do("POST", "/api/v1/users/me/devices", body, "")
	assert.Equal(t, http.StatusUnauthorized, status, "registration needs a session")

	status, res := a.do("GET", "/api/v1/users/me/devices", nil, alice.Access)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, []any{}, res["data"], "an empty list is [], not null")

	status, res = a.do("POST", "/api/v1/users/me/devices", body, alice.Access)
	require.Equal(t, http.StatusOK, status, "%v", res)
	status, res = a.do("POST", "/api/v1/users/me/devices", body, alice.Access)
	require.Equal(t, http.StatusOK, status, "re-registering the same token is idempotent: %v", res)
	_, res = a.do("GET", "/api/v1/users/me/devices", nil, alice.Access)
	assert.Len(t, res["data"], 1)

	status, _ = a.do("POST", "/api/v1/users/me/devices", map[string]any{"token": "x", "platform": "symbian"}, alice.Access)
	assert.Equal(t, http.StatusUnprocessableEntity, status)

	unregister := map[string]any{"token": "fcm-token-alice-0123456789abcdefghijklmnop"}
	status, _ = a.do("POST", "/api/v1/users/me/devices/unregister", unregister, bob.Access)
	assert.Equal(t, http.StatusNotFound, status, "one user can't unregister another's device")

	status, _ = a.do("POST", "/api/v1/users/me/devices/unregister", unregister, alice.Access)
	assert.Equal(t, http.StatusOK, status)
	_, res = a.do("GET", "/api/v1/users/me/devices", nil, alice.Access)
	assert.Equal(t, []any{}, res["data"])

	// The deprecated DELETE form still works for older builds.
	status, _ = a.do("POST", "/api/v1/users/me/devices", body, alice.Access)
	require.Equal(t, http.StatusOK, status)
	status, _ = a.do("DELETE", "/api/v1/users/me/devices", unregister, alice.Access)
	assert.Equal(t, http.StatusOK, status)
	status, _ = a.do("POST", "/api/v1/users/me/devices/unregister", map[string]any{}, alice.Access)
	assert.Equal(t, http.StatusUnprocessableEntity, status, "token is required")
}

func TestNearbyRoute(t *testing.T) {
	a := newTestAPI(t)
	mod := a.register("nearbymod")
	a.grantRoles(&mod, "moderator")
	status, res := a.do("POST", "/api/v1/targets", map[string]any{
		"target_type": "restaurant", "category_id": catRestaurantID, "name": "Nearby Route Café",
		"city_id": addisCityID, "latitude": 9.0300, "longitude": 38.7400,
	}, mod.Access)
	require.Equal(t, http.StatusCreated, status, "%v", res)
	id := data(res)["id"].(string)

	status, res = a.do("GET", "/api/v1/targets/nearby?lat=9.0310&lng=38.7410&radius_km=1", nil, "")
	require.Equal(t, http.StatusOK, status, "%v", res)
	found := false
	for _, item := range res["data"].([]any) {
		m := item.(map[string]any)
		if m["id"] == id {
			found = true
			assert.Less(t, m["distance_km"].(float64), 1.0)
		}
	}
	assert.True(t, found, "the target ~150 m away is in a 1 km radius")

	status, res = a.do("GET", "/api/v1/targets/nearby?lat=-60&lng=-60&radius_km=1", nil, "")
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, []any{}, res["data"], "nothing nearby is [], not null")

	for _, q := range []string{"lng=38.74", "lat=91&lng=38.74", "lat=9&lng=181", "lat=9&lng=38&radius_km=0", "lat=NaN&lng=38"} {
		status, _ = a.do("GET", "/api/v1/targets/nearby?"+q, nil, "")
		assert.Equal(t, http.StatusUnprocessableEntity, status, q)
	}
}

func TestAppVersionRoute(t *testing.T) {
	a := newTestAPIWith(t, func(c *config.Config) {
		c.AndroidMinVersion = "1.2.0"
		c.AndroidLatestVersion = "1.4.0"
		c.AndroidStoreURL = "https://play.example/adera"
	})

	status, res := a.do("GET", "/api/v1/app/version?platform=android&version=1.1.9", nil, "")
	require.Equal(t, http.StatusOK, status, "%v", res)
	assert.Equal(t, true, data(res)["update_required"])
	assert.Equal(t, "https://play.example/adera", data(res)["store_url"])

	_, res = a.do("GET", "/api/v1/app/version?platform=android&version=1.3.0", nil, "")
	assert.Equal(t, false, data(res)["update_required"])
	assert.Equal(t, true, data(res)["update_available"])

	status, _ = a.do("GET", "/api/v1/app/version?platform=android&version=v1", nil, "")
	assert.Equal(t, http.StatusUnprocessableEntity, status)
	status, _ = a.do("GET", "/api/v1/app/version?platform=blackberry", nil, "")
	assert.Equal(t, http.StatusUnprocessableEntity, status)
}

func TestMyBusinessesRoute(t *testing.T) {
	a := newTestAPI(t)
	status, _ := a.do("GET", "/api/v1/businesses/mine", nil, "")
	assert.Equal(t, http.StatusUnauthorized, status)

	u := a.register("mybiz")
	status, res := a.do("GET", "/api/v1/businesses/mine", nil, u.Access)
	require.Equal(t, http.StatusOK, status, "the literal route wins over /businesses/{id}: %v", res)
	assert.Equal(t, []any{}, res["data"])
}
