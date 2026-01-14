package cmd

import (
	"fmt"
	"time"

	"github.com/attunehq/caliper/matrix"
	"github.com/spf13/cobra"
)

var (
	allImage     string
	allRepo      string
	allCommand   string
	allRuns      int
	allCpus      string
	allRams      string
	allOutputDir string
	allName      string
	allNoWarmup  bool
	allDebug     bool
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Run benchmarks across a full CPU x RAM grid",
	Long: `Run benchmarks across all combinations of CPU and RAM values.

This command tests every combination of the specified CPU and RAM values,
providing comprehensive data on how both resources affect build times.
Multiple graphs are generated showing CPU scaling (one per RAM value)
and RAM scaling (one per CPU value).`,
	Example: `  caliper matrix all \
    --image ubuntu-2404-go-rust \
    --repo https://github.com/influxdata/influxdb \
    --runs 10 \
    --command "cargo clean && cargo build" \
    --cpus "2,4,8,16" \
    --rams "8,16,32,64"

This will test 16 configurations (4 CPUs x 4 RAMs) and generate:
  - 4 graphs showing CPU scaling (one per RAM value)
  - 4 graphs showing RAM scaling (one per CPU value)`,
	RunE: runAll,
}

func init() {
	allCmd.Flags().StringVar(&allImage, "image", "", "Docker image to use (required)")
	allCmd.Flags().StringVar(&allRepo, "repo", "", "Git repository URL to clone (required)")
	allCmd.Flags().StringVarP(&allCommand, "command", "c", "", "Command to benchmark (required)")
	allCmd.Flags().IntVarP(&allRuns, "runs", "n", 10, "Number of benchmark runs per configuration")
	allCmd.Flags().StringVar(&allCpus, "cpus", "", "CPU values to test (e.g., '2,4,8,16') (required)")
	allCmd.Flags().StringVar(&allRams, "rams", "", "RAM values in GB to test (e.g., '8,16,32,64') (required)")
	allCmd.Flags().StringVar(&allOutputDir, "output-dir", "./matrix-results", "Directory to save output files")
	allCmd.Flags().StringVar(&allName, "name", "", "Benchmark name for reports (default: timestamp)")
	allCmd.Flags().BoolVar(&allNoWarmup, "no-warmup", false, "Skip the warm-up run")
	allCmd.Flags().BoolVar(&allDebug, "debug", false, "Enable debug logging with real-time output")

	allCmd.MarkFlagRequired("image")
	allCmd.MarkFlagRequired("repo")
	allCmd.MarkFlagRequired("command")
	allCmd.MarkFlagRequired("cpus")
	allCmd.MarkFlagRequired("rams")

	matrixCmd.AddCommand(allCmd)
}

func runAll(cmd *cobra.Command, args []string) error {
	// Parse CPU list
	cpuList, err := matrix.ParseIntList(allCpus)
	if err != nil {
		return fmt.Errorf("error parsing cpus: %w", err)
	}

	// Parse RAM list
	ramList, err := matrix.ParseIntList(allRams)
	if err != nil {
		return fmt.Errorf("error parsing rams: %w", err)
	}

	// Generate full grid configurations (CPU first, then RAM)
	resourceConfigs := matrix.GenerateGridConfigs(cpuList, ramList)

	// Generate benchmark name if not provided
	benchmarkName := allName
	if benchmarkName == "" {
		benchmarkName = fmt.Sprintf("all_%s", time.Now().Format("20060102_150405"))
	}

	// Create matrix configuration
	config := matrix.Config{
		Image:      allImage,
		RepoURL:    allRepo,
		Command:    allCommand,
		Runs:       allRuns,
		OutputDir:  allOutputDir,
		Name:       benchmarkName,
		Configs:    resourceConfigs,
		SkipWarmup: allNoWarmup,
		Debug:      allDebug,
		Type:       matrix.BenchmarkTypeAll,
		CPUList:    cpuList,
		RAMList:    ramList,
	}

	return runMatrixBenchmark(config)
}
