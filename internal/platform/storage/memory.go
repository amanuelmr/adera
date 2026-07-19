package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"
)

// MemoryStore is an in-memory Store for tests. Presigned URLs are synthetic;
// tests upload with Put directly and then exercise the finalize path.
type MemoryStore struct {
	mu      sync.RWMutex
	objects map[string]memObject // key: bucket + "/" + key
}

type memObject struct {
	data        []byte
	contentType string
}

func NewMemory() *MemoryStore {
	return &MemoryStore{objects: make(map[string]memObject)}
}

func (s *MemoryStore) EnsureBuckets(context.Context) error { return nil }

func (s *MemoryStore) PresignUpload(_ context.Context, bucket, key, contentType string, _, _ int64, expiry time.Duration) (PresignedPost, error) {
	return PresignedPost{
		URL:       "memory://" + bucket,
		Fields:    map[string]string{"key": key, "Content-Type": contentType},
		Key:       key,
		ExpiresAt: time.Now().UTC().Add(expiry),
	}, nil
}

func (s *MemoryStore) PresignGet(_ context.Context, bucket, key string, _ time.Duration) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.objects[bucket+"/"+key]; !ok {
		return "", fmt.Errorf("object %s/%s not found", bucket, key)
	}
	return "memory://" + bucket + "/" + key + "?signed=1", nil
}

func (s *MemoryStore) Stat(_ context.Context, bucket, key string) (ObjectInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	obj, ok := s.objects[bucket+"/"+key]
	if !ok {
		return ObjectInfo{}, fmt.Errorf("object %s/%s not found", bucket, key)
	}
	return ObjectInfo{Size: int64(len(obj.data)), ContentType: obj.contentType}, nil
}

func (s *MemoryStore) Get(_ context.Context, bucket, key string) (io.ReadCloser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	obj, ok := s.objects[bucket+"/"+key]
	if !ok {
		return nil, fmt.Errorf("object %s/%s not found", bucket, key)
	}
	return io.NopCloser(bytes.NewReader(obj.data)), nil
}

func (s *MemoryStore) Put(_ context.Context, bucket, key string, r io.Reader, _ int64, contentType string) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading object body: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[bucket+"/"+key] = memObject{data: data, contentType: contentType}
	return nil
}

func (s *MemoryStore) Remove(_ context.Context, bucket, key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.objects, bucket+"/"+key)
	return nil
}
