package loadbalancer

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/dimxxkun/zenith-gateway/pkg/circuit"
)

var ErrNoHealthyUpstream = errors.New("no healthy upstream servers available")

// Upstream represents a downstream backend service target.
type Upstream struct {
	ID      string
	URL     string
	Weight  int
	Alive   bool
	Breaker *circuit.CircuitBreaker
}

// WeightedRoundRobin distributes load according to upstream capacity.
type WeightedRoundRobin struct {
	mu        sync.RWMutex
	upstreams []*Upstream
	counter   uint64
}

func NewWeightedRoundRobin(upstreams []*Upstream) *WeightedRoundRobin {
	return &WeightedRoundRobin{
		upstreams: upstreams,
	}
}

// Next selects the next available, healthy upstream server.
func (w *WeightedRoundRobin) Next() (*Upstream, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var candidates []*Upstream
	for _, u := range w.upstreams {
		if u.Alive && u.Breaker.State() != circuit.StateOpen {
			for i := 0; i < u.Weight; i++ {
				candidates = append(candidates, u)
			}
		}
	}

	if len(candidates) == 0 {
		return nil, ErrNoHealthyUpstream
	}

	idx := atomic.AddUint64(&w.counter, 1) % uint64(len(candidates))
	return candidates[idx], nil
}

func (w *WeightedRoundRobin) SetHealth(upstreamID string, alive bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, u := range w.upstreams {
		if u.ID == upstreamID {
			u.Alive = alive
			break
		}
	}
}
