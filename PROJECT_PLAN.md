# 1. Project Vision and Scope

This document is the living plan and progress tracker for building a Redis-like in-memory key-value datastore in Go. It is intentionally detailed and numbered to support planning, reviews, and incremental delivery. Track progress by checking the boxes.

## 1.1 Goals

- [x] Deliver a production-grade, Redis-inspired KV store with strong observability, operability, and reliability.
- [x] Start with a minimal core (GET/SET/DEL, TTL) and iterate towards persistence, replication, sentinel-style failover, clustering/sharding, pub/sub, and more.
- [x] Provide first-class metrics, logs, and tracing; ship Docker-based local and demo deployments with dashboards.

## 1.2 Non-goals (initially)

- [ ] Perfect protocol compatibility with all Redis commands (we will scope and prioritize subsets).
- [ ] Modules ecosystem. We may expose hooks later.
- [ ] Full RESP3 immediately; start with RESP2 subset and evolve.

## 1.3 Guiding Principles

- [x] Correctness before micro-optimizations; measurable performance targets.
- [x] Simple concurrency model; predictable latencies; avoid unnecessary shared-state contention.
- [ ] Incremental durability and HA; graceful degradation where possible.
- [x] Observability by default: metrics, logs, traces in every milestone.

# 2. High-level Architecture

## 2.1 Process Model and Concurrency

- [x] Single-writer-per-shard model: keys are partitioned by consistent hash into N shards, each served by a dedicated goroutine/event loop (minimizes locks, predictable tail latency).
- [x] Network IO: per-connection goroutine with parsing, dispatch to shard mailboxes; backpressure via bounded channels.
- [x] Pipelining support from day one; batching where possible.

## 2.2 Protocol

- [x] RESP2 encoder/decoder to enable `redis-cli` compatibility for core subset (PING, ECHO, GET, SET, DEL, EXPIRE, TTL, INFO, AUTH, etc.).
- [ ] Later: selective RESP3 features (optional).

## 2.3 Storage

- [x] In-memory store per shard: `map[string]ValueRecord` with metadata (type, ttlExpireAt, version, size, flags).
- [x] TTL index per shard: hierarchical timing wheel or min-heap + lazy expiration on access.
- [ ] Eviction policies: none initially; later LRU/LFU approximations with sampling.

## 2.4 Persistence

- [ ] Snapshots (RDB-like): periodic background save to a compact binary format; CRC and versioning.
- [ ] Append-Only Log (AOF): sequential write-ahead log with periodic fsync, rewrite/compaction, crash-safety guarantees.
- [ ] Startup logic: fast snapshot restore + optional AOF tail replay.

## 2.5 Replication and HA

- [ ] Leader-follower asynchronous replication (full sync + partial resync via replication backlog).
- [ ] Read replicas; optional read-your-writes on leader-only.
- [ ] Sentinel-style supervisor: health checks, quorum, leader election, automatic failover, client redirection.

## 2.6 Cluster and Sharding

- [ ] Key hashing via 16384 slots or consistent hashing ring with virtual nodes.
- [ ] Slot-based routing, rebalancing, resharding with minimal downtime.
- [ ] Gossip metadata propagation and redirection (MOVED/ASK equivalents).

## 2.7 Observability

- [ ] Structured logging (zerolog/zap) with request IDs and sampling.
- [ ] Metrics (Prometheus): RPS, cmd latency (p50/p90/p95/p99), errors, memory, CPU, GC, connections, replication lag, backlog size.
- [ ] Tracing (OpenTelemetry): spans for accept → parse → execute → persist → replicate; Jaeger/Tempo compatible.

# 3. Milestones and Detailed Tasks

## 3.1 Phase 0 — Repository and Tooling Foundation

### 3.1.1 Scaffolding

- [x] Initialize Go module `github.com/Abhishek2095/kv-stash` (Go 1.22+).
- [x] Directory layout:
  - [x] `cmd/kvstash` (server main)
  - [x] `internal/proto` (RESP)
  - [x] `internal/server` (net, sessions, dispatcher)
  - [x] `internal/store` (shards, engine, ttl)
  - [ ] `internal/persist` (rdb, aof)
  - [ ] `internal/replica` (replication)
  - [ ] `internal/sentinel` (failover control plane)
  - [ ] `internal/cluster` (slots, routing)
  - [x] `internal/obs` (metrics, logs, tracing)
  - [ ] `pkg/client` (Go client; CLI)
  - [x] `configs/`, `deploy/`, `docs/`, `scripts/`

### 3.1.2 Dev Experience

- [x] Makefile: build, lint, test, bench, run, docker, compose, clean.
- [x] `golangci-lint` configured (vet, staticcheck, errcheck, gofumpt).
- [x] Unit test harness and seed example tests (`PING`/`ECHO`).
- [ ] Pre-commit hooks (fmt, lint, unit tests).

### 3.1.3 CI/CD

- [x] GitHub Actions: lint + unit tests on PR; build artifacts.
- [ ] Integration matrix job using docker-compose (single-node smoke tests).
- [ ] Release workflow with `goreleaser` (static binaries, Docker images).

## 3.2 Phase 1 — Minimal Server and Protocol

### 3.2.1 Networking

- [x] TCP server with graceful shutdown; configuration (port, workers, shard count).
- [x] Connection handling, read/write timeouts, max pipeline depth.
- [ ] Simple DoS protection: command rate limiter per connection.

### 3.2.2 RESP2 Codec

- [x] Parser with pooled buffers to minimize allocations.
- [x] Encoder with pre-sized byte builders.
- [x] Support inline and bulk commands; error handling and protocol tests.

### 3.2.3 Boot Commands

- [x] `PING`, `ECHO`, `INFO`, `COMMAND`, `QUIT`.
- [ ] `AUTH` with a single shared password (configurable).
- [x] Unit tests and golden test vectors.

## 3.3 Phase 2 — Core KV and TTL

### 3.3.1 Basic Commands

- [x] `GET`, `SET`, `DEL`, `EXISTS`, `MGET`, `MSET`.
- [x] Status codes and integer replies per RESP.

### 3.3.2 SET Variants and TTL

- [x] `SET` options: `NX|XX`, `EX|PX`, `KEEPTTL`, `GET`.
- [x] `EXPIRE`, `PEXPIRE`, `TTL`, `PTTL`, `PERSIST`.
- [x] TTL storage: lazy expiration + periodic scan; timing wheel/min-heap experiment gate.

### 3.3.3 Numeric Ops

- [x] `INCR`, `DECR`, `INCRBY`, `DECRBY`, `APPEND`, `GETRANGE`, `SETRANGE`.

### 3.3.4 Introspection

- [x] `DBSIZE`, `KEYS` (warn: heavy), `SCAN` (cursor-based), `TYPE`.

## 3.4 Phase 3 — Observability Baseline

### 3.4.1 Logging

- [ ] Zerolog/Zap integration; request-scoped fields; sampling for hot paths.
- [ ] Log levels via config; structured errors.

### 3.4.2 Metrics (Prometheus)

- [x] Command RPS, latency histograms (include p95/p99), failures per command.
- [x] Process metrics (CPU, RSS), Go runtime (GC, goroutines), connections.
- [x] Store metrics: keys, expired keys rate, TTL heap size, shard queue depth.

### 3.4.3 Tracing (OpenTelemetry)

- [ ] Spans: accept → parse → route → execute → persist → replicate.
- [ ] Exporters: OTLP → Jaeger/Tempo; trace sampling policies.

## 3.5 Phase 4 — Persistence

### 3.5.1 Snapshots (RDB-like)

- [ ] Binary format with header (magic, version), CRC, record entries (key, type, ttl, value).
- [ ] Background save without pausing writers (copy-on-write per shard or versioned iterators).
- [ ] Startup restore; integrity checks.

### 3.5.2 Append-Only Log (AOF)

- [ ] Command logging with durable write policies: `everysec`, `always`, `no`.
- [ ] Buffered writer with fsync; crash-safety test (kill -9 during load).
- [ ] Rewrite/compaction process; atomic swap; AOF + snapshot hybrid.

## 3.6 Phase 5 — Replication and Read Replicas

### 3.6.1 Protocol

- [ ] Full sync: snapshot transfer + command stream tailing.
- [ ] Partial resync (PSYNC-like) with rolling backlog per leader.

### 3.6.2 Topology and Reads

- [ ] Follower replication offset tracking; lag metrics; reconnection logic.
- [ ] Read-only replicas; leader-only writes; optional `WAIT`-style durability.

## 3.7 Phase 6 — Sentinel-style HA

### 3.7.1 Sentinel Agent

- [ ] Health-check leaders/followers; collect replication offsets; quorum config.
- [ ] Leader election and failover orchestration; client redirection info.
- [ ] Notification hooks (webhooks/logs); fencing tokens to avoid split-brain.

### 3.7.2 Client Discovery

- [ ] Simple service registry (static config/DNS in dev); sentinel resolver for CLI/clients.

## 3.8 Phase 7 — Cluster and Sharding

### 3.8.1 Keyspace Partitioning

- [ ] 16384 slot map (Redis-like) or consistent-hash ring with virtual nodes (configurable).
- [ ] MOVED/ASK equivalents and client-side redirection hints.

### 3.8.2 Reshard and Rebalance

- [ ] Live slot migration with dual-writes or handoff logs; bounded inconsistency.
- [ ] Gossip and config epoch/versioning; converge under churn.

## 3.9 Phase 8 — Pub/Sub

- [ ] Channels and pattern subscriptions; fanout within node and across cluster.
- [ ] Backpressure-aware delivery; slow consumer handling; metrics and tests.

## 3.10 Phase 9 — Advanced Capabilities (Optional, Staged)

### 3.10.1 Transactions and Watch

- [ ] `MULTI`/`EXEC` with optimistic concurrency (`WATCH` on key versions).
- [ ] Rollback semantics for command queues pre-`EXEC`.

### 3.10.2 Scripting

- [ ] Deterministic embedded scripting (Lua or Starlark) with timeouts and sandboxing.

### 3.10.3 Data Structures

- [ ] Hashes, Lists, Sets, Sorted Sets (skiplist) — scoped, incremental.
- [ ] Bitmaps/HyperLogLog (optional later); approximate structures.

### 3.10.4 Memory and Eviction

- [ ] Maxmemory with policies: `noeviction`, `allkeys-lru`, `volatile-lru`, `allkeys-lfu` (approx sampling).
- [ ] Key frequency sketch; aging; eviction statistics.

### 3.10.5 Security

- [ ] TLS (server and client); mTLS optional; certificate reload.
- [ ] ACLs: users, keys pattern, command categories.

# 4. Operations and Delivery

## 4.1 Configuration

- [ ] YAML/TOML config file with env overrides; dynamic config for logging and sampling.
- [ ] Hot-reload via SIGHUP or admin command channel.

## 4.2 Docker and Local Deployments

- [x] Multi-stage Dockerfile (small static image).
- [x] `docker-compose` for: single node, leader+replicas+sentinel, cluster (3+ nodes), Prometheus, Grafana, Jaeger.
- [ ] Pre-baked Grafana dashboards and Prometheus scrape config.

## 4.3 Packaging and Release

- [ ] `goreleaser` for Linux/macOS/Windows; SBOM/attestations; changelogs.
- [ ] Versioning strategy and upgrade notes (snapshot/AOF compatibility).

# 5. Observability: Implementation Details

## 5.1 Logging

- [ ] Correlate logs with trace IDs; redact secrets; structured fields (`cmd`, `key`, `latency_ms`).

## 5.2 Metrics

- [ ] Histograms per command and per shard; exemplars with trace links.
- [ ] Export p95/p99 latencies and RPS; replication lag; backlog; eviction rate; memory by type.

## 5.3 Tracing

- [ ] Parent/child spans across network, shard dispatch, persistence and replication.
- [ ] Trace attributes: `db.system=kvstash`, `net.peer.ip`, `cmd.name`, `key.hashslot`.

# 6. Testing Strategy

## 6.1 Unit Tests

- [ ] RESP parser/encoder tests with golden vectors.
- [ ] Store semantics tests: TTL accuracy, lazy vs active expiration correctness, boundary conditions.
- [ ] PING/PONG, GET/SET/DEL, SET with options; error cases.

## 6.2 Property and Fuzz Testing

- [ ] Property-based tests for idempotence, invariants (e.g., TTL never negative on read).
- [ ] Go fuzzers for parser and command handlers.

## 6.3 Integration Tests

- [ ] Black-box tests via `redis-cli` speaking RESP2 to our server for subset coverage.
- [ ] Persistence crash tests (kill -9 during load), recovery assertions.
- [ ] Replication consistency tests; split-brain simulations with sentinel.

## 6.4 Performance and Soak

- [ ] Benchmarks for pipelines, mixed read/write, TTL churn.
- [ ] Tail-latency tracking (p95/p99) under load; GC regression detection.
- [ ] Long-running soak with memory leak detection.

# 7. Performance Targets (initial)

- [ ] Single-node: ≥ 100k ops/sec on localhost for small GET/SET with p95 < 3ms, p99 < 10ms.
- [ ] Replication lag p95 < 50ms on LAN.
- [ ] Snapshot time proportional to dataset size with bounded pause (< 5ms per shard).

# 8. CLI and Client Tooling

- [ ] Minimal `kvstash-cli` compatible with RESP2; supports `--cluster` and sentinel discovery.
- [ ] Admin-only commands for introspection: `STATS`, `CONFIG GET/SET`, `SLOWLOG`, `LATENCY LATEST`.

# 9. Security and Compliance

- [ ] AUTH secret management via env files or secret stores.
- [ ] TLS termination; optional client cert verification.
- [ ] Basic ACLs: command categories and key patterns.

# 10. Risk Register and Mitigations

- [ ] GC pauses: mitigate by pooling, reduced allocations, shard sizing.
- [ ] AOF fsync stalls: mitigate with `everysec` default and write coalescing.
- [ ] Replica divergence on failover: fencing tokens, epochs, and idempotent handoff.
- [ ] Slot migration complexity: staged migration with handoff log and client redirection.

# 11. Definition of Done (per feature)

- [ ] Tests: unit + integration; coverage added.
- [ ] Observability: metrics, logs, and at least one trace span.
- [ ] Docs: README snippet and config examples updated.
- [ ] Bench/latency snapshot recorded in `docs/perf/`.

# 12. Roadmap with Suggested Ordering

1. Phase 0: repo + tooling
   1.1 Makefile, lint, tests, CI
   1.2 Dockerfile, compose skeleton
2. Phase 1: server + RESP + PING/ECHO/AUTH
   2.1 Per-connection handling + dispatch
   2.2 Parser/encoder + tests
3. Phase 2: core KV + TTL
   3.1 GET/SET/DEL/EXISTS; SET options; EXPIRE/TTL
   3.2 Introspection (DBSIZE/TYPE/SCAN)
4. Phase 3: observability baseline (metrics/logs/tracing)
5. Phase 4: snapshots + AOF
6. Phase 5: replication + read replicas
7. Phase 6: sentinel (failover)
8. Phase 7: clustering/sharding
9. Phase 8: pub/sub
10. Phase 9+: advanced features (transactions, scripting, more data types, eviction policies, security, RESP3)

# 13. Example Configs and Commands

## 13.1 Example Server Config (YAML)

```yaml
server:
  listen_addr: ":6380"
  shards: 8
  auth_password: "change-me"
limits:
  max_clients: 10000
  max_pipeline: 1024
storage:
  maxmemory_bytes: 0  # 0 = unlimited
  eviction_policy: "noeviction"
ttl:
  strategy: "lazy+active"
  active_cycle_ms: 50
persistence:
  snapshot:
    enabled: true
    interval_seconds: 300
    dir: "./data"
  aof:
    enabled: true
    fsync: "everysec"  # always|everysec|no
    dir: "./data"
replication:
  role: "leader"  # leader|follower
  leader_addr: ""
observability:
  log_level: "info"
  prometheus_listen: ":9100"
  otlp_endpoint: "http://otel-collector:4317"
```

## 13.2 Docker Compose (sketch)

```yaml
version: "3.9"
services:
  kv-leader:
    image: ghcr.io/abhishek2095/kv-stash:latest
    command: ["--config", "/etc/kvstash/leader.yaml"]
    ports: ["6380:6380", "9100:9100"]
    volumes: ["./deploy/configs:/etc/kvstash"]
  kv-replica:
    image: ghcr.io/abhishek2095/kv-stash:latest
    command: ["--config", "/etc/kvstash/replica.yaml"]
    depends_on: [kv-leader]
  sentinel:
    image: ghcr.io/abhishek2095/kv-stash-sentinel:latest
    ports: ["26379:26379"]
  prometheus:
    image: prom/prometheus
    volumes: ["./deploy/prometheus.yml:/etc/prometheus/prometheus.yml"]
    ports: ["9090:9090"]
  grafana:
    image: grafana/grafana
    ports: ["3000:3000"]
  jaeger:
    image: jaegertracing/all-in-one
    ports: ["16686:16686"]
```

# 14. Backlog (to triage)

- [ ] RESP3 upgrade path; client-side redirection improvements.
- [ ] Streams (append-only log data type) and consumer groups.
- [ ] Geospatial indices; probabilistic structures via addons.
- [ ] Pluggable storage engines for experimentation.
- [ ] Helm charts for K8s.

# 15. Acceptance for MVP

- [ ] Single node with GET/SET/DEL/EXPIRE/TTL; PING/ECHO/AUTH.
- [ ] Basic observability: Prometheus metrics, structured logs, minimal tracing.
- [ ] Docker image + compose example; basic README and quickstart.
- [ ] Unit + integration tests passing in CI.

# 16. Quick Start Tasks (Week 1)

- [ ] Repo bootstrap (module, layout, Makefile, lint, CI).
- [ ] RESP2 codec and `PING`/`ECHO`.
- [ ] Minimal store with `GET`/`SET`/`DEL`.
- [ ] TTL support and `EXPIRE`/`TTL`.
- [ ] Metrics endpoints and initial dashboards.

---

This plan is intentionally ambitious; we will iterate pragmatically and adjust scope based on measurement and feedback. Every milestone must land with tests and observability.
