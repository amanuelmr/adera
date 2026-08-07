package locations

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/testdb"
)

// addisCityID is fixed by migrations/0010_reference_data.sql.
const addisCityID = "22222222-2222-4222-8222-222222222201"

func TestRepoCitiesReturnsSeededActiveCitiesInOrder(t *testing.T) {
	pool := testdb.New(t)
	repo := NewRepo(pool)

	cities, err := repo.Cities(context.Background())

	require.NoError(t, err)
	require.NotEmpty(t, cities)
	var names []string
	for i, c := range cities {
		names = append(names, c.Name)
		if i > 0 {
			assert.LessOrEqual(t, cities[i-1].SortOrder, c.SortOrder, "cities must be sorted by sort_order")
		}
	}
	assert.Contains(t, names, "Addis Ababa")
}

func TestRepoCitiesExcludesInactive(t *testing.T) {
	pool := testdb.New(t)
	repo := NewRepo(pool)
	ctx := context.Background()

	inactiveID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO cities (id, name, name_am, active) VALUES ($1, 'Hidden City', 'ተደብቆ', false)`, inactiveID)
	require.NoError(t, err)

	cities, err := repo.Cities(ctx)

	require.NoError(t, err)
	for _, c := range cities {
		assert.NotEqual(t, inactiveID, c.ID, "inactive cities must not be listed")
	}
}

func TestRepoAreasReturnsSeededAreasForCity(t *testing.T) {
	pool := testdb.New(t)
	repo := NewRepo(pool)
	cityID, err := uuid.Parse(addisCityID)
	require.NoError(t, err)

	areas, err := repo.Areas(context.Background(), cityID)

	require.NoError(t, err)
	require.NotEmpty(t, areas)
	for _, a := range areas {
		assert.Equal(t, cityID, a.CityID)
	}
}

func TestRepoAreasExcludesInactiveAndOtherCities(t *testing.T) {
	pool := testdb.New(t)
	repo := NewRepo(pool)
	ctx := context.Background()
	cityID, err := uuid.Parse(addisCityID)
	require.NoError(t, err)

	inactiveID := uuid.New()
	_, err = pool.Exec(ctx, `
		INSERT INTO areas (id, city_id, name, name_am, active) VALUES ($1, $2, 'Hidden Area', 'ተደብቆ', false)`,
		inactiveID, cityID)
	require.NoError(t, err)

	areas, err := repo.Areas(ctx, cityID)

	require.NoError(t, err)
	for _, a := range areas {
		assert.NotEqual(t, inactiveID, a.ID, "inactive areas must not be listed")
	}
}

func TestRepoAreasForUnknownCityIsEmpty(t *testing.T) {
	pool := testdb.New(t)
	repo := NewRepo(pool)

	areas, err := repo.Areas(context.Background(), uuid.New())

	require.NoError(t, err)
	assert.Empty(t, areas)
}
