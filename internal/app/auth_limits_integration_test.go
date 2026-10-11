package app

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/config"
	"github.com/adera-platform/backend/internal/platform/security"
	"github.com/adera-platform/backend/internal/platform/web"
)

// Writing a phone number another way is the same account, so it must not
// get a fresh rate-limit bucket.
func TestAccountLimitIgnoresIdentifierSpelling(t *testing.T) {
	a := newTestAPIWith(t, func(c *config.Config) {
		c.RateLimitEnabled = true
		c.ClientIPSource = web.IPSource{Header: "Cf-Connecting-Ip"}
	})
	spellings := []string{"0911000001", "+251911000001", "+251 911 000 001", "+251-911-000-001", "251911000001", "(091) 100 0001"}
	var statuses []int
	for i, id := range spellings {
		// A different address each time, so only the account limit applies.
		status, _ := a.do("POST", "/api/v1/auth/login", map[string]any{"identifier": id, "password": "wrongwrong"},
			"", "CF-Connecting-IP", fmt.Sprintf("196.188.2.%d", i+1))
		statuses = append(statuses, status)
	}
	assert.Equal(t, []int{401, 401, 401, 401, 401, 429}, statuses)
}

// Many phones share one carrier address: the address limit is looser than
// the per-account one.
func TestAddressLimitAllowsSharedAddresses(t *testing.T) {
	a := newTestAPIWith(t, func(c *config.Config) { c.RateLimitEnabled = true })
	for i := range 20 {
		status, _ := a.do("POST", "/api/v1/auth/login", map[string]any{
			"identifier": fmt.Sprintf("someone-%d@example.com", i), "password": "wrongwrong",
		}, "")
		require.Equal(t, http.StatusUnauthorized, status, "attempt %d", i)
	}
}

func TestPasswordResetRequestsAreCappedPerAccount(t *testing.T) {
	a := newTestAPI(t) // in-process limits off: only the per-account cap applies
	u := a.register("resetcap")
	codes := map[string]bool{}
	for i := range 7 {
		status, res := a.do("POST", "/api/v1/auth/password-reset/request", map[string]any{"identifier": u.Email}, "")
		require.Equal(t, http.StatusOK, status, "request %d: %v", i, res)
		assert.Equal(t, "if_account_exists_code_sent", data(res)["status"], "a capped request looks the same")
		codes[a.codes.last(u.Email)] = true
	}
	var stored int
	require.NoError(t, a.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM verification_tokens WHERE user_id = $1 AND purpose = 'password_reset'`, u.ID).Scan(&stored))
	assert.Equal(t, 5, stored, "five codes a day")
	assert.LessOrEqual(t, len(codes), 5, "nothing sent past the cap")
}

// A copy of the codes table mustn't give codes away: a plain digest of six
// digits is reversed by trying all million.
func TestOneTimeCodesAreStoredKeyed(t *testing.T) {
	a := newTestAPI(t)
	u := a.register("keyedcode")
	status, _ := a.do("POST", "/api/v1/auth/password-reset/request", map[string]any{"identifier": u.Email}, "")
	require.Equal(t, http.StatusOK, status)
	code := a.codes.last(u.Email)
	require.NotEmpty(t, code)

	var stored []byte
	require.NoError(t, a.pool.QueryRow(context.Background(), `
		SELECT code_hash FROM verification_tokens WHERE user_id = $1 AND purpose = 'password_reset'`, u.ID).Scan(&stored))
	assert.NotEqual(t, security.HashToken(code), stored)

	status, _ = a.do("POST", "/api/v1/auth/password-reset/confirm",
		map[string]any{"identifier": u.Email, "code": code, "new_password": "password456"}, "")
	assert.Equal(t, http.StatusOK, status, "the keyed hash still verifies")
}
