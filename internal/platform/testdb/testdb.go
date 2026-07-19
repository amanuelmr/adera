// Package testdb provisions a disposable PostgreSQL database per test,
// migrated to the current schema. Integration tests run against real
// PostgreSQL, never mocks.
//
// Configuration: TEST_DATABASE_ADMIN_URL points at a superuser-ish connection
// (default matches docker-compose / the dev container:
// postgres://adera:adera@localhost:55432/postgres). Tests are skipped when the
// server is unreachable so `go test ./...` still passes on machines without
// Docker; CI always provides the database.
package testdb

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adera-platform/backend/internal/platform/database"
	"github.com/adera-platform/backend/migrations"
)

// New creates a migrated, disposable database and returns a pool bound to it.
// The database is dropped when the test finishes.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()
	adminURL := os.Getenv("TEST_DATABASE_ADMIN_URL")
	if adminURL == "" {
		adminURL = "postgres://adera:adera@localhost:55432/postgres?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Skipf("test database unavailable (set TEST_DATABASE_ADMIN_URL or start docker compose): %v", err)
	}

	var suffix [6]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatalf("generating db suffix: %v", err)
	}
	name := "adera_test_" + hex.EncodeToString(suffix[:])
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		_ = admin.Close(ctx)
		t.Fatalf("creating test database: %v", err)
	}

	u, err := url.Parse(adminURL)
	if err != nil {
		t.Fatalf("parsing admin url: %v", err)
	}
	u.Path = "/" + name
	testURL := u.String()

	pool, err := database.Connect(ctx, testURL, 8)
	if err != nil {
		t.Fatalf("connecting to test database: %v", err)
	}
	if err := database.Migrate(ctx, pool, migrations.FS); err != nil {
		t.Fatalf("migrating test database: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		dropCtx, dropCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer dropCancel()
		if _, err := admin.Exec(dropCtx, fmt.Sprintf("DROP DATABASE %s WITH (FORCE)", name)); err != nil {
			t.Logf("dropping test database %s: %v", name, err)
		}
		_ = admin.Close(dropCtx)
	})
	return pool
}
