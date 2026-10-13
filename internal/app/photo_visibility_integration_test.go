package app

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/media"
)

// uploadPhoto attaches a finalized photo to the review and returns its
// public object key and thumbnail key.
func (a *testAPI) uploadPhoto(author testUser, reviewID string) (key, thumbKey string) {
	a.t.Helper()
	ctx := context.Background()
	status, res := a.do("POST", "/api/v1/reviews/"+reviewID+"/media",
		map[string]any{"content_type": "image/jpeg"}, author.Access)
	require.Equal(a.t, http.StatusCreated, status, "%v", res)
	uploadID := data(res)["upload_id"].(string)
	staging := data(res)["upload"].(map[string]any)["key"].(string)

	// Large enough to get a thumbnail.
	var buf bytes.Buffer
	require.NoError(a.t, jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1200, 900)), nil))
	require.NoError(a.t, a.store.Put(ctx, a.cfg.StoragePrivateBucket, staging,
		bytes.NewReader(buf.Bytes()), int64(buf.Len()), "image/jpeg"))
	status, res = a.do("POST", "/api/v1/media/uploads/"+uploadID+"/finalize", nil, author.Access)
	require.Equal(a.t, http.StatusOK, status, "%v", res)

	var thumb *string
	require.NoError(a.t, a.pool.QueryRow(ctx,
		`SELECT object_key, thumb_key FROM review_media WHERE id = $1`, uploadID).Scan(&key, &thumb))
	require.NotNil(a.t, thumb, "test photo should get a thumbnail")
	return key, *thumb
}

func (a *testAPI) objectIn(bucket, key string) bool {
	_, err := a.store.Stat(context.Background(), bucket, key)
	return err == nil
}

// A review that leaves public view takes its photos out of the public
// bucket; a restored review brings them back.
func TestPhotosFollowTheirReviewsVisibility(t *testing.T) {
	a := newTestAPI(t)
	ctx := context.Background()
	mod := a.register("photomod")
	a.grantRoles(&mod, "moderator")
	author := a.register("photoauthor")
	target := a.createTarget(mod, "Photo Visibility Cafe", catRestaurantID, "restaurant")
	review := a.review(author, target, 4, nil)
	key, thumb := a.uploadPhoto(author, review)
	svc := media.NewService(a.pool, a.store, nil, a.cfg.StoragePrivateBucket, a.cfg.StoragePublicBucket)
	pub, priv := a.cfg.StoragePublicBucket, a.cfg.StoragePrivateBucket

	n, err := svc.SyncVisibility(ctx, 100)
	require.NoError(t, err)
	assert.Zero(t, n, "a published review's photos stay put")

	status, _ := a.do("POST", "/api/v1/moderation/reviews/"+review+"/decision",
		map[string]any{"action": "remove", "note": "photo shows a face"}, mod.Access)
	require.Equal(t, http.StatusOK, status)
	n, err = svc.SyncVisibility(ctx, 100)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.False(t, a.objectIn(pub, key), "photo out of the public bucket")
	assert.False(t, a.objectIn(pub, thumb), "thumbnail too")
	assert.True(t, a.objectIn(priv, "withheld/"+key))
	assert.True(t, a.objectIn(priv, "withheld/"+thumb))
	n, err = svc.SyncVisibility(ctx, 100)
	require.NoError(t, err)
	assert.Zero(t, n, "nothing left to move")

	status, _ = a.do("POST", "/api/v1/moderation/reviews/"+review+"/decision",
		map[string]any{"action": "restore"}, mod.Access)
	require.Equal(t, http.StatusOK, status)
	// Between the restore and the next sync, no broken URL is listed.
	status, list := a.do("GET", "/api/v1/targets/"+target+"/reviews", nil, "")
	require.Equal(t, http.StatusOK, status)
	require.Len(t, dataList(list), 1)
	assert.Nil(t, dataList(list)[0].(map[string]any)["media"], "withheld photos aren't listed")

	n, err = svc.SyncVisibility(ctx, 100)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.True(t, a.objectIn(pub, key), "back in public view")
	assert.True(t, a.objectIn(pub, thumb))
	assert.False(t, a.objectIn(priv, "withheld/"+key))
	_, list = a.do("GET", "/api/v1/targets/"+target+"/reviews", nil, "")
	assert.Len(t, dataList(list)[0].(map[string]any)["media"], 1)

	// The author removing their own review withholds too.
	status, _ = a.do("DELETE", "/api/v1/reviews/"+review, nil, author.Access)
	require.Equal(t, http.StatusOK, status)
	n, err = svc.SyncVisibility(ctx, 100)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.False(t, a.objectIn(pub, key))
}

// An interrupted move (copied, original not yet removed or row not yet
// updated) is simply redone.
func TestPhotoMoveResumes(t *testing.T) {
	a := newTestAPI(t)
	ctx := context.Background()
	mod := a.register("resumemod")
	a.grantRoles(&mod, "moderator")
	author := a.register("resumeauthor")
	target := a.createTarget(mod, "Photo Resume Cafe", catRestaurantID, "restaurant")
	review := a.review(author, target, 4, nil)
	key, thumb := a.uploadPhoto(author, review)
	svc := media.NewService(a.pool, a.store, nil, a.cfg.StoragePrivateBucket, a.cfg.StoragePublicBucket)
	pub, priv := a.cfg.StoragePublicBucket, a.cfg.StoragePrivateBucket

	status, _ := a.do("POST", "/api/v1/moderation/reviews/"+review+"/decision",
		map[string]any{"action": "hide"}, mod.Access)
	require.Equal(t, http.StatusOK, status)
	// Simulate a crash after the full photo moved but before the thumbnail
	// and the row: the photo is already private, the thumbnail still public.
	obj, err := a.store.Get(ctx, pub, key)
	require.NoError(t, err)
	var buf bytes.Buffer
	_, err = buf.ReadFrom(obj)
	require.NoError(t, err)
	require.NoError(t, a.store.Put(ctx, priv, "withheld/"+key, &buf, int64(buf.Len()), "image/jpeg"))
	require.NoError(t, a.store.Remove(ctx, pub, key))

	n, err := svc.SyncVisibility(ctx, 100)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
	assert.True(t, a.objectIn(priv, "withheld/"+key))
	assert.True(t, a.objectIn(priv, "withheld/"+thumb))
	assert.False(t, a.objectIn(pub, thumb))
}
