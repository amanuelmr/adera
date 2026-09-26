package targets

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/testdb"
)

// catRestaurantID is fixed by migrations/0010_reference_data.sql.
const catRestaurantID = "11111111-1111-4111-8111-111111111104"

func TestParseNearbyPoint(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantErr    bool
		wantRadius float64
	}{
		{name: "lat and lng with default radius", query: "?lat=9.0&lng=38.75", wantRadius: DefaultNearbyRadiusKm},
		{name: "explicit radius", query: "?lat=9.0&lng=38.75&radius_km=12.5", wantRadius: 12.5},
		{name: "missing lat", query: "?lng=38.75", wantErr: true},
		{name: "missing lng", query: "?lat=9.0", wantErr: true},
		{name: "lat out of range", query: "?lat=91&lng=38.75", wantErr: true},
		{name: "lng out of range", query: "?lat=9.0&lng=181", wantErr: true},
		{name: "lat not a number", query: "?lat=north&lng=38.75", wantErr: true},
		{name: "radius zero", query: "?lat=9.0&lng=38.75&radius_km=0", wantErr: true},
		{name: "radius negative", query: "?lat=9.0&lng=38.75&radius_km=-3", wantErr: true},
		{name: "radius over the maximum", query: "?lat=9.0&lng=38.75&radius_km=500", wantErr: true},
		{name: "radius at the maximum", query: "?lat=9.0&lng=38.75&radius_km=50", wantRadius: MaxNearbyRadiusKm},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/api/v1/targets/nearby"+tt.query, nil)

			_, _, radius, err := ParseNearbyPoint(r)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.InDelta(t, tt.wantRadius, radius, 0.0001)
		})
	}
}

func TestBoundingBox(t *testing.T) {
	t.Run("encloses the circle at Addis latitude", func(t *testing.T) {
		minLat, maxLat, minLng, maxLng, wraps := boundingBox(9.0, 38.75, 5)

		assert.False(t, wraps)
		// 5 km is about 0.045 degrees of latitude everywhere.
		assert.InDelta(t, 9.0-0.045, minLat, 0.002)
		assert.InDelta(t, 9.0+0.045, maxLat, 0.002)
		// Longitude degrees are slightly shorter away from the equator, so
		// the longitude window must be at least as wide as the latitude one.
		assert.Greater(t, maxLng-minLng, maxLat-minLat)
	})

	t.Run("clamps latitude at the poles", func(t *testing.T) {
		minLat, maxLat, _, _, _ := boundingBox(89.99, 0, 50)

		assert.GreaterOrEqual(t, minLat, -90.0)
		assert.LessOrEqual(t, maxLat, 90.0)
	})

	t.Run("reports a window crossing the antimeridian", func(t *testing.T) {
		_, _, _, _, wraps := boundingBox(0, 179.99, 50)

		assert.True(t, wraps)
	})
}

func seedNearbyUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := pool.Exec(context.Background(), `
		INSERT INTO users (id, display_name, email) VALUES ($1, 'Test Creator', $2)`,
		id, id.String()+"@example.com")
	require.NoError(t, err)
	return id
}

func seedAt(t *testing.T, repo *Repo, creator uuid.UUID, name string, lat, lng *float64, publish bool) Target {
	t.Helper()
	catID, err := uuid.Parse(catRestaurantID)
	require.NoError(t, err)
	target, err := repo.Create(context.Background(), CreateInput{
		TargetType:  "restaurant",
		CategoryID:  catID,
		Name:        name,
		Latitude:    lat,
		Longitude:   lng,
		CreatedBy:   creator,
		AutoPublish: publish,
	})
	require.NoError(t, err)
	return target
}

func TestRepoNearby(t *testing.T) {
	pool := testdb.New(t)
	creator := seedNearbyUser(t, pool)
	repo := NewRepo(pool)

	at := func(lat, lng float64) (*float64, *float64) { return &lat, &lng }

	// Query origin; roughly Bole, Addis Ababa.
	const originLat, originLng = 9.0000, 38.7500

	lat, lng := at(originLat, originLng)
	seedAt(t, repo, creator, "Right Here Cafe", lat, lng, true)
	lat, lng = at(9.0180, 38.7500) // ~2 km north
	seedAt(t, repo, creator, "Two Km North", lat, lng, true)
	lat, lng = at(9.1800, 38.7500) // ~20 km north
	seedAt(t, repo, creator, "Twenty Km North", lat, lng, true)
	lat, lng = at(9.0010, 38.7500) // close, but not published
	seedAt(t, repo, creator, "Unpublished Neighbour", lat, lng, false)
	seedAt(t, repo, creator, "Online Only Shop", nil, nil, true)

	t.Run("returns targets inside the radius, closest first", func(t *testing.T) {
		got, err := repo.Nearby(context.Background(), BrowseFilter{}, originLat, originLng, 5, 20)

		require.NoError(t, err)
		names := make([]string, 0, len(got))
		for _, g := range got {
			names = append(names, g.Name)
		}
		assert.Equal(t, []string{"Right Here Cafe", "Two Km North"}, names)
		assert.InDelta(t, 0.0, got[0].DistanceKm, 0.01)
		assert.InDelta(t, 2.0, got[1].DistanceKm, 0.15)
	})

	t.Run("a wider radius reaches further targets", func(t *testing.T) {
		got, err := repo.Nearby(context.Background(), BrowseFilter{}, originLat, originLng, 25, 20)

		require.NoError(t, err)
		require.Len(t, got, 3)
		assert.Equal(t, "Twenty Km North", got[2].Name)
		assert.InDelta(t, 20.0, got[2].DistanceKm, 0.5)
	})

	t.Run("honours the limit", func(t *testing.T) {
		got, err := repo.Nearby(context.Background(), BrowseFilter{}, originLat, originLng, 25, 1)

		require.NoError(t, err)
		assert.Len(t, got, 1)
	})

	t.Run("excludes non-matching categories", func(t *testing.T) {
		other := uuid.MustParse("11111111-1111-4111-8111-111111111101")
		got, err := repo.Nearby(context.Background(), BrowseFilter{CategoryID: &other}, originLat, originLng, 25, 20)

		require.NoError(t, err)
		assert.Empty(t, got)
	})
}
