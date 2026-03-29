# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`cn-system-load-testing` is a cloud native system load testing tool in Go. The main binary is `csl-bench`.

Module: `github.com/BernhardDenner/cn-system-load-testing`

## Commands

```bash
make build       # build bin/csl-bench
make test        # run all Ginkgo test suites (randomized, with race detector)
make lint        # run golangci-lint
make clean       # remove bin/
make bootstrap   # install ginkgo CLI into $GOPATH/bin
```

Run a single package's tests:
```bash
go test ./pkg/cpu/...
$(go env GOPATH)/bin/ginkgo ./pkg/cpu/
```

## Architecture

```
cmd/csl-bench/      CLI (cobra)
  main.go           entry point
  root.go           root command + all persistent flags
  benchmark.go      benchmark subcommand + run loop
  baseline.go       baseline subcommand (rate-limited mode)
  prometheus.go     custom Prometheus collector + /metrics HTTP server
  version.go        version variable (injected via ldflags)
pkg/bench/          Core framework
  bench.go          Scenario interface, Result
  metrics.go        Metrics struct (JSON output), PhaseRunning/PhaseSummary constants
  throttle.go       ThrottledScenario — rate-limiting wrapper for baseline mode
pkg/cpu/            CPU load module
  cpu.go            Scenario implementation (LockOSThread per goroutine)
  pi.go             ComputePi — Chudnovsky algorithm with binary splitting
pkg/diskio/         Disk IO load module
  diskio.go         Scenario with three IO modes, lazy file init, io.Closer
pkg/memory/         Memory load module
  memory.go         Ring buffer of 1 MB blocks, allocate/write/read/evict per Run()
  sysinfo_linux.go  cgroup v2/v1 then sysinfo memory detection
  sysinfo_other.go  1 GB fallback for non-Linux
deploy/examples/    Kubernetes deployment manifests
```

### Core abstractions (`pkg/bench`)

- **`Scenario`** — interface every module implements: `Name() string` and `Run(ctx) Result`
- **`Result`** — outcome of one `Run` call: `Duration` and `Err`
- **`Metrics`** — JSON record printed each interval and as a final summary; fields: `timestamp`, `phase`, `module`, `ops`, `ops_per_sec`, `avg_latency_ms`, `errors`, `bytes_read`, `bytes_written`, `bytes_read_per_sec`, `bytes_written_per_sec`
- **`ThrottledScenario`** — wraps any `Scenario` to enforce read/write bytes-per-second rate limits; used by the baseline command for disk IO throttling

### CLI flags (`cmd/csl-bench`)

Global flags (available to both `benchmark` and `baseline`):

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--duration` | `-d` | 300 | seconds to run; 0 = infinite |
| `--interval` | `-i` | 1 | seconds between metric reports |
| `--module` | `-m` | — | module to run; repeatable |
| `--metrics_port` | — | 9090 | Prometheus /metrics port; 0 to disable |
| `--cpu_num_threads` | — | 1 | threads for the cpu module |
| `--memory_max_use_mb` | — | 0 | max memory in MB; 0 = auto-detect |
| `--io_mode` | — | `randomized_rw` | disk IO pattern (`txn_rw`, `sequential_rw`, `randomized_rw`) |
| `--io_file_path` | — | `/tmp/bench-data` | data file path |
| `--io_batch_size_kb` | — | 4 | read/write batch size in KB |
| `--io_file_size_mb` | — | 1024 | data file size in MB |

Baseline-only flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--io_read_bps` | 0 | max read bytes/sec for disk; 0 = unlimited |
| `--io_write_bps` | 0 | max write bytes/sec for disk; 0 = unlimited |

### Build & Docker

```bash
make image       # docker build with VERSION tag
make push        # push to registry
```

Version is injected via `-ldflags -X main.version=...` (see `Makefile` and `Dockerfile`).

### Adding a new module

1. Create `pkg/<name>/` with a type implementing `bench.Scenario` (and optionally `io.Closer`)
2. Add a `case "<name>"` to `buildScenarios()` in `cmd/csl-bench/benchmark.go`
3. Add module flags to `root.go` and read them via `moduleParams` in `benchmark.go`

### CPU module (`pkg/cpu`)

Each `Run()` call spawns `Config.Threads` goroutines, locks each to its own OS thread (`runtime.LockOSThread`), computes π to `Config.PiDigits` decimal places using the Chudnovsky algorithm (binary splitting, `math/big`), then waits for all threads. The benchmark loop in `runLoop` calls `Run()` repeatedly for the full duration and aggregates throughput/latency metrics atomically.

### Disk IO module (`pkg/diskio`)

Each `Run()` performs one write+read cycle according to the configured `Mode`:
- **`txn_rw`** — write random batch → `fsync` → read a *different* random batch (simulates transactional DB IO)
- **`sequential_rw`** — write and read the next batch in sequence, wrapping at EOF
- **`randomized_rw`** — write and read at independent random offsets

The data file is lazily created and pre-populated with random data (1 MB chunks) on the first `Run()` call. The scenario implements `io.Closer`; the benchmark loop calls `Close()` after the run finishes.

## Testing

Tests use [Ginkgo v2](https://onsi.github.io/ginkgo/) + [Gomega](https://onsi.github.io/gomega/). Each package has a `suite_test.go` bootstrapping `RunSpecs` and `*_test.go` files using `Describe`/`It` blocks in the external `<pkg>_test` package.
