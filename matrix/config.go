package matrix

import (
	"fmt"
	"strconv"
	"strings"
)

// BenchmarkType represents the type of matrix benchmark being run
type BenchmarkType string

const (
	BenchmarkTypeCustom   BenchmarkType = "custom"
	BenchmarkTypeSweepCPU BenchmarkType = "sweep-cpu"
	BenchmarkTypeSweepRAM BenchmarkType = "sweep-ram"
	BenchmarkTypeAll      BenchmarkType = "all"
)

// ResourceConfig represents a single CPU/RAM configuration
type ResourceConfig struct {
	CPUs   int // Number of CPUs
	Memory int // RAM in GB
}

// String returns a human-readable representation of the config
func (r ResourceConfig) String() string {
	return fmt.Sprintf("%d CPU, %d GB", r.CPUs, r.Memory)
}

// DirName returns a directory-safe name for the config
func (r ResourceConfig) DirName() string {
	return fmt.Sprintf("%dcpu_%dgb", r.CPUs, r.Memory)
}

// Config holds the matrix benchmark configuration
type Config struct {
	Image      string           // Docker image name
	RepoURL    string           // Git repository URL to clone
	Command    string           // Benchmark command to run
	Runs       int              // Number of benchmark runs per configuration
	OutputDir  string           // Directory to save output files
	Name       string           // Benchmark name for reports
	Configs    []ResourceConfig // CPU/RAM configurations to test
	SkipWarmup bool             // Skip warm-up run
	Debug      bool             // Enable debug logging with real-time output
	Type       BenchmarkType    // Type of benchmark (custom, sweep-cpu, sweep-ram, all)
	FixedCPU   int              // For sweep-ram: the fixed CPU value
	FixedRAM   int              // For sweep-cpu: the fixed RAM value
	CPUList    []int            // For all: list of CPU values tested
	RAMList    []int            // For all: list of RAM values tested
}

// RepoName extracts the repository name from the RepoURL
func (c Config) RepoName() string {
	// Remove trailing .git if present
	url := strings.TrimSuffix(c.RepoURL, ".git")

	// Get the last part of the path
	parts := strings.Split(url, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return "repo"
}

// ConfigResult holds the result for a single configuration
type ConfigResult struct {
	Config      ResourceConfig
	Success     bool
	Error       string
	Mean        float64 // Mean duration in seconds
	Median      float64 // Median duration in seconds
	StdDev      float64 // Standard deviation in seconds
	Min         float64 // Minimum duration in seconds
	Max         float64 // Maximum duration in seconds
	P90         float64 // 90th percentile in seconds
	P95         float64 // 95th percentile in seconds
	SuccessRate float64 // Percentage of successful runs
	TotalRuns   int     // Total number of runs attempted
	SuccessRuns int     // Number of successful runs
}

// MatrixResult holds the complete matrix benchmark results
type MatrixResult struct {
	Config  Config
	Results []ConfigResult
}

// ParseConfigs parses a config string like "2:8,4:16,8:32" into ResourceConfig slice
func ParseConfigs(configStr string) ([]ResourceConfig, error) {
	if configStr == "" {
		return nil, fmt.Errorf("config string cannot be empty")
	}

	pairs := strings.Split(configStr, ",")
	configs := make([]ResourceConfig, 0, len(pairs))

	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		parts := strings.Split(pair, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid config format '%s': expected 'CPU:RAM' (e.g., '2:8')", pair)
		}

		cpus, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || cpus <= 0 {
			return nil, fmt.Errorf("invalid CPU value '%s': must be a positive integer", parts[0])
		}

		memory, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil || memory <= 0 {
			return nil, fmt.Errorf("invalid memory value '%s': must be a positive integer (GB)", parts[1])
		}

		configs = append(configs, ResourceConfig{
			CPUs:   cpus,
			Memory: memory,
		})
	}

	return configs, nil
}

// ParseIntList parses a comma-separated string of integers like "2,4,8,16" into []int
func ParseIntList(str string) ([]int, error) {
	if str == "" {
		return nil, fmt.Errorf("list string cannot be empty")
	}

	parts := strings.Split(str, ",")
	result := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		val, err := strconv.Atoi(part)
		if err != nil || val <= 0 {
			return nil, fmt.Errorf("invalid value '%s': must be a positive integer", part)
		}
		result = append(result, val)
	}

	return result, nil
}

// GenerateSweepCPUConfigs creates configs with varying CPUs and fixed RAM
func GenerateSweepCPUConfigs(cpus []int, fixedRAM int) []ResourceConfig {
	configs := make([]ResourceConfig, 0, len(cpus))
	for _, cpu := range cpus {
		configs = append(configs, ResourceConfig{
			CPUs:   cpu,
			Memory: fixedRAM,
		})
	}
	return configs
}

// GenerateSweepRAMConfigs creates configs with varying RAM and fixed CPU
func GenerateSweepRAMConfigs(rams []int, fixedCPU int) []ResourceConfig {
	configs := make([]ResourceConfig, 0, len(rams))
	for _, ram := range rams {
		configs = append(configs, ResourceConfig{
			CPUs:   fixedCPU,
			Memory: ram,
		})
	}
	return configs
}

// GenerateGridConfigs creates configs for all CPU x RAM combinations
// Ordered by CPU first, then RAM (e.g., 2:8, 2:16, 2:32, 4:8, 4:16, ...)
func GenerateGridConfigs(cpus []int, rams []int) []ResourceConfig {
	configs := make([]ResourceConfig, 0, len(cpus)*len(rams))
	for _, cpu := range cpus {
		for _, ram := range rams {
			configs = append(configs, ResourceConfig{
				CPUs:   cpu,
				Memory: ram,
			})
		}
	}
	return configs
}
