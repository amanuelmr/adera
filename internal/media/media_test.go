package media

import (
	"bytes"
	"image"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
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
