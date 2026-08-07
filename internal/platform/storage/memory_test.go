package storage

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryStorePutGetRoundTrip(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	require.NoError(t, s.Put(ctx, "bucket", "key", strings.NewReader("hello"), 5, "text/plain"))

	rc, err := s.Get(ctx, "bucket", "key")
	require.NoError(t, err)
	defer rc.Close()
	data, err := io.ReadAll(rc)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

func TestMemoryStoreGetMissingObject(t *testing.T) {
	s := NewMemory()
	_, err := s.Get(context.Background(), "bucket", "missing")
	assert.Error(t, err)
}

func TestMemoryStoreSameKeyDifferentBucketsDoNotCollide(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	require.NoError(t, s.Put(ctx, "bucket-a", "key", strings.NewReader("a"), 1, "text/plain"))
	require.NoError(t, s.Put(ctx, "bucket-b", "key", strings.NewReader("b"), 1, "text/plain"))

	rcA, err := s.Get(ctx, "bucket-a", "key")
	require.NoError(t, err)
	defer rcA.Close()
	dataA, _ := io.ReadAll(rcA)
	assert.Equal(t, "a", string(dataA))

	rcB, err := s.Get(ctx, "bucket-b", "key")
	require.NoError(t, err)
	defer rcB.Close()
	dataB, _ := io.ReadAll(rcB)
	assert.Equal(t, "b", string(dataB))
}

func TestMemoryStoreStatReflectsSizeAndContentType(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	require.NoError(t, s.Put(ctx, "bucket", "key", strings.NewReader("hello"), 5, "image/png"))

	info, err := s.Stat(ctx, "bucket", "key")
	require.NoError(t, err)
	assert.EqualValues(t, 5, info.Size)
	assert.Equal(t, "image/png", info.ContentType)
}

func TestMemoryStoreStatMissingObject(t *testing.T) {
	s := NewMemory()
	_, err := s.Stat(context.Background(), "bucket", "missing")
	assert.Error(t, err)
}

func TestMemoryStoreRemove(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	require.NoError(t, s.Put(ctx, "bucket", "key", strings.NewReader("hello"), 5, "text/plain"))
	require.NoError(t, s.Remove(ctx, "bucket", "key"))

	_, err := s.Get(ctx, "bucket", "key")
	assert.Error(t, err, "object must be gone after Remove")
}

func TestMemoryStoreRemoveMissingObjectIsNoop(t *testing.T) {
	s := NewMemory()
	assert.NoError(t, s.Remove(context.Background(), "bucket", "missing"))
}

func TestMemoryStorePresignGetRequiresExistingObject(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()

	_, err := s.PresignGet(ctx, "bucket", "missing", 0)
	assert.Error(t, err)

	require.NoError(t, s.Put(ctx, "bucket", "key", strings.NewReader("hello"), 5, "text/plain"))
	url, err := s.PresignGet(ctx, "bucket", "key", 0)
	require.NoError(t, err)
	assert.Contains(t, url, "bucket/key")
}
