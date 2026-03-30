# cn-system-load-testing

A cloud native system load testing tool in Golang

# Description

This is a system load testing tool designed to run in Kubernetes clusters to
assess the clusters system performance. It performs CPU, memory, network and
disk IO benchmarks in a distributed manner. This is done by running a the
benchmark tool in individual pods across the whole Kubernetes cluster.
During the benchmark, performance metrics are measured and written to standard
out, which allows easy analysis via common log collection and analysis tools
like OpenSearch. The benchmark tool can be run in two main modes: `benchmark`
and `baseline`. In benchmark mode, the load tests aren't restricted an run
with full capacity (within their set limits), while in baseline mode the load
test will stay within defined usage limits and reports if defined performance
thresholds are met. The benchmark is designed to find the maximum performance
characteristics of the overall Kubernetes cluster and can help to optimize
and finetune the underlaying system setup. The baseline mode is design to
assess if a given cluster meets certain performance criteria and helps to
ensure a given system is running properly.


## `csl-bench`

This is the main tool to run performance tests and to orchestrate them.

### Usage

`csl-bench` provides an intuitive commandline interface to control the
runtime behavior.

#### Byte Size Values

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

#### Common Config Options

These options are valid for all sub commands

* `-d, --duration`: number of seconds to run the benchmark. Use 0 to run until
  cancelled with Ctrl-C (default: 0)
* `-i, --interval`: number of seconds to report performance metrics (default: 1)
* `-m, --module`: run the given test module. Valid values are `cpu`, `memory`,
  `disk`, `network`. Can be specified multiple times to run multiple modules at the
  same time.

#### Sub-Commands

The tool provides the following sub-commands:

* `benchmark`: run in benchmark mode
* `baseline`: run in baseline mode

### `csl-bench` Modules

The benchmark tool is highly configurable and consists of the following modules:

#### CPU

This module will perform various calculations to generate CPU load.

Configuration options:
* `cpu_num_threads`: number of threads to use during load testing (default: 1)

#### Memory

This module performs heap memory allocations and deallocations as well as
read/write operations on the allocated memory.

Configuration options:
* `memory_max_use`: maximum memory to use for the benchmark, e.g. `512mb`, `2gb`
  (default: maximum available memory for the machine or container)

#### Disk IO

This module performs different disk IO operations to stress the storage system
while simulating different kinds of usage pattern.

Configuration options:
* `io_mode`: one of (default: `randomized_rw`)
  * `txn_rw`: transactional read/write. A small batch of data (`io_batch_size`)
    is written to disk and fsynced. Afterwards the different (previously written)
    data batch from a random location is read again. Simulates transactional database
    IO behaviour.
  * `sequential_rw`: Sequential read/write. A series of small batches of data
    is written and read in sequence from the disk.
  * `randomized_rw`: Randomized read/write. Small batches of data is written and read
    from disk in randomized order.
* `io_file_path`: This is the path to the data file, which is used to perform all
   IO operations (default: `/tmp/bench-data`)
* `io_batch_size`: size of data batches to read/write at once, e.g. `4kb`, `1mb` (default: `4kb`)
* `io_file_size`: maximum data file size, e.g. `512mb`, `2gb` (default: `1gb`)

#### Network IO

To be defined.

### Performance Metrics Output

During the benchmark run, `csl-bench` will print the latest recoreded performance
metrics in JSON format. This can be later used in log collection tools to parse
and aggregate the results.

Fields:
* `timestamp`: current time
* `phase`: `running` if the benchmark is still running, `summary` if the benchmark
   has reached its desired run time.
* ...  (tbd: module specific fields)


## `csl-cli`

To be defined.