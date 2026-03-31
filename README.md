# cn-system-load-testing

A cloud native system load testing tool in Golang

# Description

This is a system load testing tool designed to run in Kubernetes clusters to
assess the cluster's system performance. It performs CPU, memory, and disk IO
benchmarks in a distributed manner by running the benchmark tool in individual
pods across the whole Kubernetes cluster. During the benchmark, performance
metrics are measured and written to standard out, which allows easy analysis
via common log collection and analysis tools like OpenSearch.

The benchmark tool can be run in two main modes:

* **`benchmark`**: load tests run without restrictions at full capacity. Designed
  to find the maximum performance characteristics of the cluster and help
  optimise and fine-tune the underlying system setup.
* **`baseline`**: load tests run within defined usage limits. Reports whether
  configured performance thresholds are met, making it suitable for continuous
  health checks and detecting CPU starvation or storage degradation.


## `csl-bench`

This is the main tool to run performance tests.

### Usage

```
csl-bench <benchmark|baseline> -m <module> [flags]
```

### Byte Size Values

Flags that specify an amount of bytes accept a numeric value with an optional
unit suffix (case-insensitive):

| Suffix | Unit |
|--------|------|
| `b` | bytes |
| `k` or `kb` | kibibytes (1 024 bytes) |
| `m` or `mb` | mebibytes (1 024 KiB) |
| `g` or `gb` | gibibytes (1 024 MiB) |

A plain number without a suffix is interpreted as bytes.

Examples: `4kb`, `512mb`, `2gb`, `1073741824`

### Common Flags

These flags are available to both `benchmark` and `baseline`:

| Flag | Default | Description |
|------|---------|-------------|
| `-d, --duration` | `0` | seconds to run; `0` = run until cancelled with Ctrl-C |
| `-i, --interval` | `1` | seconds between metric reports |
| `-m, --module` | — | module to run (`cpu`, `memory`, `disk`); may be repeated |
| `--metrics_port` | `9090` | port for the Prometheus `/metrics` endpoint; `0` to disable |
| `--cpu_num_threads` | `1` | number of threads for the CPU module |
| `--memory_max_use` | `0` | max memory to allocate, e.g. `512mb`, `2gb`; `0` = auto-detect |
| `--io_mode` | `randomized_rw` | disk IO pattern: `txn_rw`, `sequential_rw`, `randomized_rw` |
| `--io_file_path` | `/tmp/bench-data` | path to the data file for IO operations |
| `--io_batch_size` | `4kb` | read/write batch size, e.g. `4kb`, `1mb` |
| `--io_file_size` | `1gb` | maximum data file size, e.g. `512mb`, `2gb` |

### Modules

#### CPU

Computes π to a fixed number of decimal places (Chudnovsky algorithm) in one
or more OS-thread-locked goroutines. Generates sustained, predictable CPU load.

#### Memory

Maintains a ring buffer of 1 MiB blocks. Each operation allocates a new block
(write stress), stores it in the ring evicting the oldest when full, and reads
a random existing block — exercising allocation, GC, and memory bandwidth.

#### Disk IO

Performs write+read cycles on a pre-allocated data file. Three IO patterns are
available:

* **`txn_rw`** — write a batch, `fsync`, then read a different random batch.
  Simulates transactional database IO.
* **`sequential_rw`** — write and read batches in sequence through the file.
* **`randomized_rw`** — write and read batches at independent random offsets.

The data file is created and pre-populated on the first run and deleted on exit.

### Baseline Mode Flags

In addition to all common flags, the `baseline` subcommand accepts:

#### Disk IO rate limiting

| Flag | Default | Description |
|------|---------|-------------|
| `--io_read_bps` | `0` | max read bytes/sec, e.g. `50mb`; `0` = unlimited |
| `--io_write_bps` | `0` | max write bytes/sec, e.g. `50mb`; `0` = unlimited |

When a limit is set, the scenario throttles to that rate and `baseline_met`
reports whether actual throughput reaches at least 98% of the target.

#### CPU load control

Exactly one of the following may be specified (they are mutually exclusive):

| Flag | Default | Description |
|------|---------|-------------|
| `--cpu_load_factor` | `0` | duty cycle 0.0–1.0; `0.5` = active 50% of the time |
| `--cpu_ops_per_sec` | `0` | target π calculations per second; `baseline_met=0` if actual rate drops below 98% of target |

`--cpu_load_factor` is for controlled background load generation and does not
produce a `baseline_met` result. `--cpu_ops_per_sec` enforces a target rate and
reports `baseline_met=0` when the system cannot sustain it — indicating CPU
starvation or degradation.

#### Memory load control

| Flag | Default | Description |
|------|---------|-------------|
| `--memory_load_factor` | `0` | fraction of available memory to allocate, e.g. `0.5` for 50%; `0` = use `--memory_max_use` or auto-detect |

### Performance Metrics Output

At each reporting interval and at the end of the run, `csl-bench` prints one
JSON record per active module to stdout.

| Field | Description |
|-------|-------------|
| `timestamp` | ISO 8601 timestamp with millisecond precision |
| `mode` | `benchmark` or `baseline` |
| `phase` | `running` during the run; `summary` at the end |
| `module` | module name: `cpu`, `memory`, `disk` |
| `ops` | cumulative operation count |
| `ops_per_sec` | operations per second during the last interval |
| `avg_latency_ms` | average operation latency in milliseconds during the last interval |
| `errors` | cumulative error count |
| `bytes_read` | cumulative bytes read (disk and memory modules) |
| `bytes_written` | cumulative bytes written (disk and memory modules) |
| `bytes_read_per_sec` | read throughput during the last interval |
| `bytes_written_per_sec` | write throughput during the last interval |
| `baseline_met` | `1` if all configured thresholds are met (≥98%), `0` otherwise; omitted when no threshold is configured or in benchmark mode |
| `target_read_bps` | configured read bytes/sec threshold; omitted when not set |
| `target_write_bps` | configured write bytes/sec threshold; omitted when not set |
| `target_ops_per_sec` | configured ops/sec threshold; omitted when not set |

Example output (baseline mode, disk module):

```json
{"timestamp":"2026-03-31T08:00:00.000Z","mode":"baseline","phase":"running","module":"disk","ops":256,"ops_per_sec":256.0,"avg_latency_ms":0.018,"errors":0,"bytes_read":1048576,"bytes_written":1048576,"bytes_read_per_sec":1048576.0,"bytes_written_per_sec":1048576.0,"baseline_met":1}
```

### Prometheus Metrics

When `--metrics_port` is non-zero (default 9090), `csl-bench` exposes a
Prometheus-compatible `/metrics` endpoint. All metrics carry `module` and `mode`
labels.

| Metric | Type | Description |
|--------|------|-------------|
| `csl_bench_ops_total` | counter | total operations performed |
| `csl_bench_errors_total` | counter | total failed operations |
| `csl_bench_latency_seconds_total` | counter | cumulative operation latency |
| `csl_bench_bytes_read_total` | counter | total bytes read |
| `csl_bench_bytes_written_total` | counter | total bytes written |
| `csl_bench_baseline_met` | gauge | `1` if thresholds are met, `0` otherwise |
| `csl_bench_target_read_bps` | gauge | configured read bytes/sec threshold; only present when set |
| `csl_bench_target_write_bps` | gauge | configured write bytes/sec threshold; only present when set |
| `csl_bench_target_ops_per_sec` | gauge | configured ops/sec threshold; only present when set |

A `PodMonitor` resource for the Prometheus Operator is provided in
`deploy/examples/pod-monitor.yaml`.
