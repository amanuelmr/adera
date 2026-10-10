package app

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/config"
)

var jsonLDPattern = regexp.MustCompile(`(?s)<script type="application/ld\+json">(.*?)</script>`)

// jsonLD extracts and parses a page's structured data, if any.
func jsonLD(t *testing.T, body string) map[string]any {
	t.Helper()
	m := jsonLDPattern.FindStringSubmatch(body)
	if m == nil {
		return nil
	}
	var out map[string]any
	require.NoError(t, json.Unmarshal([]byte(m[1]), &out), "JSON-LD must be valid JSON: %s", m[1])
	return out
}

func TestDiscoveryPages(t *testing.T) {
	a := newTestAPIWith(t, func(c *config.Config) { c.BaseURL = "https://adera.example" })
	mod := a.register("discmod")
	a.grantRoles(&mod, "moderator")

	rated := a.createTarget(mod, "Discovery Rated Café", catRestaurantID, "restaurant")
	ratedSlug := a.slugOf(rated)
	for i, stars := range []int{5, 4, 4} {
		a.review(a.register("disc"+string(rune('a'+i))), rated, stars, nil)
	}
	quiet := a.createTarget(mod, "Discovery Quiet Café", catRestaurantID, "restaurant")
	quietSlug := a.slugOf(quiet)
	a.review(a.register("discquiet"), quiet, 5, nil)
	electronics := a.createTarget(mod, "Discovery Phone Shop", catElectronicsID, "online_seller")

	t.Run("home lists top rated, trending and categories", func(t *testing.T) {
		res, body := a.page("/")
		require.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, body, `href="/t/`+ratedSlug+`"`)
		assert.Contains(t, body, "4.3 ★ · 3 reviews")
		assert.Contains(t, body, "3 new reviews this month")
		assert.Contains(t, body, `href="/c/restaurant_cafe"`)
		assert.Contains(t, body, "1 review", "the quiet café is listed…")
		assert.NotContains(t, body, "5.0 ★ · 1 review", "…but without an average from a single review")
		assert.Less(t, len(body), 40<<10)

		_, am := a.page("/?lang=am")
		assert.Contains(t, am, `<html lang="am">`)
		assert.Contains(t, am, `href="/c/restaurant_cafe?lang=am"`)
	})

	t.Run("category page lists only its places", func(t *testing.T) {
		res, body := a.page("/c/restaurant_cafe")
		require.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, body, "<h1>Restaurants &amp; Cafés</h1>")
		assert.Contains(t, body, `href="/t/`+ratedSlug+`"`)
		assert.Contains(t, body, `href="/t/`+quietSlug+`"`)
		assert.NotContains(t, body, "Discovery Phone Shop")

		for _, path := range []string{"/c/no_such_category", "/c/" + catRestaurantID} {
			res, _ := a.page(path)
			assert.Equal(t, http.StatusNotFound, res.StatusCode, path)
		}
	})

	t.Run("place pages carry JSON-LD, with a rating only when one is shown", func(t *testing.T) {
		_, body := a.page("/t/" + ratedSlug)
		ld := jsonLD(t, body)
		require.NotNil(t, ld)
		assert.Equal(t, "LocalBusiness", ld["@type"])
		assert.Equal(t, "Discovery Rated Café", ld["name"])
		assert.Equal(t, "https://adera.example/t/"+ratedSlug, ld["url"])
		rating := ld["aggregateRating"].(map[string]any)
		assert.Equal(t, "4.3", rating["ratingValue"])
		assert.Equal(t, float64(3), rating["reviewCount"])

		_, quietBody := a.page("/t/" + quietSlug)
		quietLD := jsonLD(t, quietBody)
		require.NotNil(t, quietLD)
		assert.NotContains(t, quietLD, "aggregateRating", "one review: no rating on the page, none in the markup")
	})

	t.Run("sitemap lists the home, categories and published places", func(t *testing.T) {
		res, body := a.page("/sitemap.xml")
		require.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "application/xml; charset=utf-8", res.Header.Get("Content-Type"))
		var set struct {
			URLs []struct {
				Loc     string `xml:"loc"`
				LastMod string `xml:"lastmod"`
			} `xml:"url"`
		}
		require.NoError(t, xml.Unmarshal([]byte(body), &set))
		locs := map[string]string{}
		for _, u := range set.URLs {
			locs[u.Loc] = u.LastMod
		}
		assert.Contains(t, locs, "https://adera.example/")
		assert.Contains(t, locs, "https://adera.example/c/restaurant_cafe")
		assert.Contains(t, locs, "https://adera.example/t/"+ratedSlug)
		assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, locs["https://adera.example/t/"+ratedSlug])
		assert.Contains(t, locs, "https://adera.example/t/"+a.slugOf(electronics))

		user := a.register("discpending")
		status, created := a.do("POST", "/api/v1/targets", map[string]any{
			"target_type": "restaurant", "category_id": catRestaurantID, "name": "Discovery Pending Place",
		}, user.Access)
		require.Equal(t, http.StatusCreated, status, "%v", created)
		_, body = a.page("/sitemap.xml")
		assert.False(t, strings.Contains(body, data(created)["slug"].(string)), "unpublished places stay out")
	})
}
