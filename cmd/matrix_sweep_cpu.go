package cmd

import (
	"fmt"
	"time"

	"github.com/attunehq/caliper/matrix"
	"github.com/spf13/cobra"
)

var (
	sweepCPUImage     string
	sweepCPURepo      string
	sweepCPUCommand   string
	sweepCPURuns      int
	sweepCPUCpus      string
	sweepCPURam       int
	sweepCPUOutputDir string
	sweepCPUName      string
	sweepCPUNoWarmup  bool
	sweepCPUDebug     bool
)

var sweepCPUCmd = &cobra.Command{
	Use:   "sweep-cpu",
	Short: "Run benchmarks varying CPU count with fixed RAM",
	Long: `Run benchmarks varying CPU count while keeping RAM constant.

This command helps you understand how build times scale with CPU count
for a given memory allocation.`,
	Example: `  caliper matrix sweep-cpu \
    --image ubuntu-2404-go-rust \
    --repo https://github.com/influxdata/influxdb \
    --runs 10 \
    --command "cargo clean && cargo build" \
    --ram 16 \
    --cpus "2,4,8,16,32"`,
	RunE: runSweepCPU,
}

func init() {
	sweepCPUCmd.Flags().StringVar(&sweepCPUImage, "image", "", "Docker image to use (required)")
	sweepCPUCmd.Flags().StringVar(&sweepCPURepo, "repo", "", "Git repository URL to clone (required)")
	sweepCPUCmd.Flags().StringVarP(&sweepCPUCommand, "command", "c", "", "Command to benchmark (required)")
	sweepCPUCmd.Flags().IntVarP(&sweepCPURuns, "runs", "n", 10, "Number of benchmark runs per configuration")
	sweepCPUCmd.Flags().StringVar(&sweepCPUCpus, "cpus", "", "CPU values to test (e.g., '2,4,8,16') (required)")
	sweepCPUCmd.Flags().IntVar(&sweepCPURam, "ram", 0, "Fixed RAM in GB (required)")
	sweepCPUCmd.Flags().StringVar(&sweepCPUOutputDir, "output-dir", "./matrix-results", "Directory to save output files")
	sweepCPUCmd.Flags().StringVar(&sweepCPUName, "name", "", "Benchmark name for reports (default: timestamp)")
	sweepCPUCmd.Flags().BoolVar(&sweepCPUNoWarmup, "no-warmup", false, "Skip the warm-up run")
	sweepCPUCmd.Flags().BoolVar(&sweepCPUDebug, "debug", false, "Enable debug logging with real-time output")

	sweepCPUCmd.MarkFlagRequired("image")
	sweepCPUCmd.MarkFlagRequired("repo")
	sweepCPUCmd.MarkFlagRequired("command")
	sweepCPUCmd.MarkFlagRequired("cpus")
	sweepCPUCmd.MarkFlagRequired("ram")

	matrixCmd.AddCommand(sweepCPUCmd)
}

func runSweepCPU(cmd *cobra.Command, args []string) error {
	// Parse CPU list
	cpuList, err := matrix.ParseIntList(sweepCPUCpus)
	if err != nil {
		return fmt.Errorf("error parsing cpus: %w", err)
	}

	// Validate RAM
	if sweepCPURam <= 0 {
		return fmt.Errorf("ram must be a positive integer")
	}

	// Generate configurations
	resourceConfigs := matrix.GenerateSweepCPUConfigs(cpuList, sweepCPURam)

	// Generate benchmark name if not provided
	benchmarkName := sweepCPUName
	if benchmarkName == "" {
		benchmarkName = fmt.Sprintf("sweep-cpu_%s", time.Now().Format("20060102_150405"))
	}

	// Create matrix configuration
	config := matrix.Config{
		Image:      sweepCPUImage,
		RepoURL:    sweepCPURepo,
		Command:    sweepCPUCommand,
		Runs:       sweepCPURuns,
		OutputDir:  sweepCPUOutputDir,
		Name:       benchmarkName,
		Configs:    resourceConfigs,
		SkipWarmup: sweepCPUNoWarmup,
		Debug:      sweepCPUDebug,
		Type:       matrix.BenchmarkTypeSweepCPU,
		FixedRAM:   sweepCPURam,
		CPUList:    cpuList,
	}

	return runMatrixBenchmark(config)
}
