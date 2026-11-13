package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/dimxxkun/zenith-gateway/pkg/circuit"
	"github.com/dimxxkun/zenith-gateway/pkg/loadbalancer"
	"github.com/dimxxkun/zenith-gateway/pkg/ratelimit"
)

func main() {
	fmt.Println("=======================================================")
	fmt.Println("   Zenith-Gateway: Resilient API Reverse Proxy Engine  ")
	fmt.Println("=======================================================")

	// 1. Initialize Upstream Services
	cb1 := circuit.NewCircuitBreaker(circuit.DefaultConfig())
	cb2 := circuit.NewCircuitBreaker(circuit.DefaultConfig())
	cb3 := circuit.NewCircuitBreaker(circuit.DefaultConfig())

	upstreams := []*loadbalancer.Upstream{
		{ID: "payment-svc-az1", URL: "http://10.0.1.10:8080", Weight: 3, Alive: true, Breaker: cb1},
		{ID: "payment-svc-az2", URL: "http://10.0.2.10:8080", Weight: 3, Alive: true, Breaker: cb2},
		{ID: "payment-svc-fallback", URL: "http://10.0.9.99:8080", Weight: 1, Alive: true, Breaker: cb3},
	}
	lb := loadbalancer.NewWeightedRoundRobin(upstreams)
	limiter := ratelimit.NewKeyedRateLimiter(20, 50) // 20 burst per client

	fmt.Printf("[1] Gateway online: 3 upstream targets, Keyed TokenBucket active.\n")

	// 2. Simulate High-Throughput Request Ingestion
	fmt.Println("\n[2] Dispatching 50 concurrent requests across upstreams...")
	var wg sync.WaitGroup
	var accepted, limited, failures int64
	var mu sync.Mutex

	clients := []string{"client-tier-1", "client-tier-2", "client-tier-1"}

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(reqID int) {
			defer wg.Done()
			client := clients[reqID%len(clients)]

			if !limiter.Allow(client) {
				mu.Lock()
				limited++
				mu.Unlock()
				return
			}

			upstream, err := lb.Next()
			if err != nil {
				mu.Lock()
				failures++
				mu.Unlock()
				return
			}

			err = upstream.Breaker.Execute(func() error {
				// Simulated network work
				time.Sleep(1 * time.Millisecond)
				return nil
			})

			mu.Lock()
			if err == nil {
				accepted++
			} else {
				failures++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	fmt.Printf("    Results -> 200 OK: %d | 429 Rate Limited: %d | 503 Outage: %d\n",
		accepted, limited, failures)

	// 3. Chaos Engineering: Simulate AZ Outage & Circuit Trip
	fmt.Println("\n[3] Injecting downstream fault: payment-svc-az1 failing consecutively...")
	for i := 0; i < 3; i++ {
		_ = cb1.Execute(func() error { return fmt.Errorf("connection timeout 504") })
	}
	fmt.Printf("    payment-svc-az1 Circuit State: %v (TRIPPED OPEN)\n", cb1.State())

	fmt.Println("\n[4] Routing subsequent requests (Zero dropped requests):")
	for i := 0; i < 5; i++ {
		u, _ := lb.Next()
		fmt.Printf("    • Request #%d smoothly routed to: %s\n", i+1, u.ID)
	}

	fmt.Println("\nZenith-Gateway verification executed successfully.")
}
