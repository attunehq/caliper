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
	// Flags for matrix command
	matrixImage     string
	matrixRepo      string
	matrixCommand   string
	matrixRuns      int
	matrixConfigs   string
	matrixOutputDir string
	matrixName      string
	matrixNoWarmup  bool
	matrixDebug     bool
)

var matrixCmd = &cobra.Command{
	Use:   "matrix",
	Short: "Run benchmarks across multiple CPU/RAM configurations",
	Long: `Run benchmarks across multiple CPU/RAM configurations in Docker containers.

This command allows you to test how a command performs with different resource
allocations, helping you understand scaling characteristics and resource requirements.`,
	Example: `  caliper matrix \
    --image ubuntu-2404-go-rust \
    --repo https://github.com/influxdata/influxdb \
    --runs 10 \
    --command "cargo clean && cargo build" \
    --configs "2:8,4:16,8:32,16:64,32:128"`,
	RunE: runMatrix,
}

func init() {
	matrixCmd.Flags().StringVar(&matrixImage, "image", "", "Docker image to use (required)")
	matrixCmd.Flags().StringVar(&matrixRepo, "repo", "", "Git repository URL to clone (required)")
	matrixCmd.Flags().StringVarP(&matrixCommand, "command", "c", "", "Command to benchmark (required)")
	matrixCmd.Flags().IntVarP(&matrixRuns, "runs", "n", 10, "Number of benchmark runs per configuration")
	matrixCmd.Flags().StringVar(&matrixConfigs, "configs", "", "CPU:RAM configurations (e.g., '2:8,4:16,8:32') (required)")
	matrixCmd.Flags().StringVar(&matrixOutputDir, "output-dir", "./matrix-results", "Directory to save output files")
	matrixCmd.Flags().StringVar(&matrixName, "name", "", "Benchmark name for reports (default: timestamp)")
	matrixCmd.Flags().BoolVar(&matrixNoWarmup, "no-warmup", false, "Skip the warm-up run")
	matrixCmd.Flags().BoolVar(&matrixDebug, "debug", false, "Enable debug logging with real-time output")

	// Mark required flags
	matrixCmd.MarkFlagRequired("image")
	matrixCmd.MarkFlagRequired("repo")
	matrixCmd.MarkFlagRequired("command")
	matrixCmd.MarkFlagRequired("configs")

	// Register with root command
	rootCmd.AddCommand(matrixCmd)
}

func runMatrix(cmd *cobra.Command, args []string) error {
	// Parse configurations
	resourceConfigs, err := matrix.ParseConfigs(matrixConfigs)
	if err != nil {
		return fmt.Errorf("error parsing configs: %w", err)
	}

	// Generate benchmark name if not provided
	benchmarkName := matrixName
	if benchmarkName == "" {
		benchmarkName = fmt.Sprintf("matrix_%s", time.Now().Format("20060102_150405"))
	}

	// Create matrix configuration
	config := matrix.Config{
		Image:      matrixImage,
		RepoURL:    matrixRepo,
		Command:    matrixCommand,
		Runs:       matrixRuns,
		OutputDir:  matrixOutputDir,
		Name:       benchmarkName,
		Configs:    resourceConfigs,
		SkipWarmup: matrixNoWarmup,
		Debug:      matrixDebug,
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
		return fmt.Errorf("error building static binary: %w", err)
	}
	defer os.Remove(tmpBinary)

	// Run the matrix benchmark
	result, err := matrix.Run(ctx, config, tmpBinary)
	if err != nil {
		return fmt.Errorf("error running matrix benchmark: %w", err)
	}

	// Display summary table and graph
	matrix.PrintSummaryTable(result)
	matrix.PrintBuildTimeGraph(result)

	// Save outputs (prefix with repo name)
	repoName := config.RepoName()
	jsonPath := filepath.Join(matrixOutputDir, fmt.Sprintf("%s_matrix_summary.json", repoName))
	if err := matrix.SaveSummaryJSON(result, jsonPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to save JSON output: %v\n", err)
	} else {
		fmt.Printf("JSON summary saved to: %s\n", jsonPath)
	}

	csvPath := filepath.Join(matrixOutputDir, fmt.Sprintf("%s_matrix_summary.csv", repoName))
	if err := matrix.SaveSummaryCSV(result, csvPath); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to save CSV output: %v\n", err)
	} else {
		fmt.Printf("CSV summary saved to: %s\n", csvPath)
	}

	mdPath := filepath.Join(matrixOutputDir, fmt.Sprintf("%s_matrix_summary.md", repoName))
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
