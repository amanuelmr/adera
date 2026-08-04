package app

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/storage"
	"github.com/adera-platform/backend/internal/platform/testdb"
)

// TestAuthorizationMatrix verifies function-level authorization: every
// protected endpoint rejects anonymous callers, and role-gated endpoints
// reject plain customers.
func TestAuthorizationMatrix(t *testing.T) {
	a := newTestAPI(t)
	customer := a.register("matrix")
	someID := "11111111-1111-4111-8111-111111111101"

	authRequired := []struct{ method, path string }{
		{"GET", "/api/v1/users/me"},
		{"PATCH", "/api/v1/users/me"},
		{"POST", "/api/v1/users/me/password"},
		{"DELETE", "/api/v1/users/me"},
		{"GET", "/api/v1/users/me/reviews"},
		{"GET", "/api/v1/users/me/reports"},
		{"GET", "/api/v1/users/me/notifications"},
		{"GET", "/api/v1/users/me/notifications/unread-count"},
		{"PUT", "/api/v1/users/me/notifications/read-all"},
		{"PUT", "/api/v1/users/me/notifications/" + someID + "/read"},
		{"POST", "/api/v1/auth/logout"},
		{"POST", "/api/v1/auth/logout-all"},
		{"GET", "/api/v1/auth/sessions"},
		{"POST", "/api/v1/targets"},
		{"POST", "/api/v1/reviews"},
		{"PUT", "/api/v1/reviews/" + someID},
		{"DELETE", "/api/v1/reviews/" + someID},
		{"PUT", "/api/v1/reviews/" + someID + "/helpful"},
		{"POST", "/api/v1/reviews/" + someID + "/reports"},
		{"POST", "/api/v1/reviews/" + someID + "/media"},
		{"POST", "/api/v1/reviews/" + someID + "/evidence"},
		{"POST", "/api/v1/businesses"},
		{"POST", "/api/v1/businesses/" + someID + "/claims"},
		{"GET", "/api/v1/claims/mine"},
	}
	for _, ep := range authRequired {
		status, _ := a.do(ep.method, ep.path, map[string]any{}, "")
		assert.Equal(t, http.StatusUnauthorized, status, "%s %s must require auth", ep.method, ep.path)
	}

	roleGated := []struct{ method, path string }{
		{"GET", "/api/v1/moderation/reports"},
		{"GET", "/api/v1/moderation/reports/" + someID},
		{"POST", "/api/v1/moderation/reports/" + someID + "/resolve"},
		{"POST", "/api/v1/moderation/reviews/" + someID + "/decision"},
		{"POST", "/api/v1/moderation/targets/" + someID + "/decision"},
		{"GET", "/api/v1/moderation/reviews/" + someID + "/evidence"},
		{"POST", "/api/v1/moderation/evidence/" + someID + "/decision"},
		{"POST", "/api/v1/moderation/claims/" + someID + "/decision"},
		{"POST", "/api/v1/moderation/notes"},
		{"GET", "/api/v1/moderation/audit?subject_type=review&subject_id=" + someID},
		{"POST", "/api/v1/admin/users/" + someID + "/suspend"},
		{"POST", "/api/v1/admin/categories"},
		{"PATCH", "/api/v1/admin/categories/" + someID},
		{"POST", "/api/v1/admin/targets/" + someID + "/merge"},
	}
	for _, ep := range roleGated {
		status, _ := a.do(ep.method, ep.path, map[string]any{}, customer.Access)
		assert.Equal(t, http.StatusForbidden, status, "%s %s must reject customers", ep.method, ep.path)
	}

	// Moderator can reach moderation but not admin endpoints.
	moderator := a.register("modmatrix")
	a.grantRoles(&moderator, "moderator")
	status, _ := a.do("GET", "/api/v1/moderation/reports", nil, moderator.Access)
	assert.Equal(t, http.StatusOK, status)
	status, _ = a.do("POST", "/api/v1/admin/categories", map[string]any{}, moderator.Access)
	assert.Equal(t, http.StatusForbidden, status, "moderator must not reach admin endpoints")
}

// exifJPEG builds a real JPEG with an injected APP1 (EXIF) segment.
func exifJPEG(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 32, 32)), nil))
	raw := buf.Bytes()
	require.Equal(t, []byte{0xFF, 0xD8}, raw[:2])

	payload := append([]byte("Exif\x00\x00"), bytes.Repeat([]byte{0xAB}, 600)...)
	segment := make([]byte, 4+len(payload))
	segment[0], segment[1] = 0xFF, 0xE1
	binary.BigEndian.PutUint16(segment[2:], uint16(len(payload)+2))
	copy(segment[4:], payload)

	out := append([]byte{0xFF, 0xD8}, segment...)
	return append(out, raw[2:]...)
}

// hasAPP1 walks JPEG segments looking for an APP1 marker.
func hasAPP1(data []byte) bool {
	i := 2 // skip SOI
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			return false
		}
		marker := data[i+1]
		if marker == 0xE1 {
			return true
		}
		if marker == 0xDA { // start of scan: entropy data follows
			return false
		}
		size := int(binary.BigEndian.Uint16(data[i+2:]))
		i += 2 + size
	}
	return false
}

func TestMediaPipeline(t *testing.T) {
	a := newTestAPI(t)
	mod := a.register("mod")
	a.grantRoles(&mod, "moderator")
	target := a.createTarget(mod, "Media Restaurant", catRestaurantID, "restaurant")
	author := a.register("uploader")
	review := a.review(author, target, 4, nil)
	ctx := context.Background()

	t.Run("public media is re-encoded and EXIF-stripped", func(t *testing.T) {
		status, res := a.do("POST", "/api/v1/reviews/"+review+"/media",
			map[string]any{"content_type": "image/jpeg"}, author.Access)
		require.Equal(t, http.StatusCreated, status, "%v", res)
		uploadID := data(res)["upload_id"].(string)
		key := data(res)["upload"].(map[string]any)["key"].(string)

		dirty := exifJPEG(t)
		require.True(t, hasAPP1(dirty), "test fixture must contain EXIF")
		require.NoError(t, a.store.Put(ctx, a.cfg.StoragePrivateBucket, key,
			bytes.NewReader(dirty), int64(len(dirty)), "image/jpeg"))

		status, res = a.do("POST", "/api/v1/media/uploads/"+uploadID+"/finalize", nil, author.Access)
		require.Equal(t, http.StatusOK, status, "%v", res)
		finalKey := data(res)["object_key"].(string)
		status, res = a.do("POST", "/api/v1/media/uploads/"+uploadID+"/finalize", nil, author.Access)
		require.Equal(t, http.StatusOK, status, "%v", res)
		assert.Equal(t, finalKey, data(res)["object_key"], "finalization must be idempotent")

		obj, err := a.store.Get(ctx, a.cfg.StoragePublicBucket, finalKey)
		require.NoError(t, err)
		clean, err := io.ReadAll(obj)
		require.NoError(t, err)
		require.NoError(t, obj.Close())
		assert.False(t, hasAPP1(clean), "EXIF/APP1 metadata must be stripped from public media")

		// Staging object removed; verification level raised to media_attached.
		_, err = a.store.Stat(ctx, a.cfg.StoragePrivateBucket, key)
		assert.Error(t, err, "staging object must be deleted")
		status, res = a.do("GET", "/api/v1/reviews/"+review, nil, "")
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, "media_attached", data(res)["verification_level"])
	})

	t.Run("content mismatch and bad sizes rejected", func(t *testing.T) {
		status, res := a.do("POST", "/api/v1/reviews/"+review+"/media",
			map[string]any{"content_type": "image/png"}, author.Access)
		require.Equal(t, http.StatusCreated, status)
		uploadID := data(res)["upload_id"].(string)
		key := data(res)["upload"].(map[string]any)["key"].(string)

		// A JPEG uploaded where PNG was declared must be rejected.
		payload := exifJPEG(t)
		require.NoError(t, a.store.Put(ctx, a.cfg.StoragePrivateBucket, key,
			bytes.NewReader(payload), int64(len(payload)), "image/png"))
		status, res = a.do("POST", "/api/v1/media/uploads/"+uploadID+"/finalize", nil, author.Access)
		assert.Equal(t, http.StatusUnprocessableEntity, status, "%v", res)

		// Tiny non-image payload also rejected.
		status, res = a.do("POST", "/api/v1/reviews/"+review+"/media",
			map[string]any{"content_type": "image/jpeg"}, author.Access)
		require.Equal(t, http.StatusCreated, status)
		uploadID = data(res)["upload_id"].(string)
		key = data(res)["upload"].(map[string]any)["key"].(string)
		require.NoError(t, a.store.Put(ctx, a.cfg.StoragePrivateBucket, key,
			bytes.NewReader([]byte("tiny")), 4, "image/jpeg"))
		status, _ = a.do("POST", "/api/v1/media/uploads/"+uploadID+"/finalize", nil, author.Access)
		assert.Equal(t, http.StatusUnprocessableEntity, status)
	})

	t.Run("unsupported public content type rejected at presign", func(t *testing.T) {
		status, _ := a.do("POST", "/api/v1/reviews/"+review+"/media",
			map[string]any{"content_type": "image/gif"}, author.Access)
		assert.Equal(t, http.StatusUnprocessableEntity, status)
		status, _ = a.do("POST", "/api/v1/reviews/"+review+"/media",
			map[string]any{"content_type": "application/x-sh"}, author.Access)
		assert.Equal(t, http.StatusUnprocessableEntity, status)
	})

	t.Run("only the author can attach media", func(t *testing.T) {
		stranger := a.register("stranger")
		status, _ := a.do("POST", "/api/v1/reviews/"+review+"/media",
			map[string]any{"content_type": "image/jpeg"}, stranger.Access)
		assert.Equal(t, http.StatusForbidden, status)
	})

	t.Run("private evidence flow and verification upgrade", func(t *testing.T) {
		status, res := a.do("POST", "/api/v1/reviews/"+review+"/evidence",
			map[string]any{"kind": "receipt", "content_type": "image/jpeg"}, author.Access)
		require.Equal(t, http.StatusCreated, status, "%v", res)
		evidenceID := data(res)["upload_id"].(string)
		key := data(res)["upload"].(map[string]any)["key"].(string)

		payload := exifJPEG(t)
		require.NoError(t, a.store.Put(ctx, a.cfg.StoragePrivateBucket, key,
			bytes.NewReader(payload), int64(len(payload)), "image/jpeg"))
		status, _ = a.do("POST", "/api/v1/evidence/"+evidenceID+"/finalize", nil, author.Access)
		require.Equal(t, http.StatusOK, status)

		// Evidence is not in any public listing; owner and moderator can list it.
		status, res = a.do("GET", "/api/v1/reviews/"+review+"/evidence", nil, author.Access)
		require.Equal(t, http.StatusOK, status)
		require.Len(t, dataList(res), 1)
		stranger := a.register("evidence-stranger")
		status, _ = a.do("GET", "/api/v1/reviews/"+review+"/evidence", nil, stranger.Access)
		assert.Equal(t, http.StatusForbidden, status, "evidence is private")

		status, res = a.do("GET", "/api/v1/moderation/reviews/"+review+"/evidence", nil, mod.Access)
		require.Equal(t, http.StatusOK, status)
		items := dataList(res)
		require.Len(t, items, 1)
		assert.NotEmpty(t, items[0].(map[string]any)["access_url"], "moderator gets a presigned URL")

		// Review is NOT verified until the moderator accepts.
		status, res = a.do("GET", "/api/v1/reviews/"+review, nil, "")
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, "media_attached", data(res)["verification_level"], "submission alone must not verify")

		status, _ = a.do("POST", "/api/v1/moderation/evidence/"+evidenceID+"/decision",
			map[string]any{"decision": "accepted", "note": "receipt checks out"}, mod.Access)
		require.Equal(t, http.StatusOK, status)
		status, res = a.do("GET", "/api/v1/reviews/"+review, nil, "")
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, "receipt_submitted", data(res)["verification_level"])

		// Verified aggregates follow.
		s := a.stats(target)
		assert.EqualValues(t, 1, s["verified_count"])
		assert.InDelta(t, 4.0, s["verified_average"].(float64), 0.001)
	})
}

func TestStaleUploadTicketsDoNotConsumeSlots(t *testing.T) {
	a := newTestAPI(t)
	mod := a.register("stale-mod")
	a.grantRoles(&mod, "moderator")
	target := a.createTarget(mod, "Stale Upload Restaurant", catRestaurantID, "restaurant")
	author := a.register("stale-uploader")
	review := a.review(author, target, 4, nil)

	for range 5 {
		status, res := a.do("POST", "/api/v1/reviews/"+review+"/media",
			map[string]any{"content_type": "image/jpeg"}, author.Access)
		require.Equal(t, http.StatusCreated, status, "%v", res)
	}
	status, _ := a.do("POST", "/api/v1/reviews/"+review+"/media",
		map[string]any{"content_type": "image/jpeg"}, author.Access)
	require.Equal(t, http.StatusConflict, status)

	_, err := a.pool.Exec(context.Background(), `
		UPDATE review_media SET created_at = now() - interval '1 hour'
		WHERE review_id = $1 AND status = 'staged'`, review)
	require.NoError(t, err)
	status, res := a.do("POST", "/api/v1/reviews/"+review+"/media",
		map[string]any{"content_type": "image/jpeg"}, author.Access)
	require.Equal(t, http.StatusCreated, status, "%v", res)

	var mediaRows int
	err = a.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM review_media WHERE review_id = $1`, review).Scan(&mediaRows)
	require.NoError(t, err)
	assert.Equal(t, 1, mediaRows, "expired staged media rows should be removed")

	for range 5 {
		status, res = a.do("POST", "/api/v1/reviews/"+review+"/evidence",
			map[string]any{"kind": "receipt", "content_type": "image/jpeg"}, author.Access)
		require.Equal(t, http.StatusCreated, status, "%v", res)
	}
	status, _ = a.do("POST", "/api/v1/reviews/"+review+"/evidence",
		map[string]any{"kind": "receipt", "content_type": "image/jpeg"}, author.Access)
	require.Equal(t, http.StatusConflict, status)

	_, err = a.pool.Exec(context.Background(), `
		UPDATE review_evidence SET created_at = now() - interval '1 hour'
		WHERE review_id = $1 AND status = 'staged'`, review)
	require.NoError(t, err)
	status, res = a.do("POST", "/api/v1/reviews/"+review+"/evidence",
		map[string]any{"kind": "receipt", "content_type": "image/jpeg"}, author.Access)
	require.Equal(t, http.StatusCreated, status, "%v", res)

	var evidenceRows int
	err = a.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM review_evidence WHERE review_id = $1`, review).Scan(&evidenceRows)
	require.NoError(t, err)
	assert.Equal(t, 1, evidenceRows, "expired staged evidence rows should be removed")
}

func TestSearchAmharicAndFilters(t *testing.T) {
	a := newTestAPI(t)
	mod := a.register("mod")
	a.grantRoles(&mod, "moderator")
	a.createTarget(mod, "Selam Online Store", catElectronicsID, "online_seller", "ሰላም ኦንላይን")
	a.createTarget(mod, "Tomoca Coffee", catRestaurantID, "cafe", "ቶሞካ ቡና")
	a.createTarget(mod, "Sheger Phone Repair", catElectronicsID, "repair_provider")

	find := func(q string, extra string) []string {
		status, res := a.do("GET", "/api/v1/search/targets?q="+q+extra, nil, "")
		require.Equal(t, http.StatusOK, status, "%v", res)
		var names []string
		for _, item := range dataList(res) {
			names = append(names, item.(map[string]any)["name"].(string))
		}
		return names
	}

	// Amharic alias lookup.
	assert.Contains(t, find("%E1%89%B6%E1%88%9E%E1%8A%AB", ""), "Tomoca Coffee") // ቶሞካ
	// Homophone folding: ሠላም (with ሠ) must find the ሰላም alias.
	assert.Contains(t, find("%E1%88%A0%E1%88%8B%E1%88%9D", ""), "Selam Online Store") // ሠላም
	// Latin typo tolerance.
	assert.Contains(t, find("shegar+phone", ""), "Sheger Phone Repair")
	// Category filter excludes other categories.
	names := find("selam", "&category="+catRestaurantID)
	assert.NotContains(t, names, "Selam Online Store")
	// Type filter.
	names = find("tomoca", "&type=cafe")
	assert.Contains(t, names, "Tomoca Coffee")

	// Query length bounds.
	status, _ := a.do("GET", "/api/v1/search/targets?q=x", nil, "")
	assert.Equal(t, http.StatusUnprocessableEntity, status)
}

func TestTargetReferenceValidation(t *testing.T) {
	a := newTestAPI(t)
	mod := a.register("targetrefs")
	a.grantRoles(&mod, "moderator")

	status, _ := a.do("POST", "/api/v1/targets", map[string]any{
		"target_type": "restaurant",
		"category_id": "11111111-1111-4111-8111-000000000000",
		"name":        "Bad Category Cafe",
	}, mod.Access)
	assert.Equal(t, http.StatusNotFound, status)

	status, _ = a.do("POST", "/api/v1/targets", map[string]any{
		"target_type": "restaurant",
		"category_id": catRestaurantID,
		"name":        "Bad Website Cafe",
		"website":     "http://example.com",
	}, mod.Access)
	assert.Equal(t, http.StatusUnprocessableEntity, status)

	otherCity := "33333333-3333-4333-8333-333333333301"
	otherArea := "33333333-3333-4333-8333-333333333302"
	_, err := a.pool.Exec(context.Background(), `
		INSERT INTO cities (id, name, name_am, country, active) VALUES ($1, 'Hawassa', 'ሀዋሳ', 'ET', true);
		INSERT INTO areas (id, city_id, name, name_am, active) VALUES ($2, $1, 'Piazza', 'ፒያሳ', true)`,
		otherCity, otherArea)
	require.NoError(t, err)

	status, _ = a.do("POST", "/api/v1/targets", map[string]any{
		"target_type": "restaurant",
		"category_id": catRestaurantID,
		"name":        "Mismatched Area Cafe",
		"city_id":     addisCityID,
		"area_id":     otherArea,
	}, mod.Access)
	assert.Equal(t, http.StatusUnprocessableEntity, status)
}

func TestClaimsResponsesAndModeration(t *testing.T) {
	a := newTestAPI(t)
	mod := a.register("mod")
	a.grantRoles(&mod, "moderator")

	owner := a.register("owner")
	reviewer := a.register("reviewer")

	// Owner creates the business profile; creation grants no control.
	status, res := a.do("POST", "/api/v1/businesses",
		map[string]any{"name": "Claims Bistro PLC", "description": "test"}, owner.Access)
	require.Equal(t, http.StatusCreated, status)
	businessID := data(res)["id"].(string)

	// Link a published target to the business.
	status, res = a.do("POST", "/api/v1/targets", map[string]any{
		"target_type": "restaurant", "category_id": catRestaurantID,
		"name": "Claims Bistro Bole", "business_id": businessID,
	}, mod.Access)
	require.Equal(t, http.StatusCreated, status)
	targetID := data(res)["id"].(string)
	reviewID := a.review(reviewer, targetID, 2, nil)

	t.Run("no response rights before claim approval", func(t *testing.T) {
		status, _ := a.do("POST", "/api/v1/reviews/"+reviewID+"/response",
			map[string]any{"body": "We are sorry!"}, owner.Access)
		assert.Equal(t, http.StatusForbidden, status)
	})

	t.Run("claim then respond", func(t *testing.T) {
		status, res := a.do("POST", "/api/v1/businesses/"+businessID+"/claims",
			map[string]any{"method": "document", "message": "trade license attached"}, owner.Access)
		require.Equal(t, http.StatusCreated, status)
		claimID := data(res)["id"].(string)

		// Duplicate pending claim blocked.
		status, _ = a.do("POST", "/api/v1/businesses/"+businessID+"/claims",
			map[string]any{"method": "document", "message": "again"}, owner.Access)
		assert.Equal(t, http.StatusConflict, status)

		// Customer cannot decide claims.
		status, _ = a.do("POST", "/api/v1/moderation/claims/"+claimID+"/decision",
			map[string]any{"decision": "approved"}, owner.Access)
		assert.Equal(t, http.StatusForbidden, status)

		status, res = a.do("POST", "/api/v1/moderation/claims/"+claimID+"/decision",
			map[string]any{"decision": "approved", "note": "license verified"}, mod.Access)
		require.Equal(t, http.StatusOK, status)
		assert.Equal(t, "approved", data(res)["status"])

		// Approval grants response rights (one response per review).
		status, res = a.do("POST", "/api/v1/reviews/"+reviewID+"/response",
			map[string]any{"body": "We are sorry — please give us another chance."}, owner.Access)
		require.Equal(t, http.StatusCreated, status)
		responseID := data(res)["id"].(string)
		status, _ = a.do("POST", "/api/v1/reviews/"+reviewID+"/response",
			map[string]any{"body": "double"}, owner.Access)
		assert.Equal(t, http.StatusConflict, status)

		// Edit keeps an audit trail of the previous body.
		status, _ = a.do("PUT", "/api/v1/responses/"+responseID,
			map[string]any{"body": "Updated response."}, owner.Access)
		require.Equal(t, http.StatusOK, status)
		var edits int
		err := a.pool.QueryRow(context.Background(),
			`SELECT count(*) FROM business_response_edits WHERE response_id = $1`, responseID).Scan(&edits)
		require.NoError(t, err)
		assert.Equal(t, 1, edits)

		// A stranger cannot edit the business response (object-level authz).
		stranger := a.register("resp-stranger")
		status, _ = a.do("PUT", "/api/v1/responses/"+responseID,
			map[string]any{"body": "hacked"}, stranger.Access)
		assert.Equal(t, http.StatusForbidden, status)

		// The business cannot delete or alter the customer review itself:
		// no such endpoint exists; owner editing the review is forbidden.
		status, _ = a.do("PUT", "/api/v1/reviews/"+reviewID, map[string]any{
			"overall_rating": 5, "body": "Actually we loved it! Great place, changed my mind.",
			"version": 1,
		}, owner.Access)
		assert.Equal(t, http.StatusForbidden, status)
	})

	t.Run("reports lifecycle", func(t *testing.T) {
		// The owner reports the review for moderation.
		status, res := a.do("POST", "/api/v1/reviews/"+reviewID+"/reports",
			map[string]any{"reason": "fake_experience", "details": "competitor, never visited"}, owner.Access)
		require.Equal(t, http.StatusCreated, status)
		reportID := data(res)["id"].(string)

		// Duplicate open report by the same user blocked.
		status, _ = a.do("POST", "/api/v1/reviews/"+reviewID+"/reports",
			map[string]any{"reason": "spam", "details": "again"}, owner.Access)
		assert.Equal(t, http.StatusConflict, status)

		// Reporter sees status; review stays published (reports never
		// auto-hide).
		status, res = a.do("GET", "/api/v1/users/me/reports", nil, owner.Access)
		require.Equal(t, http.StatusOK, status)
		require.Len(t, dataList(res), 1)
		status, _ = a.do("GET", "/api/v1/reviews/"+reviewID, nil, "")
		assert.Equal(t, http.StatusOK, status)

		// Moderator queue, resolve, audit.
		status, res = a.do("GET", "/api/v1/moderation/reports?status=open", nil, mod.Access)
		require.Equal(t, http.StatusOK, status)
		require.NotEmpty(t, dataList(res))
		status, _ = a.do("POST", "/api/v1/moderation/reports/"+reportID+"/resolve",
			map[string]any{"status": "dismissed", "note": "review is genuine"}, mod.Access)
		require.Equal(t, http.StatusOK, status)
		status, res = a.do("GET", "/api/v1/moderation/audit?subject_type=report&subject_id="+reportID, nil, mod.Access)
		require.Equal(t, http.StatusOK, status)
		assert.NotEmpty(t, dataList(res))
	})

	t.Run("claim revocation removes control", func(t *testing.T) {
		status, res := a.do("GET", "/api/v1/claims/mine", nil, owner.Access)
		require.Equal(t, http.StatusOK, status)
		claimID := dataList(res)[0].(map[string]any)["id"].(string)

		status, _ = a.do("POST", "/api/v1/moderation/claims/"+claimID+"/revoke",
			map[string]any{"note": "ownership dispute"}, mod.Access)
		require.Equal(t, http.StatusOK, status)

		status, _ = a.do("GET", "/api/v1/businesses/"+businessID+"/stats", nil, owner.Access)
		assert.Equal(t, http.StatusForbidden, status)
	})

	t.Run("activity inbox and outbox", func(t *testing.T) {
		status, res := a.do("GET", "/api/v1/users/me/notifications", nil, owner.Access)
		require.Equal(t, http.StatusOK, status)
		ownerItems := dataList(res)
		require.Len(t, ownerItems, 3)

		eventTypes := make(map[string]bool, len(ownerItems))
		for _, raw := range ownerItems {
			item := raw.(map[string]any)
			eventTypes[item["event_type"].(string)] = true
			assert.Nil(t, item["read_at"])
		}
		assert.True(t, eventTypes["claim.approved"])
		assert.True(t, eventTypes["report.dismissed"])
		assert.True(t, eventTypes["claim.revoked"])

		status, res = a.do("GET", "/api/v1/users/me/notifications/unread-count", nil, owner.Access)
		require.Equal(t, http.StatusOK, status)
		assert.EqualValues(t, 3, data(res)["unread_count"])

		status, res = a.do("GET", "/api/v1/users/me/notifications", nil, reviewer.Access)
		require.Equal(t, http.StatusOK, status)
		reviewerItems := dataList(res)
		require.Len(t, reviewerItems, 1)
		assert.Equal(t, "response.created", reviewerItems[0].(map[string]any)["event_type"])

		ownerNotificationID := ownerItems[0].(map[string]any)["id"].(string)
		status, _ = a.do("PUT", "/api/v1/users/me/notifications/"+ownerNotificationID+"/read", nil, reviewer.Access)
		assert.Equal(t, http.StatusNotFound, status, "users must not mutate another user's inbox")

		status, _ = a.do("PUT", "/api/v1/users/me/notifications/"+ownerNotificationID+"/read", nil, owner.Access)
		require.Equal(t, http.StatusOK, status)
		status, res = a.do("GET", "/api/v1/users/me/notifications?unread=true", nil, owner.Access)
		require.Equal(t, http.StatusOK, status)
		assert.Len(t, dataList(res), 2)

		status, res = a.do("PUT", "/api/v1/users/me/notifications/read-all", nil, owner.Access)
		require.Equal(t, http.StatusOK, status)
		assert.EqualValues(t, 2, data(res)["updated"])
		status, res = a.do("GET", "/api/v1/users/me/notifications/unread-count", nil, owner.Access)
		require.Equal(t, http.StatusOK, status)
		assert.EqualValues(t, 0, data(res)["unread_count"])

		var notifications, pendingOutbox int
		err := a.pool.QueryRow(context.Background(), `SELECT count(*) FROM notifications`).Scan(&notifications)
		require.NoError(t, err)
		err = a.pool.QueryRow(context.Background(), `SELECT count(*) FROM notification_outbox WHERE processed_at IS NULL`).Scan(&pendingOutbox)
		require.NoError(t, err)
		assert.Equal(t, 4, notifications)
		assert.Equal(t, notifications, pendingOutbox, "every inbox item needs a durable outbox event")
	})
}

func TestAdminMergeTargets(t *testing.T) {
	a := newTestAPI(t)
	adminUser := a.register("admin")
	a.grantRoles(&adminUser, "admin")

	src := a.createTarget(adminUser, "Duplicate Cafe", catRestaurantID, "cafe", "ዱፕ ካፌ")
	dst := a.createTarget(adminUser, "Original Cafe", catRestaurantID, "cafe")

	u1, u2, u3 := a.register("m1"), a.register("m2"), a.register("m3")
	a.review(u1, src, 5, nil)
	a.review(u2, src, 3, nil)
	a.review(u3, dst, 4, nil)

	status, _ := a.do("POST", "/api/v1/admin/targets/"+src+"/merge",
		map[string]any{"into_id": dst, "note": "same place"}, adminUser.Access)
	require.Equal(t, http.StatusOK, status)

	// Destination absorbed the reviews and recomputed stats.
	s := a.stats(dst)
	assert.EqualValues(t, 3, s["review_count"])
	assert.InDelta(t, 4.0, s["average"].(float64), 0.001) // (5+3+4)/3

	// Source is gone from public view; its name became an alias of dest.
	status, _ = a.do("GET", "/api/v1/targets/"+src, nil, "")
	assert.Equal(t, http.StatusNotFound, status)
	status, res := a.do("GET", "/api/v1/search/targets?q=duplicate+cafe", nil, "")
	require.Equal(t, http.StatusOK, status)
	found := false
	for _, item := range dataList(res) {
		if item.(map[string]any)["id"].(string) == dst {
			found = true
		}
	}
	assert.True(t, found, "searching the old name finds the merged destination")
}

func TestAdminUserSuspension(t *testing.T) {
	a := newTestAPI(t)
	adminUser := a.register("admin")
	a.grantRoles(&adminUser, "admin")
	victim := a.register("suspended")

	status, _ := a.do("POST", "/api/v1/admin/users/"+victim.ID+"/suspend",
		map[string]any{"note": "abuse"}, adminUser.Access)
	require.Equal(t, http.StatusOK, status)

	// Refresh dead immediately; login blocked with a distinct code.
	status, _ = a.do("POST", "/api/v1/auth/refresh", map[string]any{"refresh_token": victim.Refresh}, "")
	assert.Equal(t, http.StatusUnauthorized, status)
	status, _ = a.do("GET", "/api/v1/users/me", nil, victim.Access)
	assert.Equal(t, http.StatusUnauthorized, status, "suspended user's existing access token must stop working immediately")
	status, res := a.do("POST", "/api/v1/auth/login",
		map[string]any{"identifier": victim.Email, "password": "password123"}, "")
	assert.Equal(t, http.StatusForbidden, status)
	assert.Equal(t, "account_suspended", res["error"].(map[string]any)["code"])

	// Self-suspension refused.
	status, _ = a.do("POST", "/api/v1/admin/users/"+adminUser.ID+"/suspend",
		map[string]any{"note": "oops"}, adminUser.Access)
	assert.Equal(t, http.StatusUnprocessableEntity, status)

	// Reinstate restores login.
	status, _ = a.do("POST", "/api/v1/admin/users/"+victim.ID+"/reinstate",
		map[string]any{"note": "appeal accepted"}, adminUser.Access)
	require.Equal(t, http.StatusOK, status)
	status, _ = a.do("POST", "/api/v1/auth/login",
		map[string]any{"identifier": victim.Email, "password": "password123"}, "")
	assert.Equal(t, http.StatusOK, status)
}

func TestIdempotentReviewCreation(t *testing.T) {
	a := newTestAPI(t)
	mod := a.register("mod")
	a.grantRoles(&mod, "moderator")
	target := a.createTarget(mod, "Idem Restaurant", catRestaurantID, "restaurant")
	u := a.register("idem")

	body := map[string]any{
		"target_id": target, "overall_rating": 4,
		"body": "An idempotency test body which is long enough to pass.",
	}
	status, res := a.do("POST", "/api/v1/reviews", body, u.Access, "Idempotency-Key", "key-1")
	require.Equal(t, http.StatusCreated, status)
	first := data(res)["id"].(string)

	// Replay: same review, no duplicate, aggregates unchanged.
	status, res = a.do("POST", "/api/v1/reviews", body, u.Access, "Idempotency-Key", "key-1")
	require.Equal(t, http.StatusCreated, status)
	assert.Equal(t, first, data(res)["id"].(string))
	assert.EqualValues(t, 1, a.stats(target)["review_count"])

	// Same key + different body = conflict.
	body["overall_rating"] = 5
	status, _ = a.do("POST", "/api/v1/reviews", body, u.Access, "Idempotency-Key", "key-1")
	assert.Equal(t, http.StatusConflict, status)
}

func TestPublicReadCaching(t *testing.T) {
	a := newTestAPI(t)
	resp, err := a.srv.Client().Get(a.srv.URL + "/api/v1/categories")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Cache-Control"), "public")
	etag := resp.Header.Get("ETag")
	require.NotEmpty(t, etag)

	req, err := http.NewRequest(http.MethodGet, a.srv.URL+"/api/v1/categories", nil)
	require.NoError(t, err)
	req.Header.Set("If-None-Match", etag)
	resp, err = a.srv.Client().Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusNotModified, resp.StatusCode)

	u := a.register("cache-auth")
	req, err = http.NewRequest(http.MethodGet, a.srv.URL+"/api/v1/categories", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+u.Access)
	resp, err = a.srv.Client().Do(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, "no-store", resp.Header.Get("Cache-Control"))
	require.Empty(t, resp.Header.Get("ETag"))
}

func TestListingSortsAndPagination(t *testing.T) {
	a := newTestAPI(t)
	mod := a.register("mod")
	a.grantRoles(&mod, "moderator")
	target := a.createTarget(mod, "Sorting Restaurant", catRestaurantID, "restaurant")

	ratings := []int{5, 1, 3, 4, 2}
	for i, r := range ratings {
		u := a.register(fmt.Sprintf("sort%d", i))
		a.review(u, target, r, map[string]any{"language": []string{"am", "en"}[i%2]})
	}

	fetch := func(query string) []int {
		status, res := a.do("GET", "/api/v1/targets/"+target+"/reviews?"+query, nil, "")
		require.Equal(t, http.StatusOK, status, "%v", res)
		var out []int
		for _, item := range dataList(res) {
			out = append(out, int(item.(map[string]any)["overall_rating"].(float64)))
		}
		return out
	}

	assert.Equal(t, []int{5, 4, 3, 2, 1}, fetch("sort=highest"))
	assert.Equal(t, []int{1, 2, 3, 4, 5}, fetch("sort=lowest"))
	assert.Len(t, fetch("rating=3"), 1)
	assert.Len(t, fetch("language=am"), 3)

	// Cursor pagination walks the full set exactly once.
	var all []int
	cursor := ""
	for {
		q := "sort=newest&limit=2"
		if cursor != "" {
			q += "&cursor=" + cursor
		}
		status, res := a.do("GET", "/api/v1/targets/"+target+"/reviews?"+q, nil, "")
		require.Equal(t, http.StatusOK, status)
		for _, item := range dataList(res) {
			all = append(all, int(item.(map[string]any)["overall_rating"].(float64)))
		}
		meta := res["meta"].(map[string]any)
		if meta["has_more"] != true {
			break
		}
		cursor = meta["next_cursor"].(string)
	}
	assert.Len(t, all, 5, "pagination covers every review exactly once")
}

// TestRateLimiting builds a limiter-enabled instance and verifies sensitive
// endpoints throttle.
func TestRateLimiting(t *testing.T) {
	pool := testdb.New(t)
	cfg := testConfig()
	cfg.RateLimitEnabled = true
	codes := &captureProvider{codes: map[string]string{}}
	srv := httptest.NewServer(BuildAPI(cfg, pool, storage.NewMemory(), codes))
	t.Cleanup(srv.Close)

	var last int
	for i := 0; i < 8; i++ {
		req, err := http.NewRequest("POST", srv.URL+"/api/v1/auth/login",
			bytes.NewReader([]byte(`{"identifier":"nobody@example.com","password":"wrongwrong"}`)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := srv.Client().Do(req)
		require.NoError(t, err)
		last = resp.StatusCode
		resp.Body.Close()
	}
	assert.Equal(t, http.StatusTooManyRequests, last, "credential stuffing must hit the limiter")
}

// TestDiscoveryListings covers the browse, top-rated, and trending endpoints
// end-to-end (a parameter-typing bug in top-rated once slipped past unit
// coverage — keep these).
func TestDiscoveryListings(t *testing.T) {
	a := newTestAPI(t)
	mod := a.register("mod")
	a.grantRoles(&mod, "moderator")

	good := a.createTarget(mod, "Discovery Good Cafe", catRestaurantID, "cafe")
	lone := a.createTarget(mod, "Discovery One-Hit Wonder", catRestaurantID, "cafe")

	// good: three 4-5★ reviews; lone: a single 5★ review.
	for i, rating := range []int{5, 4, 5} {
		u := a.register(fmt.Sprintf("disc%d", i))
		a.review(u, good, rating, nil)
	}
	u := a.register("disc-lone")
	a.review(u, lone, 5, nil)

	// Browse with cursor pagination.
	status, res := a.do("GET", "/api/v1/targets?type=cafe&limit=1", nil, "")
	require.Equal(t, http.StatusOK, status)
	require.Len(t, dataList(res), 1)
	require.Equal(t, true, res["meta"].(map[string]any)["has_more"])

	// Top-rated: Bayesian shrinkage must rank consistent "good" above the
	// single-5★ "lone" despite lone's higher raw average.
	status, res = a.do("GET", "/api/v1/targets/top-rated?type=cafe", nil, "")
	require.Equal(t, http.StatusOK, status, "%v", res)
	items := dataList(res)
	require.GreaterOrEqual(t, len(items), 2)
	first := items[0].(map[string]any)
	assert.Equal(t, good, first["id"], "three consistent reviews outrank one 5-star")
	assert.Greater(t, first["ranking_score"].(float64), items[1].(map[string]any)["ranking_score"].(float64))

	// Trending: both appear (recent activity), ordered by volume.
	status, res = a.do("GET", "/api/v1/targets/trending?type=cafe", nil, "")
	require.Equal(t, http.StatusOK, status)
	trend := dataList(res)
	require.GreaterOrEqual(t, len(trend), 2)
	assert.Equal(t, good, trend[0].(map[string]any)["id"])
	assert.EqualValues(t, 3, trend[0].(map[string]any)["recent_review_count"])
}
