package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/attunehq/caliper/benchmark"
	"github.com/attunehq/caliper/matrix"
)

var version = "dev"

func main() {
	// Check for version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("caliper %s\n", version)
		return
	}

	// Check for subcommand
	if len(os.Args) > 1 && os.Args[1] == "matrix" {
		runMatrix(os.Args[2:])
		return
	}

	// Original single-benchmark mode
	runSingleBenchmark()
}

func runSingleBenchmark() {
	// Define CLI flags
	runs := flag.Int("runs", 0, "Number of times to run the benchmark (required)")
	runsShort := flag.Int("n", 0, "Number of times to run the benchmark (shorthand)")
	command := flag.String("command", "", "Command to benchmark (required)")
	commandShort := flag.String("c", "", "Command to benchmark (shorthand)")
	outputDir := flag.String("output-dir", ".", "Directory to save output files")
	name := flag.String("name", "", "Benchmark name for reports (default: timestamp)")
	noWarmup := flag.Bool("no-warmup", false, "Skip the warm-up run")
	debug := flag.Bool("debug", false, "Enable debug logging with real-time command output")

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
		Debug:      *debug,
	}

	fmt.Printf("Caliper\n")
	fmt.Printf("=======\n")
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

func runMatrix(args []string) {
	// Create a new FlagSet for the matrix subcommand
	matrixFlags := flag.NewFlagSet("matrix", flag.ExitOnError)

	image := matrixFlags.String("image", "", "Docker image to use (required)")
	repo := matrixFlags.String("repo", "", "Git repository URL to clone (required)")
	command := matrixFlags.String("command", "", "Command to benchmark (required)")
	commandShort := matrixFlags.String("c", "", "Command to benchmark (shorthand)")
	runs := matrixFlags.Int("runs", 10, "Number of benchmark runs per configuration")
	runsShort := matrixFlags.Int("n", 0, "Number of benchmark runs (shorthand)")
	configs := matrixFlags.String("configs", "", "CPU:RAM configurations (e.g., '2:8,4:16,8:32') (required)")
	outputDir := matrixFlags.String("output-dir", "./matrix-results", "Directory to save output files")
	name := matrixFlags.String("name", "", "Benchmark name for reports (default: timestamp)")
	noWarmup := matrixFlags.Bool("no-warmup", false, "Skip the warm-up run")
	debug := matrixFlags.Bool("debug", false, "Enable debug logging with real-time output")

	matrixFlags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: caliper matrix [options]\n\n")
		fmt.Fprintf(os.Stderr, "Run benchmarks across multiple CPU/RAM configurations in Docker containers.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		matrixFlags.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  caliper matrix \\\n")
		fmt.Fprintf(os.Stderr, "    --image ubuntu-2404-go-rust \\\n")
		fmt.Fprintf(os.Stderr, "    --repo https://github.com/influxdata/influxdb \\\n")
		fmt.Fprintf(os.Stderr, "    --runs 10 \\\n")
		fmt.Fprintf(os.Stderr, "    --command \"cargo clean && cargo build\" \\\n")
		fmt.Fprintf(os.Stderr, "    --configs \"2:8,4:16,8:32,16:64,32:128\"\n")
	}

	if err := matrixFlags.Parse(args); err != nil {
		os.Exit(1)
	}

	// Handle shorthand flags
	cmd := *command
	if cmd == "" {
		cmd = *commandShort
	}

	numRuns := *runs
	if *runsShort > 0 {
		numRuns = *runsShort
	}

	// Validate required arguments
	if *image == "" {
		fmt.Fprintf(os.Stderr, "Error: --image is required\n\n")
		matrixFlags.Usage()
		os.Exit(1)
	}

	if *repo == "" {
		fmt.Fprintf(os.Stderr, "Error: --repo is required\n\n")
		matrixFlags.Usage()
		os.Exit(1)
	}

	if cmd == "" {
		fmt.Fprintf(os.Stderr, "Error: --command/-c is required\n\n")
		matrixFlags.Usage()
		os.Exit(1)
	}

	if *configs == "" {
		fmt.Fprintf(os.Stderr, "Error: --configs is required\n\n")
		matrixFlags.Usage()
		os.Exit(1)
	}

	// Parse configurations
	resourceConfigs, err := matrix.ParseConfigs(*configs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing configs: %v\n", err)
		os.Exit(1)
	}

	// Generate benchmark name if not provided
	benchmarkName := *name
	if benchmarkName == "" {
		benchmarkName = fmt.Sprintf("matrix_%s", time.Now().Format("20060102_150405"))
	}

	// Create matrix configuration
	config := matrix.Config{
		Image:      *image,
		RepoURL:    *repo,
		Command:    cmd,
		Runs:       numRuns,
		OutputDir:  *outputDir,
		Name:       benchmarkName,
		Configs:    resourceConfigs,
		SkipWarmup: *noWarmup,
		Debug:      *debug,
	}

	// Set up context with cancellation on interrupt
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, cleaning up...")
		cancel()
	}()

	// Build the static binary for Linux containers
	tmpBinary := filepath.Join(os.TempDir(), "caliper-linux")
	if err := matrix.BuildStaticBinary(tmpBinary); err != nil {
		fmt.Fprintf(os.Stderr, "Error building static binary: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(tmpBinary)

	// Run the matrix benchmark
	result, err := matrix.Run(ctx, config, tmpBinary)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running matrix benchmark: %v\n", err)
		os.Exit(1)
	}

	// Display summary table and graph
	matrix.PrintSummaryTable(result)
	matrix.PrintBuildTimeGraph(result)

	// Save outputs (prefix with repo name)
	repoName := config.RepoName()
	jsonPath := filepath.Join(*outputDir, fmt.Sprintf("%s_matrix_summary.json", repoName))
	if err := matrix.SaveSummaryJSON(result, jsonPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to save JSON output: %v\n", err)
	} else {
		fmt.Printf("JSON summary saved to: %s\n", jsonPath)
	}

	csvPath := filepath.Join(*outputDir, fmt.Sprintf("%s_matrix_summary.csv", repoName))
	if err := matrix.SaveSummaryCSV(result, csvPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to save CSV output: %v\n", err)
	} else {
		fmt.Printf("CSV summary saved to: %s\n", csvPath)
	}

	mdPath := filepath.Join(*outputDir, fmt.Sprintf("%s_matrix_summary.md", repoName))
	if err := matrix.SaveSummaryMarkdown(result, mdPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to save Markdown output: %v\n", err)
	} else {
		fmt.Printf("Markdown report saved to: %s\n", mdPath)
	}

	// Exit with appropriate code if any configuration failed
	for _, r := range result.Results {
		if !r.Success {
			os.Exit(1)
		}
	}
}
