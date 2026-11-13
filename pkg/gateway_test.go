package gateway_test

import (
	"errors"
	"testing"
	"time"

	"github.com/dimxxkun/zenith-gateway/pkg/circuit"
	"github.com/dimxxkun/zenith-gateway/pkg/loadbalancer"
	"github.com/dimxxkun/zenith-gateway/pkg/ratelimit"
)

// Verifier 1: TokenBucket respects capacity limits and rejects excess calls
func TestTokenBucketRateLimiting(t *testing.T) {
	limiter := ratelimit.NewTokenBucket(5, 10) // 5 burst, refills 10 tokens/sec

	// First 5 must succeed
	for i := 0; i < 5; i++ {
		if !limiter.Allow() {
			t.Fatalf("expected token %d to be allowed", i+1)
		}
	}

	// 6th must be rejected immediately
	if limiter.Allow() {
		t.Fatal("expected request 6 to be rate limited")
	}

	// Wait 250ms -> should refill ~2.5 tokens
	time.Sleep(250 * time.Millisecond)
	if !limiter.Allow() {
		t.Fatal("expected token to be allowed after refill")
	}
}

// Verifier 2: CircuitBreaker transitions Closed -> Open -> HalfOpen -> Closed
func TestCircuitBreakerStateTransitions(t *testing.T) {
	cfg := circuit.BreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		CooldownPeriod:   50 * time.Millisecond,
	}
	cb := circuit.NewCircuitBreaker(cfg)

	failErr := errors.New("upstream 503")

	// 1st failure
	_ = cb.Execute(func() error { return failErr })
	if cb.State() != circuit.StateClosed {
		t.Fatal("expected circuit to stay closed after 1 failure")
	}

	// 2nd failure -> Trips open
	_ = cb.Execute(func() error { return failErr })
	if cb.State() != circuit.StateOpen {
		t.Fatal("expected circuit to trip open after 2 failures")
	}

	// Calls while open should be blocked immediately with ErrCircuitOpen
	err := cb.Execute(func() error { return nil })
	if err != circuit.ErrCircuitOpen {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}

	// Wait for cooldown
	time.Sleep(60 * time.Millisecond)

	// Next call should probe in HalfOpen state
	err = cb.Execute(func() error { return nil })
	if err != nil || cb.State() != circuit.StateHalfOpen {
		t.Fatalf("expected canary probe in half-open state, state=%v", cb.State())
	}

	// Second success should fully close circuit
	err = cb.Execute(func() error { return nil })
	if err != nil || cb.State() != circuit.StateClosed {
		t.Fatalf("expected circuit to recover and close, state=%v", cb.State())
	}
}

// Verifier 3: Weighted Round-Robin bypasses tripped upstreams and routes correctly
func TestWeightedRoundRobinWithBreakers(t *testing.T) {
	cb1 := circuit.NewCircuitBreaker(circuit.DefaultConfig())
	cb2 := circuit.NewCircuitBreaker(circuit.DefaultConfig())

	u1 := &loadbalancer.Upstream{ID: "srv-1", Weight: 1, Alive: true, Breaker: cb1}
	u2 := &loadbalancer.Upstream{ID: "srv-2", Weight: 2, Alive: true, Breaker: cb2}

	lb := loadbalancer.NewWeightedRoundRobin([]*loadbalancer.Upstream{u1, u2})

	// Both healthy: selections should be distributed
	picks := make(map[string]int)
	for i := 0; i < 30; i++ {
		u, err := lb.Next()
		if err != nil {
			t.Fatalf("unexpected balancing error: %v", err)
		}
		picks[u.ID]++
	}

	if picks["srv-2"] <= picks["srv-1"] {
		t.Fatalf("expected srv-2 (weight 2) to get more traffic than srv-1 (weight 1): %+v", picks)
	}

	// Fail srv-2 to trip its breaker
	for i := 0; i < 3; i++ {
		_ = cb2.Execute(func() error { return errors.New("upstream failed") })
	}

	// Now all traffic must seamlessly divert to srv-1
	for i := 0; i < 10; i++ {
		u, err := lb.Next()
		if err != nil {
			t.Fatalf("unexpected failover error: %v", err)
		}
		if u.ID != "srv-1" {
			t.Fatalf("expected only srv-1, got %s", u.ID)
		}
	}
}
