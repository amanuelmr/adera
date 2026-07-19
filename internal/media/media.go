// Package media implements the two-phase upload flow for review photos and
// private verification evidence.
//
// Security model:
//   - Clients never upload through the API; they receive a presigned POST
//     policy (server-chosen key, exact content type, size range, 5-minute
//     expiry) and upload directly to object storage.
//   - EVERYTHING stages into the PRIVATE bucket. Nothing a user uploaded is
//     ever publicly readable as-is.
//   - Finalizing PUBLIC review media validates the file signature, decodes,
//     and re-encodes the image — which strips all EXIF/GPS metadata — then
//     writes the clean copy to the public bucket and deletes the staging
//     object.
//   - Finalizing PRIVATE evidence validates the signature only (originals are
//     preserved for verification integrity) and never leaves the private
//     bucket; access is via short-lived presigned GETs for the owner and
//     moderators only.
package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/adera-platform/backend/internal/platform/storage"
	"github.com/adera-platform/backend/internal/platform/web"
	"github.com/adera-platform/backend/internal/reviews"
)

const (
	MinUploadBytes = 1 << 10  // 1 KiB
	MaxUploadBytes = 10 << 20 // 10 MiB
	PresignTTL     = 5 * time.Minute
	MaxMediaPerReview    = 5
	MaxEvidencePerReview = 5
	// Decompression-bomb guard.
	maxPixels = 40_000_000 // ~40 MP
)

// Content types accepted for upload. Public media must decode as JPEG/PNG at
// finalize (WebP is accepted for private evidence, where no re-encode occurs).
var publicContentTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
}

var evidenceContentTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
}

// EvidenceKinds mirror the schema constraint.
var EvidenceKinds = map[string]bool{
	"receipt": true, "order_screenshot": true, "product_photo": true,
	"service_result": true, "location_qr": true,
}

// Service coordinates DB rows and object storage.
type Service struct {
	pool          *pgxpool.Pool
	store         storage.Store
	reviewsRepo   *reviews.Repo
	privateBucket string
	publicBucket  string
}

func NewService(pool *pgxpool.Pool, store storage.Store, reviewsRepo *reviews.Repo, privateBucket, publicBucket string) *Service {
	return &Service{pool: pool, store: store, reviewsRepo: reviewsRepo,
		privateBucket: privateBucket, publicBucket: publicBucket}
}

var errStorageDisabled = &web.Error{Status: 503, Code: "storage_unavailable",
	Message: "media uploads are not available: object storage is not configured"}

// requireOwnReview loads the review and enforces authorship.
func (s *Service) requireOwnReview(ctx context.Context, reviewID, userID uuid.UUID) error {
	var authorID uuid.UUID
	var status string
	err := s.pool.QueryRow(ctx, `
		SELECT user_id, moderation_status FROM reviews WHERE id = $1`, reviewID).Scan(&authorID, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return web.ErrNotFound("review")
	}
	if err != nil {
		return fmt.Errorf("loading review: %w", err)
	}
	if authorID != userID {
		return web.ErrForbidden("you can only attach files to your own review")
	}
	if status == reviews.StatusRemoved || status == reviews.StatusRejected {
		return web.ErrConflict("this review can no longer be modified")
	}
	return nil
}

// UploadTicket is returned by presign operations.
type UploadTicket struct {
	UploadID uuid.UUID             `json:"upload_id"`
	Upload   storage.PresignedPost `json:"upload"`
}

// PresignReviewMedia stages a public review photo upload.
func (s *Service) PresignReviewMedia(ctx context.Context, reviewID, userID uuid.UUID, contentType string) (UploadTicket, error) {
	if s.store == nil {
		return UploadTicket{}, errStorageDisabled
	}
	if _, ok := publicContentTypes[contentType]; !ok {
		return UploadTicket{}, web.ErrValidation("invalid upload").
			WithDetail("content_type", "must be image/jpeg or image/png")
	}
	if err := s.requireOwnReview(ctx, reviewID, userID); err != nil {
		return UploadTicket{}, err
	}
	var count int
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM review_media WHERE review_id = $1 AND status <> 'removed'`, reviewID).Scan(&count); err != nil {
		return UploadTicket{}, fmt.Errorf("counting media: %w", err)
	}
	if count >= MaxMediaPerReview {
		return UploadTicket{}, web.ErrConflict(fmt.Sprintf("at most %d photos per review", MaxMediaPerReview))
	}

	id := uuid.New()
	stagingKey := fmt.Sprintf("staging/media/%s/%s.%s", userID, id, publicContentTypes[contentType])
	post, err := s.store.PresignUpload(ctx, s.privateBucket, stagingKey, contentType, MinUploadBytes, MaxUploadBytes, PresignTTL)
	if err != nil {
		return UploadTicket{}, fmt.Errorf("presigning media upload: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO review_media (id, review_id, object_key, content_type, status)
		VALUES ($1, $2, $3, $4, 'staged')`, id, reviewID, stagingKey, contentType); err != nil {
		return UploadTicket{}, fmt.Errorf("recording staged media: %w", err)
	}
	return UploadTicket{UploadID: id, Upload: post}, nil
}

// FinalizeReviewMedia validates and sanitizes a staged public photo:
// signature check, decode with bomb guard, re-encode (drops EXIF/GPS), copy
// to the public bucket, delete staging, mark ready, and raise the review's
// verification level to media_attached.
func (s *Service) FinalizeReviewMedia(ctx context.Context, uploadID, userID uuid.UUID) (uuid.UUID, string, error) {
	if s.store == nil {
		return uuid.Nil, "", errStorageDisabled
	}
	var (
		reviewID    uuid.UUID
		stagingKey  string
		contentType string
		status      string
		createdAt   time.Time
	)
	err := s.pool.QueryRow(ctx, `
		SELECT review_id, object_key, content_type, status, created_at
		FROM review_media WHERE id = $1`, uploadID).
		Scan(&reviewID, &stagingKey, &contentType, &status, &createdAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", web.ErrNotFound("upload")
	}
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("loading staged media: %w", err)
	}
	if err := s.requireOwnReview(ctx, reviewID, userID); err != nil {
		return uuid.Nil, "", err
	}
	if status != "staged" {
		return uuid.Nil, "", web.ErrConflict("this upload is already finalized")
	}
	if time.Since(createdAt) > 2*PresignTTL {
		return uuid.Nil, "", &web.Error{Status: 410, Code: web.CodeGone, Message: "upload window expired; request a new upload"}
	}

	clean, format, err := s.sanitizeImage(ctx, stagingKey, contentType)
	if err != nil {
		return uuid.Nil, "", err
	}
	finalKey := fmt.Sprintf("reviews/%s/%s.%s", reviewID, uploadID, format)
	finalType := "image/" + map[string]string{"jpg": "jpeg", "png": "png"}[format]
	if err := s.store.Put(ctx, s.publicBucket, finalKey, bytes.NewReader(clean), int64(len(clean)), finalType); err != nil {
		return uuid.Nil, "", fmt.Errorf("writing public media: %w", err)
	}
	if err := s.store.Remove(ctx, s.privateBucket, stagingKey); err != nil {
		return uuid.Nil, "", fmt.Errorf("removing staged object: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE review_media SET object_key = $2, content_type = $3, size_bytes = $4, status = 'ready'
		WHERE id = $1`, uploadID, finalKey, finalType, len(clean)); err != nil {
		return uuid.Nil, "", fmt.Errorf("marking media ready: %w", err)
	}
	if err := s.reviewsRepo.UpgradeVerification(ctx, reviewID, reviews.VerifyMedia); err != nil {
		return uuid.Nil, "", err
	}
	return uploadID, finalKey, nil
}

// sanitizeImage downloads a staged object, verifies its magic bytes match the
// declared type, decodes it defensively, and re-encodes it clean.
func (s *Service) sanitizeImage(ctx context.Context, key, declaredType string) ([]byte, string, error) {
	obj, err := s.store.Get(ctx, s.privateBucket, key)
	if err != nil {
		return nil, "", &web.Error{Status: 422, Code: web.CodeUploadInvalid,
			Message: "the uploaded file was not found; upload before finalizing", Internal: err}
	}
	defer obj.Close()
	raw, err := io.ReadAll(io.LimitReader(obj, MaxUploadBytes+1))
	if err != nil {
		return nil, "", fmt.Errorf("reading staged object: %w", err)
	}
	if len(raw) > MaxUploadBytes || len(raw) < MinUploadBytes {
		return nil, "", uploadInvalid("file size out of bounds")
	}
	if SniffImageType(raw) != declaredType {
		return nil, "", uploadInvalid("file content does not match its declared type")
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, "", uploadInvalid("file is not a decodable image")
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width*cfg.Height > maxPixels {
		return nil, "", uploadInvalid("image dimensions out of bounds")
	}
	img, format, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, "", uploadInvalid("file is not a decodable image")
	}
	// Re-encoding from the decoded pixel data drops every metadata segment
	// (EXIF, GPS, XMP) by construction.
	var buf bytes.Buffer
	switch format {
	case "jpeg":
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
			return nil, "", fmt.Errorf("re-encoding jpeg: %w", err)
		}
		return buf.Bytes(), "jpg", nil
	case "png":
		if err := png.Encode(&buf, img); err != nil {
			return nil, "", fmt.Errorf("re-encoding png: %w", err)
		}
		return buf.Bytes(), "png", nil
	default:
		return nil, "", uploadInvalid("unsupported image format")
	}
}

func uploadInvalid(msg string) *web.Error {
	return &web.Error{Status: 422, Code: web.CodeUploadInvalid, Message: msg}
}

// SniffImageType detects the real content type from magic bytes. Exported for
// fuzz testing.
func SniffImageType(b []byte) string {
	switch {
	case len(b) >= 3 && b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF:
		return "image/jpeg"
	case len(b) >= 8 && bytes.Equal(b[:8], []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}):
		return "image/png"
	case len(b) >= 12 && bytes.Equal(b[:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")):
		return "image/webp"
	default:
		return ""
	}
}

// PresignEvidence stages a private verification-evidence upload.
func (s *Service) PresignEvidence(ctx context.Context, reviewID, userID uuid.UUID, kind, contentType string) (UploadTicket, error) {
	if s.store == nil {
		return UploadTicket{}, errStorageDisabled
	}
	if !EvidenceKinds[kind] {
		return UploadTicket{}, web.ErrValidation("invalid evidence").WithDetail("kind", "unsupported evidence kind")
	}
	ext, ok := evidenceContentTypes[contentType]
	if !ok {
		return UploadTicket{}, web.ErrValidation("invalid evidence").
			WithDetail("content_type", "must be image/jpeg, image/png, or image/webp")
	}
	if err := s.requireOwnReview(ctx, reviewID, userID); err != nil {
		return UploadTicket{}, err
	}
	var count int
	if err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM review_evidence WHERE review_id = $1`, reviewID).Scan(&count); err != nil {
		return UploadTicket{}, fmt.Errorf("counting evidence: %w", err)
	}
	if count >= MaxEvidencePerReview {
		return UploadTicket{}, web.ErrConflict(fmt.Sprintf("at most %d evidence files per review", MaxEvidencePerReview))
	}

	id := uuid.New()
	key := fmt.Sprintf("evidence/%s/%s.%s", reviewID, id, ext)
	post, err := s.store.PresignUpload(ctx, s.privateBucket, key, contentType, MinUploadBytes, MaxUploadBytes, PresignTTL)
	if err != nil {
		return UploadTicket{}, fmt.Errorf("presigning evidence upload: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO review_evidence (id, review_id, kind, object_key, content_type, status)
		VALUES ($1, $2, $3, $4, $5, 'staged')`, id, reviewID, kind, key, contentType); err != nil {
		return UploadTicket{}, fmt.Errorf("recording staged evidence: %w", err)
	}
	return UploadTicket{UploadID: id, Upload: post}, nil
}

// FinalizeEvidence verifies the staged private object (signature + size) and
// marks it submitted for moderator verification. The original is preserved
// unmodified and remains private.
func (s *Service) FinalizeEvidence(ctx context.Context, evidenceID, userID uuid.UUID) error {
	if s.store == nil {
		return errStorageDisabled
	}
	var (
		reviewID    uuid.UUID
		key         string
		contentType string
		status      string
	)
	err := s.pool.QueryRow(ctx, `
		SELECT review_id, object_key, content_type, status FROM review_evidence WHERE id = $1`, evidenceID).
		Scan(&reviewID, &key, &contentType, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return web.ErrNotFound("evidence")
	}
	if err != nil {
		return fmt.Errorf("loading staged evidence: %w", err)
	}
	if err := s.requireOwnReview(ctx, reviewID, userID); err != nil {
		return err
	}
	if status != "staged" {
		return web.ErrConflict("this evidence is already submitted")
	}

	obj, err := s.store.Get(ctx, s.privateBucket, key)
	if err != nil {
		return uploadInvalid("the uploaded file was not found; upload before finalizing")
	}
	defer obj.Close()
	head, err := io.ReadAll(io.LimitReader(obj, 16))
	if err != nil {
		return fmt.Errorf("reading evidence head: %w", err)
	}
	if SniffImageType(head) != contentType {
		return uploadInvalid("file content does not match its declared type")
	}
	info, err := s.store.Stat(ctx, s.privateBucket, key)
	if err != nil {
		return fmt.Errorf("checking evidence object: %w", err)
	}
	if info.Size < MinUploadBytes || info.Size > MaxUploadBytes {
		return uploadInvalid("file size out of bounds")
	}
	if _, err := s.pool.Exec(ctx, `
		UPDATE review_evidence SET status = 'submitted', size_bytes = $2 WHERE id = $1`,
		evidenceID, info.Size); err != nil {
		return fmt.Errorf("marking evidence submitted: %w", err)
	}
	return nil
}

// EvidenceItem is what the owner and moderators see. AccessURL is a
// short-lived presigned GET; it is never included in public responses.
type EvidenceItem struct {
	ID        uuid.UUID `json:"id"`
	ReviewID  uuid.UUID `json:"review_id"`
	Kind      string    `json:"kind"`
	Status    string    `json:"status"`
	Note      string    `json:"note,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	AccessURL string    `json:"access_url,omitempty"`
}

// ListEvidence returns a review's evidence for its owner or a moderator,
// with presigned access URLs when includeURLs is set.
func (s *Service) ListEvidence(ctx context.Context, reviewID uuid.UUID, includeURLs bool) ([]EvidenceItem, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, review_id, kind, object_key, status, note, created_at
		FROM review_evidence WHERE review_id = $1 AND status <> 'staged'
		ORDER BY created_at`, reviewID)
	if err != nil {
		return nil, fmt.Errorf("querying evidence: %w", err)
	}
	defer rows.Close()
	var out []EvidenceItem
	var keys []string
	for rows.Next() {
		var it EvidenceItem
		var key string
		if err := rows.Scan(&it.ID, &it.ReviewID, &it.Kind, &key, &it.Status, &it.Note, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning evidence: %w", err)
		}
		out = append(out, it)
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating evidence: %w", err)
	}
	if includeURLs && s.store != nil {
		for i := range out {
			u, err := s.store.PresignGet(ctx, s.privateBucket, keys[i], 10*time.Minute)
			if err != nil {
				return nil, fmt.Errorf("presigning evidence access: %w", err)
			}
			out[i].AccessURL = u
		}
	}
	return out, nil
}
