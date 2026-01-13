package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/attunehq/caliper/benchmark"
	"github.com/spf13/cobra"
)

var (
	// Version is set at build time
	Version = "dev"

	// Flags for root command (single benchmark)
	runs      int
	command   string
	outputDir string
	name      string
	noWarmup  bool
	debug     bool
)

var rootCmd = &cobra.Command{
	Use:   "caliper",
	Short: "A CLI tool for benchmarking commands",
	Long: `Caliper is a benchmarking tool that runs commands multiple times
and provides detailed statistics about execution time, memory usage, and more.

Run a single benchmark:
  caliper -n 10 -c "make build"

Run benchmarks across multiple CPU/RAM configurations:
  caliper matrix --image ubuntu:22.04 --repo https://github.com/user/repo --configs "2:8,4:16"`,
	Version: Version,
	RunE:    runBenchmark,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

// SetVersion sets the version string (called from main)
func SetVersion(v string) {
	Version = v
	rootCmd.Version = v
}

func init() {
	rootCmd.Flags().IntVarP(&runs, "runs", "n", 0, "Number of times to run the benchmark (required)")
	rootCmd.Flags().StringVarP(&command, "command", "c", "", "Command to benchmark (required)")
	rootCmd.Flags().StringVar(&outputDir, "output-dir", ".", "Directory to save output files")
	rootCmd.Flags().StringVar(&name, "name", "", "Benchmark name for reports (default: timestamp)")
	rootCmd.Flags().BoolVar(&noWarmup, "no-warmup", false, "Skip the warm-up run")
	rootCmd.Flags().BoolVar(&debug, "debug", false, "Enable debug logging with real-time command output")
}

func runBenchmark(cmd *cobra.Command, args []string) error {
	// If no flags provided, show help
	if runs == 0 && command == "" {
		return cmd.Help()
	}

	// Validate required arguments
	if runs <= 0 {
		return fmt.Errorf("--runs/-n is required and must be greater than 0")
	}

	if command == "" {
		return fmt.Errorf("--command/-c is required")
	}

	// Generate benchmark name if not provided
	benchmarkName := name
	if benchmarkName == "" {
		benchmarkName = fmt.Sprintf("benchmark_%s", time.Now().Format("20060102_150405"))
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %w", err)
	}

	// Create benchmark configuration
	config := benchmark.Config{
		Command:    command,
		Runs:       runs,
		Name:       benchmarkName,
		OutputDir:  outputDir,
		SkipWarmup: noWarmup,
		Debug:      debug,
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
		return fmt.Errorf("error running benchmark: %w", err)
	}

	// Display results to console
	benchmark.PrintConsole(result)

	// Save outputs
	jsonPath := filepath.Join(outputDir, fmt.Sprintf("%s.json", benchmarkName))
	if err := benchmark.SaveJSON(result, jsonPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to save JSON output: %v\n", err)
	} else {
		fmt.Printf("\nJSON output saved to: %s\n", jsonPath)
	}

	csvPath := filepath.Join(outputDir, fmt.Sprintf("%s.csv", benchmarkName))
	if err := benchmark.SaveCSV(result, csvPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to save CSV output: %v\n", err)
	} else {
		fmt.Printf("CSV output saved to: %s\n", csvPath)
	}

	mdPath := filepath.Join(outputDir, fmt.Sprintf("%s.md", benchmarkName))
	if err := benchmark.SaveMarkdown(result, mdPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to save Markdown output: %v\n", err)
	} else {
		fmt.Printf("Markdown report saved to: %s\n", mdPath)
	}

	// Exit with appropriate code
	if result.SuccessRate < 100.0 {
		os.Exit(1)
	}

	return nil
}
