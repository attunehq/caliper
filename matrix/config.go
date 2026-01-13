package matrix

import (
	"fmt"
	"strconv"
	"strings"
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
