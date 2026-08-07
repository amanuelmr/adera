package ratelimit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestKeyedAllowRespectsBurst(t *testing.T) {
	k := NewKeyed(rate.Limit(0.001), 2) // refill is negligible within the test
	defer k.Close()

	assert.True(t, k.Allow("user-1"))
	assert.True(t, k.Allow("user-1"))
	assert.False(t, k.Allow("user-1"), "burst exhausted")
}

func TestKeyedAllowIsPerKey(t *testing.T) {
	k := NewKeyed(rate.Limit(0.001), 1)
	defer k.Close()

	assert.True(t, k.Allow("user-1"))
	assert.False(t, k.Allow("user-1"))
	assert.True(t, k.Allow("user-2"), "a different key must have its own bucket")
}

func TestPerMinuteConfiguresRateAndBurst(t *testing.T) {
	k := PerMinute(5)
	defer k.Close()

	assert.Equal(t, 5, k.burst)
	assert.InDelta(t, float64(5)/60.0, float64(k.rate), 1e-9)
}

func TestUnlimitedAlwaysAllows(t *testing.T) {
	u := Unlimited{}
	assert.True(t, u.Allow("anything"))
	assert.True(t, u.Allow(""))
}
