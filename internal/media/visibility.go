package media

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/google/uuid"
)

// withheldPrefix is where a withheld photo lives in the private bucket.
const withheldPrefix = "withheld/"

// SyncVisibility moves review photos between buckets to match their review:
// a review that leaves public view takes its photos out of the public bucket
// (a URL someone kept would otherwise keep working), and a restored review
// brings them back. It handles up to limit photos and returns how many it
// moved; a photo that fails is logged and retried on the next run.
//
// Each move copies before it deletes, and records the new place only after
// both, so an interrupted move is simply redone.
func (s *Service) SyncVisibility(ctx context.Context, limit int) (int, error) {
	if s.store == nil {
		return 0, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT m.id, m.object_key, m.thumb_key, m.content_type, m.withheld
		FROM review_media m JOIN reviews r ON r.id = m.review_id
		WHERE m.status = 'ready' AND m.withheld <> (r.moderation_status <> 'published')
		ORDER BY m.created_at
		LIMIT $1`, limit)
	if err != nil {
		return 0, fmt.Errorf("finding photos to move: %w", err)
	}
	type photo struct {
		id          uuid.UUID
		key         string
		thumbKey    *string
		contentType string
		withheld    bool
	}
	var photos []photo
	for rows.Next() {
		var p photo
		if err := rows.Scan(&p.id, &p.key, &p.thumbKey, &p.contentType, &p.withheld); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scanning photo: %w", err)
		}
		photos = append(photos, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterating photos: %w", err)
	}

	moved := 0
	for _, p := range photos {
		keys := []string{p.key}
		if p.thumbKey != nil {
			keys = append(keys, *p.thumbKey)
		}
		var err error
		for _, key := range keys {
			if p.withheld {
				err = s.moveObject(ctx, s.privateBucket, withheldPrefix+key, s.publicBucket, key, p.contentType)
			} else {
				err = s.moveObject(ctx, s.publicBucket, key, s.privateBucket, withheldPrefix+key, p.contentType)
			}
			if err != nil {
				break
			}
		}
		if err == nil {
			_, err = s.pool.Exec(ctx, `UPDATE review_media SET withheld = $2 WHERE id = $1`, p.id, !p.withheld)
		}
		if err != nil {
			slog.ErrorContext(ctx, "moving review photo", "media_id", p.id, "withhold", !p.withheld, "error", err)
			continue
		}
		moved++
	}
	return moved, nil
}

// moveObject copies an object between buckets and removes the original. An
// original already gone with the copy in place means an earlier move got
// that far; it counts as done.
func (s *Service) moveObject(ctx context.Context, fromBucket, fromKey, toBucket, toKey, contentType string) error {
	data, err := s.readObject(ctx, fromBucket, fromKey)
	if err != nil {
		if _, statErr := s.store.Stat(ctx, toBucket, toKey); statErr == nil {
			return nil
		}
		return fmt.Errorf("reading %s/%s: %w", fromBucket, fromKey, err)
	}
	if err := s.store.Put(ctx, toBucket, toKey, bytes.NewReader(data), int64(len(data)), contentType); err != nil {
		return fmt.Errorf("writing %s/%s: %w", toBucket, toKey, err)
	}
	if err := s.store.Remove(ctx, fromBucket, fromKey); err != nil {
		return fmt.Errorf("removing %s/%s: %w", fromBucket, fromKey, err)
	}
	return nil
}

func (s *Service) readObject(ctx context.Context, bucket, key string) ([]byte, error) {
	obj, err := s.store.Get(ctx, bucket, key)
	if err != nil {
		return nil, err
	}
	defer func() { _ = obj.Close() }()
	data, err := io.ReadAll(io.LimitReader(obj, MaxUploadBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > MaxUploadBytes {
		return nil, errors.New("object larger than any upload")
	}
	return data, nil
}
