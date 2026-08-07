package search

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/testdb"
	"github.com/adera-platform/backend/internal/targets"
)

// catRestaurantID is fixed by migrations/0010_reference_data.sql.
const catRestaurantID = "11111111-1111-4111-8111-111111111104"

// seedTestUser inserts a minimal user row to satisfy created_by foreign keys.
func seedTestUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO users (id, display_name, email) VALUES ($1, 'Test Creator', $2)`,
		id, id.String()+"@example.com")
	require.NoError(t, err)
	return id
}

func seedPublishedTarget(t *testing.T, targetsRepo *targets.Repo, creatorID uuid.UUID, name string, aliases []string) targets.Target {
	t.Helper()
	catID, err := uuid.Parse(catRestaurantID)
	require.NoError(t, err)
	target, err := targetsRepo.Create(context.Background(), targets.CreateInput{
		TargetType:  "restaurant",
		CategoryID:  catID,
		Name:        name,
		Aliases:     aliases,
		CreatedBy:   creatorID,
		AutoPublish: true,
	})
	require.NoError(t, err)
	return target
}

func TestRepoSearchMatchesByName(t *testing.T) {
	pool := testdb.New(t)
	creatorID := seedTestUser(t, pool)
	targetsRepo := targets.NewRepo(pool)
	repo := NewRepo(pool)

	seedPublishedTarget(t, targetsRepo, creatorID, "Sheger Fish and Kitfo", nil)

	results, err := repo.Search(context.Background(), "sheger", targets.BrowseFilter{}, 10)

	require.NoError(t, err)
	require.NotEmpty(t, results)
	assert.Equal(t, "Sheger Fish and Kitfo", results[0].Name)
}

func TestRepoSearchMatchesByAlias(t *testing.T) {
	pool := testdb.New(t)
	creatorID := seedTestUser(t, pool)
	targetsRepo := targets.NewRepo(pool)
	repo := NewRepo(pool)

	seedPublishedTarget(t, targetsRepo, creatorID, "Kategna Traditional Restaurant", []string{"Kategna"})

	results, err := repo.Search(context.Background(), "kategna", targets.BrowseFilter{}, 10)

	require.NoError(t, err)
	require.NotEmpty(t, results)
	assert.Equal(t, "Kategna Traditional Restaurant", results[0].Name)
}

func TestRepoSearchFiltersByCategory(t *testing.T) {
	pool := testdb.New(t)
	creatorID := seedTestUser(t, pool)
	targetsRepo := targets.NewRepo(pool)
	repo := NewRepo(pool)

	seedPublishedTarget(t, targetsRepo, creatorID, "Unique Zylophone Cafe", nil)
	otherCat := uuid.MustParse("11111111-1111-4111-8111-111111111101") // electronics_repair

	results, err := repo.Search(context.Background(), "zylophone", targets.BrowseFilter{CategoryID: &otherCat}, 10)

	require.NoError(t, err)
	assert.Empty(t, results, "a restaurant must not match an electronics category filter")
}

func TestRepoSearchExcludesUnpublishedTargets(t *testing.T) {
	pool := testdb.New(t)
	creatorID := seedTestUser(t, pool)
	targetsRepo := targets.NewRepo(pool)
	repo := NewRepo(pool)

	catID := uuid.MustParse(catRestaurantID)
	_, err := targetsRepo.Create(context.Background(), targets.CreateInput{
		TargetType: "restaurant",
		CategoryID: catID,
		Name:       "Pending Zebrafish Diner",
		CreatedBy:  creatorID,
		// AutoPublish left false: stays in moderation, must not be searchable.
	})
	require.NoError(t, err)

	results, err := repo.Search(context.Background(), "zebrafish", targets.BrowseFilter{}, 10)

	require.NoError(t, err)
	assert.Empty(t, results)
}
