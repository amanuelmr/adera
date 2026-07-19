// Package storage abstracts S3-compatible object storage. MinIO backs local
// development; any S3-compatible service works in production. Uploads use
// presigned POST policies because (unlike presigned PUT) they enforce
// content-type and size server-side.
package storage

import (
	"context"
	"io"
	"time"
)

// PresignedPost is everything a client needs to upload one object directly
// to object storage, bypassing the API.
type PresignedPost struct {
	URL       string            `json:"url"`
	Fields    map[string]string `json:"fields"`
	Key       string            `json:"key"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// ObjectInfo describes a stored object.
type ObjectInfo struct {
	Size        int64
	ContentType string
}

// Store is the object-storage abstraction used by the media module.
type Store interface {
	// EnsureBuckets creates the public and private buckets if missing
	// (development convenience; production buckets are provisioned).
	EnsureBuckets(ctx context.Context) error

	// PresignUpload returns a POST policy for a server-chosen key with
	// exact content-type and a size range enforced by the storage service.
	PresignUpload(ctx context.Context, bucket, key, contentType string, minSize, maxSize int64, expiry time.Duration) (PresignedPost, error)

	// PresignGet returns a short-lived download URL (private evidence access).
	PresignGet(ctx context.Context, bucket, key string, expiry time.Duration) (string, error)

	Stat(ctx context.Context, bucket, key string) (ObjectInfo, error)
	Get(ctx context.Context, bucket, key string) (io.ReadCloser, error)
	Put(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) error
	Remove(ctx context.Context, bucket, key string) error
}
