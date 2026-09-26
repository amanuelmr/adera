package media

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
)

// ThumbMaxEdge is the longest edge of a generated thumbnail, in pixels. A
// review list on a 360pt-wide phone shows photos in a slot well under half
// that, so 480 covers a 2-3x display without shipping the full image.
const ThumbMaxEdge = 480

// maxSamplesPerAxis bounds how many source pixels are averaged along each
// axis for one output pixel. Without it a 40 MP upload would read 40 million
// pixels on a user-facing request; with it the cost tracks the thumbnail size
// instead. For photographic downscales a 4x4 sample grid is visually
// indistinguishable from averaging every pixel in the box.
const maxSamplesPerAxis = 4

// thumbnail downscales img so its longest edge is at most maxEdge, by
// averaging over the source box for each output pixel. It never upscales: an
// image already within the bound is returned unchanged.
func thumbnail(src image.Image, maxEdge int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 || maxEdge <= 0 {
		return src
	}
	longest := w
	if h > longest {
		longest = h
	}
	if longest <= maxEdge {
		return src
	}

	dw, dh := max(1, w*maxEdge/longest), max(1, h*maxEdge/longest)
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))

	for dy := range dh {
		y0, y1 := b.Min.Y+dy*h/dh, b.Min.Y+(dy+1)*h/dh
		if y1 <= y0 {
			y1 = y0 + 1
		}
		yStep := sampleStep(y1 - y0)
		for dx := range dw {
			x0, x1 := b.Min.X+dx*w/dw, b.Min.X+(dx+1)*w/dw
			if x1 <= x0 {
				x1 = x0 + 1
			}
			xStep := sampleStep(x1 - x0)

			// RGBA() returns alpha-premultiplied values, which are the
			// correct thing to average.
			var rs, gs, bs, as, n uint64
			for y := y0; y < y1; y += yStep {
				for x := x0; x < x1; x += xStep {
					r, g, bl, a := src.At(x, y).RGBA()
					rs += uint64(r)
					gs += uint64(g)
					bs += uint64(bl)
					as += uint64(a)
					n++
				}
			}
			if n == 0 {
				n = 1
			}
			dst.SetRGBA64(dx, dy, color.RGBA64{
				R: uint16(rs / n), G: uint16(gs / n), B: uint16(bs / n), A: uint16(as / n),
			})
		}
	}
	return dst
}

// sampleStep returns the stride that keeps samples along one axis within
// maxSamplesPerAxis.
func sampleStep(extent int) int {
	if extent <= maxSamplesPerAxis {
		return 1
	}
	return (extent + maxSamplesPerAxis - 1) / maxSamplesPerAxis
}

// encodeImage renders img in the given sanitized format ("jpg" or "png").
func encodeImage(img image.Image, format string, jpegQuality int) ([]byte, error) {
	var buf bytes.Buffer
	switch format {
	case "jpg":
		if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
			return nil, fmt.Errorf("encoding jpeg: %w", err)
		}
	case "png":
		if err := png.Encode(&buf, img); err != nil {
			return nil, fmt.Errorf("encoding png: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported image format %q", format)
	}
	return buf.Bytes(), nil
}
