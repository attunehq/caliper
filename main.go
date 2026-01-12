package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/attunehq/ci-benchmarking/benchmark"
)

func main() {
	// Define CLI flags
	runs := flag.Int("runs", 0, "Number of times to run the benchmark (required)")
	runsShort := flag.Int("n", 0, "Number of times to run the benchmark (shorthand)")
	command := flag.String("command", "", "Command to benchmark (required)")
	commandShort := flag.String("c", "", "Command to benchmark (shorthand)")
	outputDir := flag.String("output-dir", ".", "Directory to save output files")
	name := flag.String("name", "", "Benchmark name for reports (default: timestamp)")
	noWarmup := flag.Bool("no-warmup", false, "Skip the warm-up run")

	flag.Parse()

	// Handle shorthand flags
	numRuns := *runs
	if numRuns == 0 {
		numRuns = *runsShort
	}

	cmd := *command
	if cmd == "" {
		cmd = *commandShort
	}

	// Validate required arguments
	if numRuns <= 0 {
		fmt.Fprintf(os.Stderr, "Error: --runs/-n is required and must be greater than 0\n\n")
		flag.Usage()
		os.Exit(1)
	}

	if cmd == "" {
		fmt.Fprintf(os.Stderr, "Error: --command/-c is required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	// Generate benchmark name if not provided
	benchmarkName := *name
	if benchmarkName == "" {
		benchmarkName = fmt.Sprintf("benchmark_%s", time.Now().Format("20060102_150405"))
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Create benchmark configuration
	config := benchmark.Config{
		Command:    cmd,
		Runs:       numRuns,
		Name:       benchmarkName,
		OutputDir:  *outputDir,
		SkipWarmup: *noWarmup,
	}

	fmt.Printf("CI Benchmark Tool\n")
	fmt.Printf("=================\n")
	fmt.Printf("Command: %s\n", config.Command)
	if config.SkipWarmup {
		fmt.Printf("Runs: %d (no warm-up)\n", config.Runs)
	} else {
		fmt.Printf("Runs: %d (+ 1 warm-up)\n", config.Runs)
	}
	fmt.Printf("Output Directory: %s\n\n", config.OutputDir)

	// Run the benchmark
	result, err := benchmark.Run(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running benchmark: %v\n", err)
		os.Exit(1)
	}

	// Display results to console
	benchmark.PrintConsole(result)

	// Save outputs
	jsonPath := filepath.Join(*outputDir, fmt.Sprintf("%s.json", benchmarkName))
	if err := benchmark.SaveJSON(result, jsonPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to save JSON output: %v\n", err)
	} else {
		fmt.Printf("\nJSON output saved to: %s\n", jsonPath)
	}

	csvPath := filepath.Join(*outputDir, fmt.Sprintf("%s.csv", benchmarkName))
	if err := benchmark.SaveCSV(result, csvPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to save CSV output: %v\n", err)
	} else {
		fmt.Printf("CSV output saved to: %s\n", csvPath)
	}

	mdPath := filepath.Join(*outputDir, fmt.Sprintf("%s.md", benchmarkName))
	if err := benchmark.SaveMarkdown(result, mdPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to save Markdown output: %v\n", err)
	} else {
		fmt.Printf("Markdown report saved to: %s\n", mdPath)
	}

	// Exit with appropriate code
	if result.SuccessRate < 100.0 {
		os.Exit(1)
	}
}
