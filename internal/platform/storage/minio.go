package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioStore implements Store against MinIO or any S3-compatible endpoint.
type MinioStore struct {
	client        *minio.Client
	publicBucket  string
	privateBucket string
}

// NewMinio connects to an S3-compatible endpoint. EnsureBuckets creates the
// two buckets if missing and marks only the public one anonymously readable.
func NewMinio(endpoint, accessKey, secretKey string, useSSL bool, publicBucket, privateBucket string) (*MinioStore, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("creating storage client: %w", err)
	}
	return &MinioStore{client: client, publicBucket: publicBucket, privateBucket: privateBucket}, nil
}

func (s *MinioStore) EnsureBuckets(ctx context.Context) error {
	for _, name := range []string{s.publicBucket, s.privateBucket} {
		exists, err := s.client.BucketExists(ctx, name)
		if err != nil {
			return fmt.Errorf("checking bucket %s: %w", name, err)
		}
		if !exists {
			if err := s.client.MakeBucket(ctx, name, minio.MakeBucketOptions{}); err != nil {
				return fmt.Errorf("creating bucket %s: %w", name, err)
			}
		}
	}
	// Finalized review media is served directly from the public bucket; the
	// private bucket (staging + evidence) keeps the private-by-default policy.
	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [{
			"Effect": "Allow",
			"Principal": {"AWS": ["*"]},
			"Action": ["s3:GetObject"],
			"Resource": ["arn:aws:s3:::%s/*"]
		}]
	}`, s.publicBucket)
	if err := s.client.SetBucketPolicy(ctx, s.publicBucket, policy); err != nil {
		return fmt.Errorf("setting public bucket policy: %w", err)
	}
	return nil
}

func (s *MinioStore) PresignUpload(ctx context.Context, bucket, key, contentType string, minSize, maxSize int64, expiry time.Duration) (PresignedPost, error) {
	policy := minio.NewPostPolicy()
	if err := policy.SetBucket(bucket); err != nil {
		return PresignedPost{}, fmt.Errorf("post policy bucket: %w", err)
	}
	if err := policy.SetKey(key); err != nil {
		return PresignedPost{}, fmt.Errorf("post policy key: %w", err)
	}
	if err := policy.SetContentType(contentType); err != nil {
		return PresignedPost{}, fmt.Errorf("post policy content type: %w", err)
	}
	if err := policy.SetContentLengthRange(minSize, maxSize); err != nil {
		return PresignedPost{}, fmt.Errorf("post policy size range: %w", err)
	}
	expiresAt := time.Now().UTC().Add(expiry)
	if err := policy.SetExpires(expiresAt); err != nil {
		return PresignedPost{}, fmt.Errorf("post policy expiry: %w", err)
	}
	u, formData, err := s.client.PresignedPostPolicy(ctx, policy)
	if err != nil {
		return PresignedPost{}, fmt.Errorf("presigning upload: %w", err)
	}
	return PresignedPost{URL: u.String(), Fields: formData, Key: key, ExpiresAt: expiresAt}, nil
}

func (s *MinioStore) PresignGet(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	u, err := s.client.PresignedGetObject(ctx, bucket, key, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("presigning download: %w", err)
	}
	return u.String(), nil
}

func (s *MinioStore) Stat(ctx context.Context, bucket, key string) (ObjectInfo, error) {
	info, err := s.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return ObjectInfo{}, fmt.Errorf("stat object: %w", err)
	}
	return ObjectInfo{Size: info.Size, ContentType: info.ContentType}, nil
}

func (s *MinioStore) Get(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	obj, err := s.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("get object: %w", err)
	}
	return obj, nil
}

func (s *MinioStore) Put(ctx context.Context, bucket, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	return nil
}

func (s *MinioStore) Remove(ctx context.Context, bucket, key string) error {
	if err := s.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("remove object: %w", err)
	}
	return nil
}
