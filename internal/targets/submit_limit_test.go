package targets

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/adera-platform/backend/internal/platform/ratelimit"
	"github.com/adera-platform/backend/internal/platform/web"
)

// Place submissions land in the moderation queue; past the per-account
// limit they're refused before anything is parsed or stored (the handler
// has no repo here, so reaching it would panic).
func TestPlaceSubmissionsAreRateLimitedPerAccount(t *testing.T) {
	limiter := ratelimit.NewKeyed(0, 1)
	h := NewHandler(nil, nil, nil, limiter)
	user := uuid.New()
	limiter.Allow("target-submit:" + user.String()) // spend this account's only token

	req := httptest.NewRequest(http.MethodPost, "/api/v1/targets", strings.NewReader(`{"name":"x"}`))
	req = req.WithContext(web.ContextWithPrincipal(req.Context(), web.Principal{UserID: user}))
	rec := httptest.NewRecorder()
	h.create(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestModeratorsAreNotRateLimitedOnPlaceSubmissions(t *testing.T) {
	limiter := ratelimit.NewKeyed(0, 1)
	h := NewHandler(nil, nil, nil, limiter)
	mod := uuid.New()
	limiter.Allow("target-submit:" + mod.String())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/targets", strings.NewReader(`{"name":""}`))
	req = req.WithContext(web.ContextWithPrincipal(req.Context(), web.Principal{UserID: mod, Roles: []string{web.RoleModerator}}))
	rec := httptest.NewRecorder()
	h.create(rec, req)

	// Past the limiter: the empty name is rejected by validation instead.
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}
