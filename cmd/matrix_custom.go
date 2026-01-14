package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/attunehq/caliper/matrix"
	"github.com/spf13/cobra"
)

var (
	customImage     string
	customRepo      string
	customCommand   string
	customRuns      int
	customConfigs   string
	customOutputDir string
	customName      string
	customNoWarmup  bool
	customDebug     bool
)

var customCmd = &cobra.Command{
	Use:   "custom",
	Short: "Run benchmarks with arbitrary CPU:RAM configuration pairs",
	Long: `Run benchmarks across arbitrary CPU:RAM configurations in Docker containers.

This command allows you to specify exact CPU:RAM pairs to test, giving you full
control over which configurations to benchmark.`,
	Example: `  caliper matrix custom \
    --image ubuntu-2404-go-rust \
    --repo https://github.com/influxdata/influxdb \
    --runs 10 \
    --command "cargo clean && cargo build" \
    --configs "2:8,4:16,8:32,16:64,32:128"`,
	RunE: runCustom,
}

func init() {
	customCmd.Flags().StringVar(&customImage, "image", "", "Docker image to use (required)")
	customCmd.Flags().StringVar(&customRepo, "repo", "", "Git repository URL to clone (required)")
	customCmd.Flags().StringVarP(&customCommand, "command", "c", "", "Command to benchmark (required)")
	customCmd.Flags().IntVarP(&customRuns, "runs", "n", 10, "Number of benchmark runs per configuration")
	customCmd.Flags().StringVar(&customConfigs, "configs", "", "CPU:RAM configurations (e.g., '2:8,4:16,8:32') (required)")
	customCmd.Flags().StringVar(&customOutputDir, "output-dir", "./matrix-results", "Directory to save output files")
	customCmd.Flags().StringVar(&customName, "name", "", "Benchmark name for reports (default: timestamp)")
	customCmd.Flags().BoolVar(&customNoWarmup, "no-warmup", false, "Skip the warm-up run")
	customCmd.Flags().BoolVar(&customDebug, "debug", false, "Enable debug logging with real-time output")

	customCmd.MarkFlagRequired("image")
	customCmd.MarkFlagRequired("repo")
	customCmd.MarkFlagRequired("command")
	customCmd.MarkFlagRequired("configs")

	matrixCmd.AddCommand(customCmd)
}

func runCustom(cmd *cobra.Command, args []string) error {
	// Parse configurations
	resourceConfigs, err := matrix.ParseConfigs(customConfigs)
	if err != nil {
		return fmt.Errorf("error parsing configs: %w", err)
	}

	// Generate benchmark name if not provided
	benchmarkName := customName
	if benchmarkName == "" {
		benchmarkName = fmt.Sprintf("custom_%s", time.Now().Format("20060102_150405"))
	}

	// Create matrix configuration
	config := matrix.Config{
		Image:      customImage,
		RepoURL:    customRepo,
		Command:    customCommand,
		Runs:       customRuns,
		OutputDir:  customOutputDir,
		Name:       benchmarkName,
		Configs:    resourceConfigs,
		SkipWarmup: customNoWarmup,
		Debug:      customDebug,
		Type:       matrix.BenchmarkTypeCustom,
	}

	return runMatrixBenchmark(config)
}

// runMatrixBenchmark is a shared function to run matrix benchmarks
func runMatrixBenchmark(config matrix.Config) error {
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
		return fmt.Errorf("error building static binary: %w", err)
	}
	defer os.Remove(tmpBinary)

	// Run the matrix benchmark
	result, err := matrix.Run(ctx, config, tmpBinary)
	if err != nil {
		return fmt.Errorf("error running matrix benchmark: %w", err)
	}

	// Display summary table and graph(s)
	matrix.PrintSummaryTable(result)
	if config.Type == matrix.BenchmarkTypeAll {
		matrix.PrintAllGraphs(result)
	} else {
		matrix.PrintBuildTimeGraph(result)
	}

	// Save outputs (prefix with repo name and benchmark type)
	repoName := config.RepoName()
	typeStr := string(config.Type)

	jsonPath := filepath.Join(config.OutputDir, fmt.Sprintf("%s_%s_summary.json", repoName, typeStr))
	if err := matrix.SaveSummaryJSON(result, jsonPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to save JSON output: %v\n", err)
	} else {
		fmt.Printf("JSON summary saved to: %s\n", jsonPath)
	}

	csvPath := filepath.Join(config.OutputDir, fmt.Sprintf("%s_%s_summary.csv", repoName, typeStr))
	if err := matrix.SaveSummaryCSV(result, csvPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to save CSV output: %v\n", err)
	} else {
		fmt.Printf("CSV summary saved to: %s\n", csvPath)
	}

	mdPath := filepath.Join(config.OutputDir, fmt.Sprintf("%s_%s_summary.md", repoName, typeStr))
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

	return nil
}
