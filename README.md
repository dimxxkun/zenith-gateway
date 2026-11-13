# Zenith-Gateway

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Zenith-Gateway** is an asynchronous API reverse proxy, traffic shaping, and resilience engine written in Go. It incorporates dynamic Token Bucket rate limiting, Circuit Breaker state automata, and Weighted Round-Robin load balancing to safeguard microservices against cascading failures and traffic spikes.

---

## Architectural Flow

```
 Client Requests
       │
       ▼
[ Keyed Token Bucket ] ──(Exceeds Burst?)──> 429 Too Many Requests
       │
       ▼ (Permitted)
[ Weighted Round-Robin ] ──(Filters Out Open Circuits)
       │
       ├───> Upstream Alpha (Healthy: Closed Breaker)
       ├───> Upstream Beta  (Faulty: Tripped Open -> Bypass)
       └───> Upstream Gamma (Canary: Half-Open Probe)
```

## Core Modules

1. **`pkg/ratelimit`**: Token Bucket algorithm tracking burst capacity and fractional refill rates per client key with thread safety.
2. **`pkg/circuit`**: State machine (`Closed` $\rightarrow$ `Open` $\rightarrow$ `HalfOpen` $\rightarrow$ `Closed`) tracking consecutive failures, cooldown periods, and canary probe executions.
3. **`pkg/loadbalancer`**: Weighted Round-Robin distributor dynamically checking health status and circuit health before dispatching traffic.

## Reproducible Environment & Verifiers

The test harness in `pkg/gateway_test.go` verifies:
- Strict rejection of requests once token capacity is exhausted.
- Automatic transition to `StateOpen` upon reaching failure threshold.
- Failover rerouting bypassing degraded backends without dropping user traffic.

```bash
# Run deterministic test verifiers
go test -v ./...

# Run simulated gateway traffic demo
go run cmd/gateway/main.go
```

## License
MIT © [dimxxkun](https://github.com/dimxxkun)
