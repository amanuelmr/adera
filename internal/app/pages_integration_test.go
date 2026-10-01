package app

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pageResult struct {
	StatusCode int
	Header     http.Header
}

// page fetches an HTML page, not following redirects.
func (a *testAPI) page(path string, headers ...string) (pageResult, string) {
	a.t.Helper()
	req, err := http.NewRequest(http.MethodGet, a.srv.URL+path, nil)
	require.NoError(a.t, err)
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	res, err := client.Do(req)
	require.NoError(a.t, err)
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	require.NoError(a.t, err)
	return pageResult{StatusCode: res.StatusCode, Header: res.Header}, string(body)
}

func (a *testAPI) slugOf(targetID string) string {
	a.t.Helper()
	status, res := a.do("GET", "/api/v1/targets/"+targetID, nil, "")
	require.Equal(a.t, http.StatusOK, status, "%v", res)
	return data(res)["slug"].(string)
}

func TestWebPages(t *testing.T) {
	a := newTestAPI(t)
	mod := a.register("pagesmod")
	a.grantRoles(&mod, "moderator", "admin")
	target := a.createTarget(mod, "Page Test Café", catRestaurantID, "restaurant")
	slug := a.slugOf(target)

	five := a.review(a.register("pages1"), target, 5, map[string]any{"title": "Best macchiato in Piassa"})
	a.review(a.register("pages2"), target, 4, map[string]any{"incentive_type": "discount"})
	a.review(a.register("pages3"), target, 2, nil)

	t.Run("target page shows the aggregate, reviews and disclosures", func(t *testing.T) {
		res, body := a.page("/t/" + slug)
		require.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, "text/html; charset=utf-8", res.Header.Get("Content-Type"))
		assert.Contains(t, res.Header.Get("Content-Security-Policy"), "style-src 'self'",
			"the API's default-src 'none' would block the stylesheet")
		assert.Equal(t, "public, max-age=300", res.Header.Get("Cache-Control"))

		assert.Contains(t, body, "Page Test Café")
		assert.Contains(t, body, `<meta property="og:title" content="Page Test Café — 3.7★ (3)">`)
		assert.Contains(t, body, "Rated 3.7 out of 5 from 3 reviews")
		assert.Contains(t, body, "Best macchiato in Piassa")
		assert.Contains(t, body, "Received a discount for this review", "moderation policy §6 labels")
		assert.Contains(t, body, `href="/r/`+five+`"`, "each review links to its permalink")
		assert.Contains(t, body, "adera://target/"+slug)

		// Size budget (docs/frontend-handoff.md §5): tiny next to 300 KB.
		assert.Less(t, len(body), 40<<10)
	})

	t.Run("revisits cost a 304", func(t *testing.T) {
		res, _ := a.page("/t/" + slug)
		etag := res.Header.Get("ETag")
		require.NotEmpty(t, etag)
		again, _ := a.page("/t/"+slug, "If-None-Match", etag)
		assert.Equal(t, http.StatusNotModified, again.StatusCode)
	})

	t.Run("histogram links filter by rating", func(t *testing.T) {
		_, body := a.page("/t/" + slug + "?rating=5")
		assert.Contains(t, body, "Best macchiato in Piassa")
		assert.NotContains(t, body, "Received a discount", "the 4-star review is filtered out")
	})

	t.Run("Amharic by Accept-Language, and an explicit choice sticks in links", func(t *testing.T) {
		_, body := a.page("/t/"+slug, "Accept-Language", "am-ET,am;q=0.9")
		assert.Contains(t, body, `<html lang="am">`)
		assert.Contains(t, body, "ለዚህ ግምገማ ቅናሽ ተቀብሏል")

		_, body = a.page("/t/" + slug + "?lang=am")
		assert.Contains(t, body, `href="/r/`+five+`?lang=am"`)
	})

	t.Run("review permalink", func(t *testing.T) {
		res, body := a.page("/r/" + five)
		require.Equal(t, http.StatusOK, res.StatusCode)
		assert.Contains(t, body, "Best macchiato in Piassa")
		assert.Contains(t, body, `<meta property="og:type" content="article">`)
		assert.Contains(t, body, `href="/t/`+slug+`"`)
	})

	t.Run("unknown and malformed ids are a 404 page, not an error", func(t *testing.T) {
		for _, path := range []string{"/t/no-such-place", "/r/not-a-uuid", "/r/00000000-0000-4000-8000-000000000000"} {
			res, body := a.page(path)
			assert.Equal(t, http.StatusNotFound, res.StatusCode, path)
			assert.Contains(t, body, "Not found", path)
			assert.Equal(t, "no-store", res.Header.Get("Cache-Control"), path)
		}
	})

	t.Run("a removed review's permalink is gone", func(t *testing.T) {
		author := a.register("pagesgone")
		gone := a.review(author, target, 3, nil)
		status, _ := a.do("DELETE", "/api/v1/reviews/"+gone, nil, author.Access)
		require.Equal(t, http.StatusOK, status)
		res, _ := a.page("/r/" + gone)
		assert.Equal(t, http.StatusNotFound, res.StatusCode)
	})

	t.Run("an unpublished target is not served", func(t *testing.T) {
		user := a.register("pagespending")
		status, res := a.do("POST", "/api/v1/targets", map[string]any{
			"target_type": "restaurant", "category_id": catRestaurantID, "name": "Pending Page Place",
		}, user.Access)
		require.Equal(t, http.StatusCreated, status, "%v", res)
		pendingSlug := data(res)["slug"].(string)
		page, _ := a.page("/t/" + pendingSlug)
		assert.Equal(t, http.StatusNotFound, page.StatusCode)
	})

	t.Run("a merged target redirects to the one it was merged into", func(t *testing.T) {
		dup := a.createTarget(mod, "Page Test Cafe Duplicate", catRestaurantID, "restaurant")
		dupSlug := a.slugOf(dup)
		status, res := a.do("POST", "/api/v1/admin/targets/"+dup+"/merge", map[string]any{"into_id": target}, mod.Access)
		require.Equal(t, http.StatusOK, status, "%v", res)

		page, _ := a.page("/t/" + dupSlug)
		assert.Equal(t, http.StatusMovedPermanently, page.StatusCode)
		assert.True(t, strings.HasSuffix(page.Header.Get("Location"), "/t/"+slug))
	})
}
