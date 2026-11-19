# Zenith-Gateway

[![License: Noncommercial](https://img.shields.io/badge/License-Noncommercial-blue.svg)](LICENSE)

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
[Noncommercial license](LICENSE) © [dimxxkun](https://github.com/dimxxkun)

## Commercial permission

Original material in this repository is available under the [Noncommercial Use and Commercial Permission License](LICENSE). **Noncommercial use is free. Commercial use requires prior written permission and a separate written agreement specifying the scope and agreed fee.** Contact [work.abdulfatai@gmail.com](mailto:work.abdulfatai@gmail.com) with the repository name and intended use.

Portfolio review and recruitment evaluation are permitted without a fee. Third-party components retain their own licenses. Permissions already granted for earlier versions remain valid; the new terms do not revoke those grants. These are custom source-available terms, not an OSI-approved open-source license.
