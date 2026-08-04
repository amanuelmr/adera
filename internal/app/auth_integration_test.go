package app

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthLifecycle(t *testing.T) {
	a := newTestAPI(t)
	u := a.register("lifecycle")

	t.Run("email verification with OTP", func(t *testing.T) {
		status, _ := a.do("POST", "/api/v1/auth/verify/request", map[string]any{"channel": "email"}, u.Access)
		require.Equal(t, http.StatusOK, status)
		code := a.codes.last(u.Email)
		require.NotEmpty(t, code)

		// Wrong code fails and counts an attempt.
		status, _ = a.do("POST", "/api/v1/auth/verify/confirm", map[string]any{"channel": "email", "code": "000000"}, u.Access)
		assert.Equal(t, http.StatusUnprocessableEntity, status)

		status, _ = a.do("POST", "/api/v1/auth/verify/confirm", map[string]any{"channel": "email", "code": code}, u.Access)
		require.Equal(t, http.StatusOK, status)

		status, res := a.do("GET", "/api/v1/users/me", nil, u.Access)
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, true, data(res)["email_verified"])
	})

	t.Run("refresh rotation and reuse detection", func(t *testing.T) {
		status, res := a.do("POST", "/api/v1/auth/refresh", map[string]any{"refresh_token": u.Refresh}, "")
		require.Equal(t, http.StatusOK, status)
		rotated := data(res)["refresh_token"].(string)
		require.NotEqual(t, u.Refresh, rotated)

		// Reusing the consumed token is theft: 401 and the family dies.
		status, _ = a.do("POST", "/api/v1/auth/refresh", map[string]any{"refresh_token": u.Refresh}, "")
		assert.Equal(t, http.StatusUnauthorized, status)
		status, _ = a.do("POST", "/api/v1/auth/refresh", map[string]any{"refresh_token": rotated}, "")
		assert.Equal(t, http.StatusUnauthorized, status, "rotated head must die with its family")
	})

	// Re-login after the family revocation above.
	status, res := a.do("POST", "/api/v1/auth/login", map[string]any{"identifier": u.Email, "password": "password123"}, "")
	require.Equal(t, http.StatusOK, status)
	u.Access = data(res)["tokens"].(map[string]any)["access_token"].(string)
	u.Refresh = data(res)["tokens"].(map[string]any)["refresh_token"].(string)

	t.Run("sessions list and revoke", func(t *testing.T) {
		// Second session from another "device".
		status, res := a.do("POST", "/api/v1/auth/login", map[string]any{"identifier": u.Email, "password": "password123"}, "")
		require.Equal(t, http.StatusOK, status)
		otherTokens := data(res)["tokens"].(map[string]any)
		otherAccess := otherTokens["access_token"].(string)
		otherRefresh := otherTokens["refresh_token"].(string)

		status, res = a.do("GET", "/api/v1/auth/sessions", nil, u.Access)
		require.Equal(t, http.StatusOK, status)
		sessions := dataList(res)
		require.GreaterOrEqual(t, len(sessions), 2)

		// Find a non-current session and revoke it.
		var revokeID string
		for _, s := range sessions {
			sm := s.(map[string]any)
			if sm["current"] != true {
				revokeID = sm["id"].(string)
				break
			}
		}
		require.NotEmpty(t, revokeID)
		status, _ = a.do("DELETE", "/api/v1/auth/sessions/"+revokeID, nil, u.Access)
		require.Equal(t, http.StatusOK, status)

		// A session I don't own cannot be revoked (object-level authz).
		other := a.register("other")
		status, _ = a.do("DELETE", "/api/v1/auth/sessions/"+revokeID, nil, other.Access)
		assert.Equal(t, http.StatusNotFound, status, "foreign session must look nonexistent")

		// The revoked family's refresh no longer works.
		status, _ = a.do("POST", "/api/v1/auth/refresh", map[string]any{"refresh_token": otherRefresh}, "")
		assert.Equal(t, http.StatusUnauthorized, status)
		status, _ = a.do("GET", "/api/v1/users/me", nil, otherAccess)
		assert.Equal(t, http.StatusUnauthorized, status, "revoked session access token must stop working immediately")
	})

	t.Run("change password revokes other sessions", func(t *testing.T) {
		status, res := a.do("POST", "/api/v1/auth/login", map[string]any{"identifier": u.Email, "password": "password123"}, "")
		require.Equal(t, http.StatusOK, status)
		otherRefresh := data(res)["tokens"].(map[string]any)["refresh_token"].(string)

		status, _ = a.do("POST", "/api/v1/users/me/password",
			map[string]any{"current_password": "password123", "new_password": "password456"}, u.Access)
		require.Equal(t, http.StatusOK, status)

		status, _ = a.do("POST", "/api/v1/auth/refresh", map[string]any{"refresh_token": otherRefresh}, "")
		assert.Equal(t, http.StatusUnauthorized, status)
		status, _ = a.do("POST", "/api/v1/auth/login", map[string]any{"identifier": u.Email, "password": "password456"}, "")
		assert.Equal(t, http.StatusOK, status)
	})

	t.Run("password reset flow", func(t *testing.T) {
		status, _ := a.do("POST", "/api/v1/auth/password-reset/request", map[string]any{"identifier": u.Email}, "")
		require.Equal(t, http.StatusOK, status)
		code := a.codes.last(u.Email)
		require.NotEmpty(t, code)

		status, _ = a.do("POST", "/api/v1/auth/password-reset/confirm",
			map[string]any{"identifier": u.Email, "code": code, "new_password": "password789"}, "")
		require.Equal(t, http.StatusOK, status)

		// Old password dead, new works; code is single-use.
		status, _ = a.do("POST", "/api/v1/auth/login", map[string]any{"identifier": u.Email, "password": "password456"}, "")
		assert.Equal(t, http.StatusUnauthorized, status)
		status, _ = a.do("POST", "/api/v1/auth/login", map[string]any{"identifier": u.Email, "password": "password789"}, "")
		assert.Equal(t, http.StatusOK, status)
		status, _ = a.do("POST", "/api/v1/auth/password-reset/confirm",
			map[string]any{"identifier": u.Email, "code": code, "new_password": "passwordAAA"}, "")
		assert.Equal(t, http.StatusUnprocessableEntity, status, "reset code must be single-use")

		// Unknown identifiers get the same 200 (no user enumeration).
		status, _ = a.do("POST", "/api/v1/auth/password-reset/request",
			map[string]any{"identifier": "ghost@example.com"}, "")
		assert.Equal(t, http.StatusOK, status)
	})

	t.Run("logout all", func(t *testing.T) {
		status, res := a.do("POST", "/api/v1/auth/login", map[string]any{"identifier": u.Email, "password": "password789"}, "")
		require.Equal(t, http.StatusOK, status)
		tokens := data(res)["tokens"].(map[string]any)
		access := tokens["access_token"].(string)
		refresh := tokens["refresh_token"].(string)

		status, _ = a.do("POST", "/api/v1/auth/logout-all", nil, access)
		require.Equal(t, http.StatusOK, status)
		status, _ = a.do("POST", "/api/v1/auth/refresh", map[string]any{"refresh_token": refresh}, "")
		assert.Equal(t, http.StatusUnauthorized, status)
		status, _ = a.do("GET", "/api/v1/users/me", nil, access)
		assert.Equal(t, http.StatusUnauthorized, status, "logout-all must revoke existing access token")
	})
}

func TestOTPAttemptLimit(t *testing.T) {
	a := newTestAPI(t)
	u := a.register("otp-limit")
	code := a.codes.last(u.Email)
	require.NotEmpty(t, code)

	for range 5 {
		status, _ := a.do("POST", "/api/v1/auth/verify/confirm",
			map[string]any{"channel": "email", "code": "000000"}, u.Access)
		require.Equal(t, http.StatusUnprocessableEntity, status)
	}

	status, _ := a.do("POST", "/api/v1/auth/verify/confirm",
		map[string]any{"channel": "email", "code": code}, u.Access)
	assert.Equal(t, http.StatusUnprocessableEntity, status,
		"the real code must be rejected after the attempt budget is exhausted")
}

func TestRegistrationValidation(t *testing.T) {
	a := newTestAPI(t)

	cases := []struct {
		name string
		body map[string]any
		want int
	}{
		{"missing identifier", map[string]any{"display_name": "X Y", "password": "password123"}, http.StatusUnprocessableEntity},
		{"short password", map[string]any{"display_name": "X Y", "email": "x@example.com", "password": "short"}, http.StatusUnprocessableEntity},
		{"bad email", map[string]any{"display_name": "X Y", "email": "nope", "password": "password123"}, http.StatusUnprocessableEntity},
		{"bad phone", map[string]any{"display_name": "X Y", "phone": "12", "password": "password123"}, http.StatusUnprocessableEntity},
		{"unknown field rejected", map[string]any{"display_name": "X Y", "email": "x2@example.com", "password": "password123", "role": "admin"}, http.StatusBadRequest},
	}
	for _, tc := range cases {
		status, res := a.do("POST", "/api/v1/auth/register", tc.body, "")
		assert.Equal(t, tc.want, status, "%s: %v", tc.name, res)
	}

	// Ethiopian phone registration normalizes to E.164.
	status, res := a.do("POST", "/api/v1/auth/register", map[string]any{
		"display_name": "Phone User", "phone": "0911 22 33 44", "password": "password123",
	}, "")
	require.Equal(t, http.StatusCreated, status, "%v", res)
	access := data(res)["tokens"].(map[string]any)["access_token"].(string)
	status, res = a.do("GET", "/api/v1/users/me", nil, access)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "+251911223344", data(res)["phone"])

	// Duplicate registration is a vague conflict (no enumeration of which field).
	u := a.register("dup")
	status, res = a.do("POST", "/api/v1/auth/register", map[string]any{
		"display_name": "Dup", "email": u.Email, "password": "password123",
	}, "")
	assert.Equal(t, http.StatusConflict, status)
	assert.NotContains(t, res["error"].(map[string]any)["message"], "email exists")
}

func TestProfileUpdate(t *testing.T) {
	a := newTestAPI(t)
	u := a.register("profile")

	status, res := a.do("PATCH", "/api/v1/users/me", map[string]any{
		"display_name": "አዲስ ስም", "preferred_language": "am",
	}, u.Access)
	require.Equal(t, http.StatusOK, status, "%v", res)
	assert.Equal(t, "አዲስ ስም", data(res)["display_name"])
	assert.Equal(t, "am", data(res)["preferred_language"])

	status, _ = a.do("PATCH", "/api/v1/users/me", map[string]any{"preferred_language": "xx"}, u.Access)
	assert.Equal(t, http.StatusUnprocessableEntity, status)

	// Mass assignment: unknown/privileged fields rejected outright.
	status, _ = a.do("PATCH", "/api/v1/users/me", map[string]any{"status": "admin"}, u.Access)
	assert.Equal(t, http.StatusBadRequest, status)
}

func TestAccountDeactivation(t *testing.T) {
	a := newTestAPI(t)
	u := a.register("deact")

	status, _ := a.do("DELETE", "/api/v1/users/me", nil, u.Access)
	require.Equal(t, http.StatusOK, status)

	// Refresh dead after deactivation.
	status, _ = a.do("POST", "/api/v1/auth/refresh", map[string]any{"refresh_token": u.Refresh}, "")
	assert.Equal(t, http.StatusUnauthorized, status)

	// Logging in reactivates (documented behavior).
	status, res := a.do("POST", "/api/v1/auth/login", map[string]any{"identifier": u.Email, "password": "password123"}, "")
	require.Equal(t, http.StatusOK, status)
	access := data(res)["tokens"].(map[string]any)["access_token"].(string)
	status, res = a.do("GET", "/api/v1/users/me", nil, access)
	require.Equal(t, http.StatusOK, status)
	assert.Equal(t, "active", data(res)["status"])
}
