package ratings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfidenceThresholds(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "none"},
		{MinSampleForTrend - 1, "none"},
		{MinSampleForTrend, "low"},
		{9, "low"},
		{10, "medium"},
		{29, "medium"},
		{30, "high"},
		{1000, "high"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, confidence(tt.n), "confidence(%d)", tt.n)
	}
}

func TestRound2(t *testing.T) {
	tests := []struct {
		in   float64
		want float64
	}{
		{4.005, 4.01}, // half-away-from-zero, not banker's rounding
		{4.004, 4.0},
		{3.335, 3.34},
		{0, 0},
		{-3.335, -3.34},
	}
	for _, tt := range tests {
		assert.InDelta(t, tt.want, round2(tt.in), 1e-9, "round2(%v)", tt.in)
	}
}
