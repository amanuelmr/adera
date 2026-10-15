package app

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A place's cover is the newest photo on one of its published reviews, and
// it appears wherever the place is listed.
func TestPlaceCoverComesFromReviewPhotos(t *testing.T) {
	a := newTestAPI(t)
	mod := a.register("covermod")
	a.grantRoles(&mod, "moderator")
	target := a.createTarget(mod, "Cover Photo Cafe", catRestaurantID, "cafe")
	cover := func() (string, string) {
		status, res := a.do("GET", "/api/v1/targets/"+target, nil, "")
		require.Equal(t, http.StatusOK, status)
		full, _ := data(res)["cover_url"].(string)
		thumb, _ := data(res)["cover_thumb_url"].(string)
		return full, thumb
	}

	full, _ := cover()
	assert.Empty(t, full, "no photos yet")

	first := a.register("cover-first")
	firstReview := a.review(first, target, 5, nil)
	firstKey, firstThumb := a.uploadPhoto(first, firstReview)
	full, thumb := cover()
	assert.True(t, strings.HasSuffix(full, "/"+firstKey), full)
	assert.True(t, strings.HasSuffix(thumb, "/"+firstThumb), thumb)
	assert.True(t, strings.HasPrefix(full, a.cfg.StoragePublicBaseURL), "a public link")

	second := a.register("cover-second")
	secondReview := a.review(second, target, 4, nil)
	secondKey, _ := a.uploadPhoto(second, secondReview)
	full, _ = cover()
	assert.True(t, strings.HasSuffix(full, "/"+secondKey), "the newest photo")

	for _, path := range []string{"/api/v1/targets?type=cafe", "/api/v1/targets/top-rated?type=cafe", "/api/v1/targets/trending?type=cafe",
		"/api/v1/search/targets?q=Cover+Photo"} {
		status, res := a.do("GET", path, nil, "")
		require.Equal(t, http.StatusOK, status, path)
		require.NotEmpty(t, dataList(res), path)
		listed := dataList(res)[0].(map[string]any)
		assert.Equal(t, full, listed["cover_url"], path)
	}

	// The newest photo's review leaves public view: its photo is never a cover.
	status, _ := a.do("POST", "/api/v1/moderation/reviews/"+secondReview+"/decision",
		map[string]any{"action": "hide", "note": "photo shows a face"}, mod.Access)
	require.Equal(t, http.StatusOK, status)
	full, _ = cover()
	assert.True(t, strings.HasSuffix(full, "/"+firstKey), "falls back to the next published photo")

	status, _ = a.do("DELETE", "/api/v1/reviews/"+firstReview, nil, first.Access)
	require.Equal(t, http.StatusOK, status)
	full, _ = cover()
	assert.Empty(t, full, "no published photos left")
}
