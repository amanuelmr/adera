// Package ratelimit provides in-process keyed token-bucket rate limiting.
//
// Limits are per-process; behind multiple replicas the effective limit scales
// with replica count. The Limiter interface is the swap point for a shared
// store (e.g., Redis) when the service scales out.
package ratelimit

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limiter answers whether one event for key is allowed right now.
type Limiter interface {
	Allow(key string) bool
}

// Keyed maintains one token bucket per key with idle-entry eviction.
type Keyed struct {
	mu       sync.Mutex
	buckets  map[string]*entry
	rate     rate.Limit
	burst    int
	lastSeen func() time.Time // injectable clock for tests
	stop     chan struct{}
}

type entry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewKeyed creates a limiter allowing r events/second with the given burst.
func NewKeyed(r rate.Limit, burst int) *Keyed {
	k := &Keyed{
		buckets:  make(map[string]*entry),
		rate:     r,
		burst:    burst,
		lastSeen: time.Now,
		stop:     make(chan struct{}),
	}
	go k.janitor()
	return k
}

// PerMinute is a convenience constructor for "n events per minute".
func PerMinute(n int) *Keyed {
	return NewKeyed(rate.Limit(float64(n)/60.0), n)
}

// Allow reports whether one event for key may proceed.
func (k *Keyed) Allow(key string) bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	e, ok := k.buckets[key]
	if !ok {
		e = &entry{limiter: rate.NewLimiter(k.rate, k.burst)}
		k.buckets[key] = e
	}
	e.lastSeen = k.lastSeen()
	return e.limiter.Allow()
}

// Close stops the janitor goroutine.
func (k *Keyed) Close() { close(k.stop) }

func (k *Keyed) janitor() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-k.stop:
			return
		case <-ticker.C:
			cutoff := k.lastSeen().Add(-10 * time.Minute)
			k.mu.Lock()
			for key, e := range k.buckets {
				if e.lastSeen.Before(cutoff) {
					delete(k.buckets, key)
				}
			}
			k.mu.Unlock()
		}
	}
}

// Unlimited is a no-op limiter for tests and disabled configurations.
type Unlimited struct{}

func (Unlimited) Allow(string) bool { return true }
