package businesses

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/testdb"
	"github.com/adera-platform/backend/internal/platform/web"
)

func TestSlugify(t *testing.T) {
	id := uuid.MustParse("11111111-2222-4333-8444-555555555555")
	suffix := "11111111"

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple name", "Kategna", "kategna-" + suffix},
		{"lowercases and dashes punctuation", "Bole Cafe & Bar!", "bole-cafe-bar-" + suffix},
		{"collapses repeated separators", "A -- B", "a-b-" + suffix},
		{"non-latin name falls back to id", "ቦሌ ካፌ", "t-" + suffix},
		{"empty name falls back to id", "", "t-" + suffix},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Slugify(tt.in, id))
		})
	}
}

func TestSlugifyTruncatesLongNames(t *testing.T) {
	id := uuid.New()
	long := strings.Repeat("a", 200)

	got := Slugify(long, id)

	base := strings.TrimSuffix(got, "-"+strings.Split(id.String(), "-")[0])
	assert.LessOrEqual(t, len(base), 100)
}

func TestRepoCreateValidatesNameAndDescription(t *testing.T) {
	pool := testdb.New(t)
	repo := NewRepo(pool, nil)
	ctx := context.Background()
	creator := seedUser(t, pool)

	_, err := repo.Create(ctx, "A", "", creator)
	assertValidationError(t, err, "name")

	_, err = repo.Create(ctx, strings.Repeat("a", 161), "", creator)
	assertValidationError(t, err, "name")

	_, err = repo.Create(ctx, "Valid Name", strings.Repeat("a", 2001), creator)
	assertValidationError(t, err, "description")

	b, err := repo.Create(ctx, "  Kategna  ", "  Great place  ", creator)
	require.NoError(t, err)
	assert.Equal(t, "Kategna", b.Name)
	assert.Equal(t, "Great place", b.Description)
	assert.Equal(t, "unverified", b.VerificationStatus)
}

func TestRepoGetAndUpdateDescription(t *testing.T) {
	pool := testdb.New(t)
	repo := NewRepo(pool, nil)
	ctx := context.Background()
	creator := seedUser(t, pool)

	b, err := repo.Create(ctx, "Kategna", "original", creator)
	require.NoError(t, err)

	fetched, err := repo.Get(ctx, b.ID)
	require.NoError(t, err)
	assert.Equal(t, b.ID, fetched.ID)

	updated, err := repo.UpdateDescription(ctx, b.ID, "updated description")
	require.NoError(t, err)
	assert.Equal(t, "updated description", updated.Description)

	_, err = repo.UpdateDescription(ctx, b.ID, strings.Repeat("a", 2001))
	assertValidationError(t, err, "description")
}

func TestRepoGetNotFound(t *testing.T) {
	pool := testdb.New(t)
	repo := NewRepo(pool, nil)

	_, err := repo.Get(context.Background(), uuid.New())

	var webErr *web.Error
	require.ErrorAs(t, err, &webErr)
	assert.Equal(t, 404, webErr.Status)
}

func TestRepoIsMember(t *testing.T) {
	pool := testdb.New(t)
	repo := NewRepo(pool, nil)
	ctx := context.Background()
	creator := seedUser(t, pool)
	b, err := repo.Create(ctx, "Kategna", "", creator)
	require.NoError(t, err)

	isMember, err := repo.IsMember(ctx, b.ID, creator)
	require.NoError(t, err)
	assert.False(t, isMember, "creating a business does not grant membership")

	_, err = pool.Exec(ctx, `
		INSERT INTO business_members (business_id, user_id, role) VALUES ($1, $2, 'owner')`, b.ID, creator)
	require.NoError(t, err)

	isMember, err = repo.IsMember(ctx, b.ID, creator)
	require.NoError(t, err)
	assert.True(t, isMember)
}

func assertValidationError(t *testing.T, err error, field string) {
	t.Helper()
	var webErr *web.Error
	require.ErrorAs(t, err, &webErr)
	assert.Equal(t, 422, webErr.Status)
	_, hasDetail := webErr.Details[field]
	assert.True(t, hasDetail, "expected a validation detail for %q, got %v", field, webErr.Details)
}

// seedUser inserts a minimal user row to satisfy businesses.created_by's
// foreign key, since this package owns no user-creation logic itself.
func seedUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO users (id, display_name, email) VALUES ($1, 'Test Owner', $2)`,
		id, id.String()+"@example.com")
	require.NoError(t, err)
	return id
}
