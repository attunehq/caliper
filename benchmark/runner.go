package benchmark

import (
	"fmt"
	"os/exec"
	"time"
)

// Config holds the benchmark configuration
type Config struct {
	Command    string
	Runs       int
	Name       string
	OutputDir  string
	SkipWarmup bool
}

// RunResult holds the result of a single benchmark run
type RunResult struct {
	RunNumber int
	Duration  time.Duration
	Success   bool
	Error     string
}

// Result holds the complete benchmark results
type Result struct {
	Config        Config
	WarmupRun     *RunResult
	Runs          []RunResult
	Stats         Statistics
	SuccessRate   float64
	StartTime     time.Time
	EndTime       time.Time
	TotalDuration time.Duration
}

// Run executes the benchmark according to the provided configuration
func Run(config Config) (*Result, error) {
	result := &Result{
		Config:    config,
		Runs:      make([]RunResult, 0, config.Runs),
		StartTime: time.Now(),
	}

	fmt.Printf("Starting benchmark...\n\n")

	// Execute warm-up run if enabled
	if !config.SkipWarmup {
		fmt.Printf("Warm-up: ")
		warmupResult := executeCommand(0, config.Command)
		result.WarmupRun = &warmupResult

		if warmupResult.Success {
			fmt.Printf("✓ Completed in %v (excluded from stats)\n", warmupResult.Duration)
		} else {
			fmt.Printf("✗ Failed: %s\n", warmupResult.Error)
			return nil, fmt.Errorf("warm-up run failed: %s", warmupResult.Error)
		}
		fmt.Println()
	}

	for i := 1; i <= config.Runs; i++ {
		fmt.Printf("Run %d/%d: ", i, config.Runs)

		runResult := executeCommand(i, config.Command)
		result.Runs = append(result.Runs, runResult)

		if runResult.Success {
			fmt.Printf("✓ Completed in %v\n", runResult.Duration)
		} else {
			fmt.Printf("✗ Failed: %s\n", runResult.Error)
		}
	}

	result.EndTime = time.Now()
	result.TotalDuration = result.EndTime.Sub(result.StartTime)

	// Calculate statistics
	successCount := 0
	durations := make([]float64, 0, config.Runs)

	for _, run := range result.Runs {
		if run.Success {
			successCount++
			durations = append(durations, run.Duration.Seconds())
		}
	}

	result.SuccessRate = (float64(successCount) / float64(config.Runs)) * 100.0

	if len(durations) > 0 {
		result.Stats = CalculateStatistics(durations)
	}

	return result, nil
}

// executeCommand runs a single benchmark iteration
func executeCommand(runNumber int, command string) RunResult {
	result := RunResult{
		RunNumber: runNumber,
	}

	// Use bash to execute the command (supports && and other shell features)
	cmd := exec.Command("bash", "-c", command)

	startTime := time.Now()
	err := cmd.Run()
	result.Duration = time.Since(startTime)

	if err != nil {
		result.Success = false
		result.Error = err.Error()
	} else {
		result.Success = true
	}

	return result
}
