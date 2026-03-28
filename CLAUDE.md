# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`cn-system-load-testing` is a cloud native system load testing tool in Go. The main binary is `cng-bench`.

Module: `github.com/BernhardDenner/cn-system-load-testing`

## Commands

```bash
make build       # build bin/cng-bench
make test        # run all Ginkgo test suites (randomized, with race detector)
make lint        # run golangci-lint
make clean       # remove bin/
make bootstrap   # install ginkgo CLI into $GOPATH/bin
```

Run a single package's tests:
```bash
go test ./pkg/bench/...
$(go env GOPATH)/bin/ginkgo ./pkg/bench/
```

## Architecture

```
cmd/cng-bench/      CLI entry point (cobra)
pkg/bench/          Core benchmark framework
  bench.go          Scenario interface, Runner, Config, Result, Report types
```

### Core abstractions (`pkg/bench`)

- **`Scenario`** — interface every load test scenario implements (`Name() string`, `Run(ctx) Result`)
- **`Runner`** — executes a `Scenario` under concurrent load according to a `Config`
- **`Config`** — tuning knobs: `Concurrency`, `Duration`, `RampUp`
- **`Result`** — outcome of one scenario invocation (`Duration`, `Err`)
- **`Report`** — aggregated output: throughput, P50/P95/P99 latencies, error count

New scenarios implement the `Scenario` interface and are wired into the cobra CLI in `cmd/cng-bench/`.

## Testing

Tests use [Ginkgo v2](https://onsi.github.io/ginkgo/) + [Gomega](https://onsi.github.io/gomega/). Each package has a `suite_test.go` that bootstraps the suite via `RunSpecs`, and `*_test.go` files with `Describe`/`It` blocks. Use the `bench_test` external test package pattern.
