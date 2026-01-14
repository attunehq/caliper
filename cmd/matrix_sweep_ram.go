package cmd

import (
	"fmt"
	"time"

	"github.com/attunehq/caliper/matrix"
	"github.com/spf13/cobra"
)

var (
	sweepRAMImage     string
	sweepRAMRepo      string
	sweepRAMCommand   string
	sweepRAMRuns      int
	sweepRAMRams      string
	sweepRAMCpu       int
	sweepRAMOutputDir string
	sweepRAMName      string
	sweepRAMNoWarmup  bool
	sweepRAMDebug     bool
)

var sweepRAMCmd = &cobra.Command{
	Use:   "sweep-ram",
	Short: "Run benchmarks varying RAM with fixed CPU count",
	Long: `Run benchmarks varying RAM while keeping CPU count constant.

This command helps you understand how build times scale with memory
for a given CPU allocation.`,
	Example: `  caliper matrix sweep-ram \
    --image ubuntu-2404-go-rust \
    --repo https://github.com/influxdata/influxdb \
    --runs 10 \
    --command "cargo clean && cargo build" \
    --cpu 4 \
    --rams "8,16,32,64"`,
	RunE: runSweepRAM,
}

func init() {
	sweepRAMCmd.Flags().StringVar(&sweepRAMImage, "image", "", "Docker image to use (required)")
	sweepRAMCmd.Flags().StringVar(&sweepRAMRepo, "repo", "", "Git repository URL to clone (required)")
	sweepRAMCmd.Flags().StringVarP(&sweepRAMCommand, "command", "c", "", "Command to benchmark (required)")
	sweepRAMCmd.Flags().IntVarP(&sweepRAMRuns, "runs", "n", 10, "Number of benchmark runs per configuration")
	sweepRAMCmd.Flags().StringVar(&sweepRAMRams, "rams", "", "RAM values in GB to test (e.g., '8,16,32,64') (required)")
	sweepRAMCmd.Flags().IntVar(&sweepRAMCpu, "cpu", 0, "Fixed CPU count (required)")
	sweepRAMCmd.Flags().StringVar(&sweepRAMOutputDir, "output-dir", "./matrix-results", "Directory to save output files")
	sweepRAMCmd.Flags().StringVar(&sweepRAMName, "name", "", "Benchmark name for reports (default: timestamp)")
	sweepRAMCmd.Flags().BoolVar(&sweepRAMNoWarmup, "no-warmup", false, "Skip the warm-up run")
	sweepRAMCmd.Flags().BoolVar(&sweepRAMDebug, "debug", false, "Enable debug logging with real-time output")

	sweepRAMCmd.MarkFlagRequired("image")
	sweepRAMCmd.MarkFlagRequired("repo")
	sweepRAMCmd.MarkFlagRequired("command")
	sweepRAMCmd.MarkFlagRequired("rams")
	sweepRAMCmd.MarkFlagRequired("cpu")

	matrixCmd.AddCommand(sweepRAMCmd)
}

func runSweepRAM(cmd *cobra.Command, args []string) error {
	// Parse RAM list
	ramList, err := matrix.ParseIntList(sweepRAMRams)
	if err != nil {
		return fmt.Errorf("error parsing rams: %w", err)
	}

	// Validate CPU
	if sweepRAMCpu <= 0 {
		return fmt.Errorf("cpu must be a positive integer")
	}

	// Generate configurations
	resourceConfigs := matrix.GenerateSweepRAMConfigs(ramList, sweepRAMCpu)

	// Generate benchmark name if not provided
	benchmarkName := sweepRAMName
	if benchmarkName == "" {
		benchmarkName = fmt.Sprintf("sweep-ram_%s", time.Now().Format("20060102_150405"))
	}

	// Create matrix configuration
	config := matrix.Config{
		Image:      sweepRAMImage,
		RepoURL:    sweepRAMRepo,
		Command:    sweepRAMCommand,
		Runs:       sweepRAMRuns,
		OutputDir:  sweepRAMOutputDir,
		Name:       benchmarkName,
		Configs:    resourceConfigs,
		SkipWarmup: sweepRAMNoWarmup,
		Debug:      sweepRAMDebug,
		Type:       matrix.BenchmarkTypeSweepRAM,
		FixedCPU:   sweepRAMCpu,
		RAMList:    ramList,
	}

	return runMatrixBenchmark(config)
}
