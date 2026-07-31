package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/config"
	"github.com/adera-platform/backend/internal/platform/storage"
	"github.com/adera-platform/backend/internal/platform/testdb"
)

// captureProvider records OTP codes so tests can complete verification flows.
type captureProvider struct {
	mu    sync.Mutex
	codes map[string]string // destination -> last code
}

func (p *captureProvider) SendCode(_ context.Context, _, destination, _, code string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.codes[destination] = code
	return nil
}

func (p *captureProvider) last(destination string) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.codes[destination]
}

type testAPI struct {
	t     *testing.T
	srv   *httptest.Server
	pool  *pgxpool.Pool
	store *storage.MemoryStore
	codes *captureProvider
	cfg   config.Config
}

func testConfig() config.Config {
	return config.Config{
		Env:                  config.EnvTest,
		JWTSecret:            "integration-test-secret-key-32-bytes!!",
		AccessTokenTTL:       15 * time.Minute,
		RefreshTokenTTL:      30 * 24 * time.Hour,
		DatabaseQueryTimeout: 30 * time.Second,
		ArgonMemoryKiB:       19456,
		ArgonIterations:      2,
		ArgonParallelism:     1,
		StoragePublicBucket:  "test-public",
		StoragePrivateBucket: "test-private",
		StoragePublicBaseURL: "http://storage.test/public",
		CORSAllowedOrigins:   []string{"http://localhost:3000"},
		RateLimitEnabled:     false,
	}
}

func newTestAPI(t *testing.T) *testAPI {
	t.Helper()
	pool := testdb.New(t)
	codes := &captureProvider{codes: map[string]string{}}
	store := storage.NewMemory()
	cfg := testConfig()
	srv := httptest.NewServer(BuildAPI(cfg, pool, store, codes))
	t.Cleanup(srv.Close)
	return &testAPI{t: t, srv: srv, pool: pool, store: store, codes: codes, cfg: cfg}
}

// do performs a JSON request and decodes the response body generically.
func (a *testAPI) do(method, path string, body any, token string, headers ...string) (int, map[string]any) {
	a.t.Helper()
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		require.NoError(a.t, err)
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, a.srv.URL+path, rd)
	require.NoError(a.t, err)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	resp, err := a.srv.Client().Do(req)
	require.NoError(a.t, err)
	defer resp.Body.Close()
	var out map[string]any
	raw, err := io.ReadAll(resp.Body)
	require.NoError(a.t, err)
	if len(raw) > 0 {
		require.NoError(a.t, json.Unmarshal(raw, &out), "body: %s", raw)
	}
	return resp.StatusCode, out
}

func data(res map[string]any) map[string]any {
	d, _ := res["data"].(map[string]any)
	return d
}

func dataList(res map[string]any) []any {
	d, _ := res["data"].([]any)
	return d
}

type testUser struct {
	ID      string
	Email   string
	Access  string
	Refresh string
}

var userSeq int
var userSeqMu sync.Mutex

// register creates a user through the real registration endpoint.
func (a *testAPI) register(name string) testUser {
	a.t.Helper()
	userSeqMu.Lock()
	userSeq++
	n := userSeq
	userSeqMu.Unlock()
	email := fmt.Sprintf("%s-%d-%s@example.com", name, n, uuid.New().String()[:8])
	status, res := a.do("POST", "/api/v1/auth/register", map[string]any{
		"display_name": "Test " + name,
		"email":        email,
		"password":     "password123",
	}, "")
	require.Equal(a.t, http.StatusCreated, status, "register: %v", res)
	d := data(res)
	tokens := d["tokens"].(map[string]any)
	return testUser{
		ID:      d["user"].(map[string]any)["id"].(string),
		Email:   email,
		Access:  tokens["access_token"].(string),
		Refresh: tokens["refresh_token"].(string),
	}
}

// grantRoles assigns roles directly in the database (the bootstrap admin in
// real deployments comes from the seed), then refreshes the login so the
// access token carries them.
func (a *testAPI) grantRoles(u *testUser, roles ...string) {
	a.t.Helper()
	for _, role := range roles {
		_, err := a.pool.Exec(context.Background(), `
			INSERT INTO user_roles (user_id, role) VALUES ($1, $2) ON CONFLICT DO NOTHING`, u.ID, role)
		require.NoError(a.t, err)
	}
	status, res := a.do("POST", "/api/v1/auth/login", map[string]any{
		"identifier": u.Email, "password": "password123",
	}, "")
	require.Equal(a.t, http.StatusOK, status, "re-login: %v", res)
	tokens := data(res)["tokens"].(map[string]any)
	u.Access = tokens["access_token"].(string)
	u.Refresh = tokens["refresh_token"].(string)
}

// Category and city IDs fixed by migrations/0010_reference_data.sql.
const (
	catRestaurantID  = "11111111-1111-4111-8111-111111111104"
	catElectronicsID = "11111111-1111-4111-8111-111111111101"
	addisCityID      = "22222222-2222-4222-8222-222222222201"
)

// createTarget makes a published target via the API as a moderator.
func (a *testAPI) createTarget(mod testUser, name, categoryID, targetType string, aliases ...string) string {
	a.t.Helper()
	status, res := a.do("POST", "/api/v1/targets", map[string]any{
		"target_type": targetType,
		"category_id": categoryID,
		"name":        name,
		"city_id":     addisCityID,
		"aliases":     aliases,
	}, mod.Access)
	require.Equal(a.t, http.StatusCreated, status, "create target: %v", res)
	require.Equal(a.t, "published", data(res)["moderation_status"])
	return data(res)["id"].(string)
}

// review posts a review and returns its id.
func (a *testAPI) review(u testUser, targetID string, rating int, extra map[string]any) string {
	a.t.Helper()
	body := map[string]any{
		"target_id":      targetID,
		"overall_rating": rating,
		"body":           "An integration test review body that is long enough.",
	}
	for k, v := range extra {
		body[k] = v
	}
	status, res := a.do("POST", "/api/v1/reviews", body, u.Access)
	require.Equal(a.t, http.StatusCreated, status, "create review: %v", res)
	return data(res)["id"].(string)
}

// stats fetches the target's aggregate stats.
func (a *testAPI) stats(targetID string) map[string]any {
	a.t.Helper()
	status, res := a.do("GET", "/api/v1/targets/"+targetID+"/stats", nil, "")
	require.Equal(a.t, http.StatusOK, status, "stats: %v", res)
	return data(res)
}
