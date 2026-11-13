package circuit

import (
	"errors"
	"sync"
	"time"
)

type State int

const (
	StateClosed State = iota
	StateHalfOpen
	StateOpen
)

var (
	ErrCircuitOpen = errors.New("circuit breaker is open: request denied")
)

type BreakerConfig struct {
	FailureThreshold uint32        // Consecutive failures before opening
	SuccessThreshold uint32        // Consecutive successes in half-open before closing
	CooldownPeriod   time.Duration // Time to wait in open state before canary probe
}

func DefaultConfig() BreakerConfig {
	return BreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		CooldownPeriod:   100 * time.Millisecond,
	}
}

// CircuitBreaker guards remote calls against cascading failures.
type CircuitBreaker struct {
	mu           sync.RWMutex
	state        State
	failures     uint32
	successes    uint32
	lastStateChange time.Time
	config       BreakerConfig
}

func NewCircuitBreaker(cfg BreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		state:           StateClosed,
		config:          cfg,
		lastStateChange: time.Now(),
	}
}

func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Execute runs the provided callback if circuit is healthy or ready for probe.
func (cb *CircuitBreaker) Execute(action func() error) error {
	if !cb.canExecute() {
		return ErrCircuitOpen
	}

	err := action()
	cb.recordResult(err)
	return err
}

func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	if cb.state == StateOpen {
		if now.Sub(cb.lastStateChange) >= cb.config.CooldownPeriod {
			// Transition to Half-Open for canary probe
			cb.state = StateHalfOpen
			cb.failures = 0
			cb.successes = 0
			cb.lastStateChange = now
			return true
		}
		return false
	}
	return true
}

func (cb *CircuitBreaker) recordResult(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	if err != nil {
		cb.failures++
		cb.successes = 0

		if cb.state == StateHalfOpen || cb.failures >= cb.config.FailureThreshold {
			cb.state = StateOpen
			cb.lastStateChange = now
		}
	} else {
		cb.successes++
		cb.failures = 0

		if cb.state == StateHalfOpen && cb.successes >= cb.config.SuccessThreshold {
			cb.state = StateClosed
			cb.lastStateChange = now
		}
	}
}
