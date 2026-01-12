# CI Benchmarking Tool

A flexible command-line tool for benchmarking CI commands with statistical analysis and multiple output formats.

## Features

- Run any shell command multiple times and measure execution time
- **Warm-up run** by default to eliminate cold-start effects (caches, JIT, filesystem)
- Handles failures gracefully and continues benchmarking
- Calculates comprehensive statistics: mean, median, standard deviation, min, max, P90, P95
- Outputs results in multiple formats: console, JSON, CSV, and Markdown
- Tracks success rate and provides detailed error reporting

## Installation

### Build from source

```bash
go build -o ci-benchmark
```

### Install globally (optional)

```bash
go install
```

## Usage

### Basic Command

```bash
./ci-benchmark --runs 10 --command "cargo clean && cargo build"
```

### Shorthand Flags

```bash
./ci-benchmark -n 10 -c "cargo clean && cargo build"
```

### With Custom Output Directory and Name

```bash
./ci-benchmark -n 5 -c "npm test" --output-dir ./results --name npm-test-benchmark
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
./ci-benchmark -n 10 -c "cargo clean && cargo build"
```

### Benchmarking with Release Mode

```bash
./ci-benchmark -n 5 -c "cargo clean && cargo build --release" --name cargo-release
```

### Benchmarking Tests

```bash
./ci-benchmark -n 20 -c "npm run test" --name npm-tests --output-dir ./benchmark-results
```

### Complex Shell Commands

```bash
./ci-benchmark -n 3 -c "docker-compose down && docker-compose up -d && npm test && docker-compose down"
```

### Skip Warm-up Run

```bash
./ci-benchmark -n 10 -c "cargo build" --no-warmup
```

## Output Format

### Console Output

```
CI Benchmark Tool
=================
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
