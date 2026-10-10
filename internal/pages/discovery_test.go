package pages

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/ratings"
	"github.com/adera-platform/backend/internal/targets"
)

func TestStructuredDataOnlyClaimsAShownRating(t *testing.T) {
	h, _ := newTestHandler(t, Config{BaseURL: "https://adera.example"})
	avg := 4.27
	place := targets.Target{Name: "Tomoca", Slug: "tomoca", AddressText: "Piassa, Addis Ababa", Phone: "+251111234567"}
	stats := ratings.TargetStats{ReviewCount: 12, Average: &avg}

	shown := h.targetStructuredData(place, stats, true)
	raw, err := json.Marshal(shown)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	assert.Equal(t, "https://schema.org", got["@context"])
	assert.Equal(t, "LocalBusiness", got["@type"])
	assert.Equal(t, "https://adera.example/t/tomoca", got["url"])
	assert.Equal(t, map[string]any{"@type": "PostalAddress", "streetAddress": "Piassa, Addis Ababa", "addressCountry": "ET"}, got["address"])
	assert.Equal(t, map[string]any{
		"@type": "AggregateRating", "ratingValue": "4.3", "reviewCount": float64(12), "bestRating": float64(5), "worstRating": float64(1),
	}, got["aggregateRating"])

	hidden := h.targetStructuredData(place, ratings.TargetStats{ReviewCount: 2, Average: &avg}, false)
	assert.Nil(t, hidden.AggregateRating, "below the minimum sample the page shows no rating, so neither does the markup")

	online := h.targetStructuredData(targets.Target{Name: "Shop", Slug: "shop", OnlineOnly: true}, stats, true)
	assert.Equal(t, "Organization", online.Type)
	assert.Nil(t, online.Address)
}

func TestPlaceItemWithholdsSmallSampleAverages(t *testing.T) {
	h, _ := newTestHandler(t, Config{})
	b := base{Lang: "en", dicts: h.dicts}
	avg := 4.5

	few := newPlaceItem(b, targets.Target{Name: "New Café", Slug: "new-cafe", ReviewCount: 2, AverageRating: &avg})
	assert.Equal(t, "2 reviews", few.Meta)
	assert.Equal(t, "/t/new-cafe", few.URL)

	many := newPlaceItem(b, targets.Target{Name: "Tomoca", Slug: "tomoca", ReviewCount: 3, AverageRating: &avg}, "1 new review this month")
	assert.Equal(t, "4.5 ★ · 3 reviews · 1 new review this month", many.Meta)
}
