package app

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Anyone holding a reviewer's account ID could collect everything they
// wrote, so it is sent only to the author and to moderators.
func TestReviewAuthorIDIsNotPublic(t *testing.T) {
	a := newTestAPI(t)
	mod := a.register("authormod")
	a.grantRoles(&mod, "moderator")
	author := a.register("authorself")
	stranger := a.register("authorstranger")
	target := a.createTarget(mod, "Author Privacy Cafe", catRestaurantID, "restaurant")
	review := a.review(author, target, 4, nil)

	for _, viewer := range []struct {
		name   string
		token  string
		mine   bool
		reveal bool
	}{
		{"anonymous", "", false, false},
		{"another user", stranger.Access, false, false},
		{"the author", author.Access, true, true},
		{"a moderator", mod.Access, false, true},
	} {
		t.Run(viewer.name, func(t *testing.T) {
			status, res := a.do("GET", "/api/v1/reviews/"+review, nil, viewer.token)
			require.Equal(t, http.StatusOK, status)
			one := data(res)
			status, res = a.do("GET", "/api/v1/targets/"+target+"/reviews", nil, viewer.token)
			require.Equal(t, http.StatusOK, status)
			require.Len(t, dataList(res), 1)
			listed := dataList(res)[0].(map[string]any)

			for _, rv := range []map[string]any{one, listed} {
				assert.Equal(t, viewer.mine, rv["is_mine"])
				if viewer.reveal {
					assert.Equal(t, author.ID, rv["user_id"])
				} else {
					assert.NotContains(t, rv, "user_id")
				}
			}
		})
	}

	status, res := a.do("GET", "/api/v1/users/me/reviews", nil, author.Access)
	require.Equal(t, http.StatusOK, status)
	require.Len(t, dataList(res), 1)
	assert.Equal(t, true, dataList(res)[0].(map[string]any)["is_mine"])
}
