package categories

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/testdb"
	"github.com/adera-platform/backend/internal/platform/web"
)

// catElectronicsID is fixed by migrations/0010_reference_data.sql.
const catElectronicsID = "11111111-1111-4111-8111-111111111101"

func TestRepoListReturnsSeededActiveCategoriesInOrder(t *testing.T) {
	pool := testdb.New(t)
	repo := NewRepo(pool)

	cats, err := repo.List(context.Background(), false)

	require.NoError(t, err)
	require.NotEmpty(t, cats)
	for i, c := range cats {
		assert.True(t, c.Active)
		if i > 0 {
			assert.LessOrEqual(t, cats[i-1].SortOrder, c.SortOrder)
		}
	}
}

func TestRepoListExcludesInactiveUnlessRequested(t *testing.T) {
	pool := testdb.New(t)
	repo := NewRepo(pool)
	ctx := context.Background()

	inactiveID := uuid.New()
	_, err := pool.Exec(ctx, `
		INSERT INTO categories (id, code, name, sort_order, active)
		VALUES ($1, 'hidden_category', 'Hidden Category', 99, false)`, inactiveID)
	require.NoError(t, err)

	active, err := repo.List(ctx, false)
	require.NoError(t, err)
	for _, c := range active {
		assert.NotEqual(t, inactiveID, c.ID)
	}

	all, err := repo.List(ctx, true)
	require.NoError(t, err)
	var found bool
	for _, c := range all {
		if c.ID == inactiveID {
			found = true
		}
	}
	assert.True(t, found, "includeInactive=true must surface inactive categories")
}

func TestRepoGetByIDOrCode(t *testing.T) {
	pool := testdb.New(t)
	repo := NewRepo(pool)
	ctx := context.Background()
	id, err := uuid.Parse(catElectronicsID)
	require.NoError(t, err)

	byID, err := repo.GetByIDOrCode(ctx, catElectronicsID)
	require.NoError(t, err)
	assert.Equal(t, id, byID.ID)
	assert.Equal(t, "electronics_repair", byID.Code)

	byCode, err := repo.GetByIDOrCode(ctx, "electronics_repair")
	require.NoError(t, err)
	assert.Equal(t, id, byCode.ID)
}

func TestRepoGetByIDOrCodeNotFound(t *testing.T) {
	pool := testdb.New(t)
	repo := NewRepo(pool)

	_, err := repo.GetByIDOrCode(context.Background(), "does-not-exist")

	var webErr *web.Error
	require.ErrorAs(t, err, &webErr)
	assert.Equal(t, 404, webErr.Status)
}

func TestRepoCriteriaScopedToCategoryAndActive(t *testing.T) {
	pool := testdb.New(t)
	repo := NewRepo(pool)
	ctx := context.Background()
	categoryID, err := uuid.Parse(catElectronicsID)
	require.NoError(t, err)

	criteria, err := repo.Criteria(ctx, categoryID, false)

	require.NoError(t, err)
	require.NotEmpty(t, criteria)
	for _, c := range criteria {
		assert.Equal(t, categoryID, c.CategoryID)
		assert.True(t, c.Active)
	}
}

func TestRepoCriteriaForUnknownCategoryIsEmpty(t *testing.T) {
	pool := testdb.New(t)
	repo := NewRepo(pool)

	criteria, err := repo.Criteria(context.Background(), uuid.New(), false)

	require.NoError(t, err)
	assert.Empty(t, criteria)
}
