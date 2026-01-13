# Caliper

Precision benchmarking to find optimal CI runner sizes.

A command-line tool that measures build performance across different CPU/RAM configurations, helping you right-size your CI runners.

## Features

- Run any shell command multiple times and measure execution time
- **Warm-up run** by default to eliminate cold-start effects (caches, JIT, filesystem)
- Handles failures gracefully and continues benchmarking
- Calculates comprehensive statistics: mean, median, standard deviation, min, max, P90, P95
- Outputs results in multiple formats: console, JSON, CSV, and Markdown
- Tracks success rate and provides detailed error reporting
- **Matrix mode**: Run benchmarks across multiple CPU/RAM configurations in Docker containers

## Installation

### Build from source

```bash
go build -o caliper
```

### Install globally (optional)

```bash
go install
```

## Usage

### Basic Command

```bash
./caliper --runs 10 --command "cargo clean && cargo build"
```

### Shorthand Flags

```bash
./caliper -n 10 -c "cargo clean && cargo build"
```

### With Custom Output Directory and Name

```bash
./caliper -n 5 -c "npm test" --output-dir ./results --name npm-test-benchmark
```

## Command-Line Options

| Flag | Shorthand | Required | Description |
|------|-----------|----------|-------------|
| `--runs` | `-n` | Yes | Number of times to run the benchmark |
| `--command` | `-c` | Yes | Command to benchmark (supports shell features like `&&`, `||`, pipes) |
| `--output-dir` | | No | Directory to save output files (default: current directory) |
| `--name` | | No | Benchmark name for reports (default: timestamp) |
| `--no-warmup` | | No | Skip the warm-up run (default: warm-up enabled) |

## Output Files

The tool generates four types of output:

1. **Console Output**: Real-time progress and formatted summary table
2. **JSON** (`{name}.json`): Machine-readable results with full metadata
3. **CSV** (`{name}.csv`): Spreadsheet-compatible format with individual runs and statistics
4. **Markdown** (`{name}.md`): Human-readable report with tables

## Examples

### Benchmarking Cargo Build

```bash
./caliper -n 10 -c "cargo clean && cargo build"
```

### Benchmarking with Release Mode

```bash
./caliper -n 5 -c "cargo clean && cargo build --release" --name cargo-release
```

### Benchmarking Tests

```bash
./caliper -n 20 -c "npm run test" --name npm-tests --output-dir ./benchmark-results
```

### Complex Shell Commands

```bash
./caliper -n 3 -c "docker-compose down && docker-compose up -d && npm test && docker-compose down"
```

### Skip Warm-up Run

```bash
./caliper -n 10 -c "cargo build" --no-warmup
```

## Matrix Mode

Matrix mode allows you to benchmark across multiple CPU/RAM configurations using Docker containers. Each configuration runs sequentially to avoid resource contention.

### Basic Matrix Command

```bash
./caliper matrix \
  --image ubuntu-2404-go-rust \
  --repo https://github.com/influxdata/influxdb \
  --runs 10 \
  --command "cargo clean && cargo build" \
  --configs "2:8,4:16,8:32,16:64,32:128"
```

### Matrix Command-Line Options

| Flag | Shorthand | Required | Description |
|------|-----------|----------|-------------|
| `--image` | | Yes | Docker image to use |
| `--repo` | | Yes | Git repository URL to clone |
| `--command` | `-c` | Yes | Command to benchmark |
| `--configs` | | Yes | CPU:RAM configurations (e.g., `2:8,4:16,8:32`) |
| `--runs` | `-n` | No | Number of runs per configuration (default: 10) |
| `--output-dir` | | No | Directory for output files (default: `./matrix-results`) |
| `--name` | | No | Benchmark name (default: timestamp) |
| `--no-warmup` | | No | Skip the warm-up run |

### How Matrix Mode Works

For each CPU/RAM configuration, the tool:

1. **Starts a Docker container** with resource limits (`--cpus`, `--cpuset-cpus`, `--memory`, `--memory-swap`)
2. **Clones the repository** inside the container
3. **Runs the benchmark** using the same warm-up + measured runs approach
4. **Copies results** to the host
5. **Stops and removes the container**
6. **Proceeds to the next configuration**

Configurations run **sequentially** to ensure accurate measurements without resource contention.

### Matrix Output Structure

```
matrix-results/
├── 2cpu_8gb/
│   ├── benchmark_2cpu_8gb.json
│   ├── benchmark_2cpu_8gb.csv
│   └── benchmark_2cpu_8gb.md
├── 4cpu_16gb/
│   └── ...
├── 8cpu_32gb/
│   └── ...
├── matrix_summary.json
├── matrix_summary.csv
└── matrix_summary.md
```

### Matrix Summary Table

The tool outputs a comparison table:

```
Matrix Benchmark Summary
========================

Image:      ubuntu-2404-go-rust
Repository: https://github.com/influxdata/influxdb
Command:    cargo clean && cargo build
Runs:       10 per configuration

CPUs  RAM      Mean     Median   Std Dev  Min      Max      Success
----  ---      ----     ------   -------  ---      ---      -------
2     8 GB     5m23s    5m18s    12.3s    5m10s    5m45s    100%
4     16 GB    3m12s    3m08s    8.1s     3m02s    3m25s    100%
8     32 GB    2m01s    1m58s    5.2s     1m52s    2m10s    100%
16    64 GB    1m15s    1m12s    3.8s     1m08s    1m22s    100%
32    128 GB   58s      56s      2.1s     54s      1m02s    100%
```

## Output Format

### Console Output

```
Caliper
=======
Command: cargo clean && cargo build
Runs: 10 (+ 1 warm-up)
Output Directory: .

Starting benchmark...

Warm-up: ✓ Completed in 47.1s (excluded from stats)

Run 1/10: ✓ Completed in 45.2s
Run 2/10: ✓ Completed in 43.8s
...

Benchmark Results
=================

Command:        cargo clean && cargo build
Total Runs:     10
Warm-up:        47.1s (excluded from stats)
Successful:     10
Failed:         0
Success Rate:   100.0%
Total Duration: 7m30s

Statistics (successful runs only)
---------------------------------

Metric   Value
------   -----
N        10
Mean     45s (45.123s)
Median   44s (44.892s)
Std Dev  2s (1.845s)
Min      43s (43.123s)
Max      48s (48.456s)
P90      47s (47.234s)
P95      48s (48.012s)
```

### JSON Structure

```json
{
  "config": {
    "command": "cargo clean && cargo build",
    "runs": 10,
    "name": "benchmark_20250113_120000",
    "outputDir": "."
  },
  "summary": {
    "totalRuns": 10,
    "successful": 10,
    "failed": 0,
    "successRate": 100,
    "startTime": "2025-01-13T12:00:00Z",
    "endTime": "2025-01-13T12:07:30Z",
    "totalDuration": 450.123
  },
  "statistics": {
    "n": 10,
    "mean": 45.123,
    "median": 44.892,
    "stdDev": 1.845,
    "min": 43.123,
    "max": 48.456,
    "p90": 47.234,
    "p95": 48.012
  },
  "runs": [...]
}
```

## Warm-up Run

By default, the tool executes a **warm-up run** before the measured benchmark runs. This eliminates cold-start effects that can skew results:

- CPU/filesystem caches
- JIT compilation
- Dynamic linker caching
- Docker layer caching

The warm-up run is:
- **Excluded from statistics** - only measured runs count
- **Recorded in output files** - for transparency (JSON `warmupRun` field, CSV `warmup` row, Markdown report)
- **Required to succeed** - if warm-up fails, the benchmark aborts

Use `--no-warmup` to disable this behavior if you specifically want to measure cold-start performance.

## Exit Codes

- `0`: All runs completed successfully (100% success rate)
- `1`: One or more runs failed, warm-up failed, or an error occurred

## Error Handling

The tool continues running even if individual benchmark iterations fail. Failed runs:
- Are excluded from statistical calculations
- Are reported in the summary
- Include error messages in the output files
- Affect the success rate metric

This allows you to benchmark flaky commands and understand their reliability.

## Tips

- Use `cargo clean` or equivalent cleanup commands as part of your benchmark command for consistent results
- Run multiple iterations (`-n 10` or more) for reliable statistics
- Store results in a dedicated directory for easier tracking: `--output-dir ./benchmark-results`
- Use meaningful names for easier identification: `--name cargo-clean-build-release`
- The tool uses `bash -c` to execute commands, so all shell features are supported
- Keep warm-up enabled (default) unless you specifically need to measure cold-start performance

## Statistics Explained

- **N**: Number of successful runs (used for statistics)
- **Mean**: Average execution time
- **Median**: Middle value when times are sorted (less affected by outliers)
- **Std Dev**: Standard deviation, measures variability
- **Min/Max**: Fastest and slowest execution times
- **P90**: 90th percentile - 90% of runs were faster than this
- **P95**: 95th percentile - 95% of runs were faster than this

## License

Apache 2.0
