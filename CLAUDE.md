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
  baseline.go       baseline subcommand (stub)
pkg/bench/          Core framework
  bench.go          Scenario interface, Runner, Config, Result, Report
  metrics.go        Metrics struct (JSON output), PhaseRunning/PhaseSummary constants
pkg/cpu/            CPU load module
  cpu.go            Scenario implementation (LockOSThread per goroutine)
  pi.go             ComputePi — Chudnovsky algorithm with binary splitting
```

### Core abstractions (`pkg/bench`)

- **`Scenario`** — interface every module implements: `Name() string` and `Run(ctx) Result`
- **`Result`** — outcome of one `Run` call: `Duration` and `Err`
- **`Metrics`** — JSON record printed each interval and as a final summary; fields: `timestamp`, `phase`, `module`, `ops`, `ops_per_sec`, `avg_latency_ms`, `errors`
- **`Runner`** / `Config` / `Report` — reserved for future single-run / reporting use; currently stub

### CLI flags (`cmd/csl-bench`)

All flags are persistent (available to both `benchmark` and `baseline`):

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--duration` | `-d` | 300 | seconds to run; 0 = infinite |
| `--interval` | `-i` | 1 | seconds between metric reports |
| `--module` | `-m` | — | module to run; repeatable |
| `--cpu_num_threads` | — | 1 | threads for the cpu module |

### Adding a new module

1. Create `pkg/<name>/` with a type implementing `bench.Scenario`
2. Add a `case "<name>"` to `buildScenarios()` in `cmd/csl-bench/benchmark.go`
3. Add any module flags to `root.go`

### CPU module (`pkg/cpu`)

Each `Run()` call spawns `Config.Threads` goroutines, locks each to its own OS thread (`runtime.LockOSThread`), computes π to `Config.PiDigits` decimal places using the Chudnovsky algorithm (binary splitting, `math/big`), then waits for all threads. The benchmark loop in `runLoop` calls `Run()` repeatedly for the full duration and aggregates throughput/latency metrics atomically.

## Testing

Tests use [Ginkgo v2](https://onsi.github.io/ginkgo/) + [Gomega](https://onsi.github.io/gomega/). Each package has a `suite_test.go` bootstrapping `RunSpecs` and `*_test.go` files using `Describe`/`It` blocks in the external `<pkg>_test` package.
