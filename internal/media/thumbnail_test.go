package media

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// solid builds an image filled with one colour.
func solid(w, h int, c color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestThumbnailNeverUpscales(t *testing.T) {
	small := solid(100, 80, color.White)

	got := thumbnail(small, ThumbMaxEdge)

	// Returned unchanged, so the caller can skip storing a derivative.
	assert.Same(t, small, got)
	assert.Equal(t, 100, got.Bounds().Dx())
}

func TestThumbnailFitsTheLongestEdge(t *testing.T) {
	tests := []struct {
		name         string
		w, h         int
		wantW, wantH int
	}{
		{"landscape", 1600, 1200, 480, 360},
		{"portrait", 1200, 1600, 360, 480},
		{"square", 1000, 1000, 480, 480},
		{"very wide keeps at least one row", 4000, 4, 480, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := thumbnail(solid(tt.w, tt.h, color.White), ThumbMaxEdge)

			assert.Equal(t, tt.wantW, got.Bounds().Dx())
			assert.Equal(t, tt.wantH, got.Bounds().Dy())
			assert.LessOrEqual(t, got.Bounds().Dx(), ThumbMaxEdge)
			assert.LessOrEqual(t, got.Bounds().Dy(), ThumbMaxEdge)
		})
	}
}

func TestThumbnailPreservesColour(t *testing.T) {
	red := color.RGBA{R: 220, G: 20, B: 30, A: 255}

	got := thumbnail(solid(1200, 900, red), ThumbMaxEdge)

	// Averaging a uniform image must return that same colour.
	r, g, b, a := got.At(10, 10).RGBA()
	assert.InDelta(t, 220, r>>8, 1)
	assert.InDelta(t, 20, g>>8, 1)
	assert.InDelta(t, 30, b>>8, 1)
	assert.EqualValues(t, 65535, a)
}

func TestThumbnailAveragesRatherThanPicksOnePixel(t *testing.T) {
	// A one-pixel checkerboard downscaled hard must go grey. A
	// nearest-neighbour resize would return pure black or pure white.
	src := image.NewRGBA(image.Rect(0, 0, 1000, 1000))
	for y := range 1000 {
		for x := range 1000 {
			if (x+y)%2 == 0 {
				src.Set(x, y, color.White)
			} else {
				src.Set(x, y, color.Black)
			}
		}
	}

	got := thumbnail(src, 100)

	r, _, _, _ := got.At(50, 50).RGBA()
	assert.InDelta(t, 32768, r, 6000, "a checkerboard should average towards mid grey")
}

func TestSampleStepBoundsWork(t *testing.T) {
	// Small boxes are read in full; large ones are capped, so cost tracks the
	// thumbnail size and not the source resolution.
	assert.Equal(t, 1, sampleStep(1))
	assert.Equal(t, 1, sampleStep(maxSamplesPerAxis))
	assert.Equal(t, 2, sampleStep(maxSamplesPerAxis*2))
	assert.Equal(t, 25, sampleStep(100))
	for _, extent := range []int{5, 17, 100, 4096} {
		samples := (extent + sampleStep(extent) - 1) / sampleStep(extent)
		assert.LessOrEqual(t, samples, maxSamplesPerAxis, "extent %d", extent)
	}
}

func TestEncodeImage(t *testing.T) {
	img := solid(20, 20, color.White)

	t.Run("jpeg round-trips", func(t *testing.T) {
		raw, err := encodeImage(img, "jpg", thumbJPEGQuality)
		require.NoError(t, err)
		_, err = jpeg.Decode(bytes.NewReader(raw))
		assert.NoError(t, err)
	})

	t.Run("png round-trips", func(t *testing.T) {
		raw, err := encodeImage(img, "png", 0)
		require.NoError(t, err)
		_, err = png.Decode(bytes.NewReader(raw))
		assert.NoError(t, err)
	})

	t.Run("rejects an unknown format", func(t *testing.T) {
		_, err := encodeImage(img, "webp", 80)
		assert.Error(t, err)
	})
}

func TestThumbnailIsSmallerOnTheWire(t *testing.T) {
	// The whole point is bytes saved on a metered connection.
	large := solid(2000, 1500, color.RGBA{R: 90, G: 140, B: 200, A: 255})
	full, err := encodeImage(large, "jpg", fullJPEGQuality)
	require.NoError(t, err)

	thumb, err := encodeImage(thumbnail(large, ThumbMaxEdge), "jpg", thumbJPEGQuality)
	require.NoError(t, err)

	assert.Less(t, len(thumb), len(full))
}
