package media

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"image/png"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adera-platform/backend/internal/platform/web"
)

func TestSniffImageType(t *testing.T) {
	var jpegBuf bytes.Buffer
	err := jpeg.Encode(&jpegBuf, image.NewRGBA(image.Rect(0, 0, 4, 4)), nil)
	assert.NoError(t, err)
	var pngBuf bytes.Buffer
	err = png.Encode(&pngBuf, image.NewRGBA(image.Rect(0, 0, 4, 4)))
	assert.NoError(t, err)

	cases := []struct {
		name string
		data []byte
		want string
	}{
		{"jpeg", jpegBuf.Bytes(), "image/jpeg"},
		{"png", pngBuf.Bytes(), "image/png"},
		{"webp", append([]byte("RIFF\x00\x00\x00\x00WEBP"), 0), "image/webp"},
		{"empty", nil, ""},
		{"text", []byte("hello world, definitely not an image"), ""},
		{"html", []byte("<html><script>alert(1)</script>"), ""},
		{"truncated riff", []byte("RIFF"), ""},
		{"riff not webp", []byte("RIFF\x00\x00\x00\x00WAVE"), ""},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, SniffImageType(tc.data), tc.name)
	}
}

// FuzzSniffImageType ensures the sniffer never panics on arbitrary input and
// only ever returns one of the known types.
func FuzzSniffImageType(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0xFF, 0xD8, 0xFF})
	f.Add([]byte("RIFF1234WEBP"))
	f.Add([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A})
	f.Fuzz(func(t *testing.T, data []byte) {
		got := SniffImageType(data)
		switch got {
		case "", "image/jpeg", "image/png", "image/webp":
		default:
			t.Fatalf("unexpected type %q", got)
		}
	})
}

func TestDimensionsOK(t *testing.T) {
	assert.True(t, dimensionsOK(image.Config{Width: 1600, Height: 1200}))
	assert.True(t, dimensionsOK(image.Config{Width: 6000, Height: 4000}), "exactly 24 MP")
	assert.False(t, dimensionsOK(image.Config{Width: 6000, Height: 4001}))
	assert.False(t, dimensionsOK(image.Config{Width: 0, Height: 100}))
	assert.False(t, dimensionsOK(image.Config{Width: 1 << 20, Height: 1 << 20}), "no int overflow")
}

func TestDecodeSlotsBoundConcurrency(t *testing.T) {
	s := NewService(nil, nil, nil, "private", "public")
	var releases []func()
	for range decodeConcurrency {
		release, err := s.acquireDecode(context.Background())
		require.NoError(t, err)
		releases = append(releases, release)
	}

	// Every slot is taken: the next caller waits until its request ends.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	_, err := s.acquireDecode(ctx)
	var werr *web.Error
	require.ErrorAs(t, err, &werr)
	assert.Equal(t, 503, werr.Status)

	// A released slot is available again.
	releases[0]()
	release, err := s.acquireDecode(context.Background())
	require.NoError(t, err)
	release()
	for _, r := range releases[1:] {
		r()
	}
}
