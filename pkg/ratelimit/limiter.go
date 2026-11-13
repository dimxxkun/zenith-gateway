package ratelimit

import (
	"sync"
	"time"
)

// TokenBucket implements a thread-safe token bucket algorithm for API rate limiting.
type TokenBucket struct {
	mu           sync.Mutex
	capacity     float64
	tokens       float64
	refillRate   float64 // tokens per second
	lastRefill   time.Time
}

// NewTokenBucket creates a limiter with maximum burst capacity and refill rate.
func NewTokenBucket(capacity float64, refillRate float64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow attempts to consume 1 token. Returns true if permitted, false if rate limited.
func (tb *TokenBucket) Allow() bool {
	return tb.AllowN(1)
}

// AllowN attempts to consume n tokens atomically.
func (tb *TokenBucket) AllowN(n float64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.lastRefill = now

	// Refill tokens based on elapsed duration
	tb.tokens = tb.tokens + (elapsed * tb.refillRate)
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}

	if tb.tokens >= n {
		tb.tokens -= n
		return true
	}
	return false
}

// KeyedRateLimiter manages token buckets per client IP or API token.
type KeyedRateLimiter struct {
	mu         sync.RWMutex
	buckets    map[string]*TokenBucket
	capacity   float64
	refillRate float64
}

func NewKeyedRateLimiter(capacity float64, refillRate float64) *KeyedRateLimiter {
	return &KeyedRateLimiter{
		buckets:    make(map[string]*TokenBucket),
		capacity:   capacity,
		refillRate: refillRate,
	}
}

func (k *KeyedRateLimiter) Allow(key string) bool {
	k.mu.RLock()
	bucket, exists := k.buckets[key]
	k.mu.RUnlock()

	if !exists {
		k.mu.Lock()
		bucket, exists = k.buckets[key]
		if !exists {
			bucket = NewTokenBucket(k.capacity, k.refillRate)
			k.buckets[key] = bucket
		}
		k.mu.Unlock()
	}

	return bucket.Allow()
}
