package matrix

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Run executes the matrix benchmark with all configurations sequentially
// binaryPath should be a path to a Linux-compatible ci-benchmark binary
func Run(ctx context.Context, config Config, binaryPath string) (*MatrixResult, error) {
	result := &MatrixResult{
		Config:  config,
		Results: make([]ConfigResult, 0, len(config.Configs)),
	}

	// Create Docker client
	dockerClient, err := NewDockerClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	defer dockerClient.Close()

	// Ensure the Docker image exists
	fmt.Printf("Checking Docker image: %s\n", config.Image)
	if err := dockerClient.EnsureImage(ctx, config.Image); err != nil {
		return nil, fmt.Errorf("failed to ensure Docker image: %w", err)
	}

	// Create output directory
	if err := os.MkdirAll(config.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create a temporary directory for workspace
	tmpDir, err := os.MkdirTemp("", "ci-benchmark-matrix-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	fmt.Printf("\nMatrix Benchmark\n")
	fmt.Printf("================\n")
	fmt.Printf("Image:      %s\n", config.Image)
	fmt.Printf("Repository: %s\n", config.RepoURL)
	fmt.Printf("Command:    %s\n", config.Command)
	fmt.Printf("Runs:       %d per configuration\n", config.Runs)
	fmt.Printf("Configs:    %d configurations\n\n", len(config.Configs))

	// Run each configuration sequentially
	for i, resourceCfg := range config.Configs {
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("Configuration %d/%d: %s\n", i+1, len(config.Configs), resourceCfg.String())
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

		configResult := runSingleConfig(ctx, dockerClient, config, resourceCfg, binaryPath, tmpDir)
		result.Results = append(result.Results, configResult)

		if configResult.Success {
			fmt.Printf("\n✓ Configuration %d/%d completed successfully\n\n", i+1, len(config.Configs))
		} else {
			fmt.Printf("\n✗ Configuration %d/%d failed: %s\n\n", i+1, len(config.Configs), configResult.Error)
		}
	}

	return result, nil
}

// runSingleConfig runs the benchmark for a single CPU/RAM configuration
func runSingleConfig(
	ctx context.Context,
	dockerClient *DockerClient,
	config Config,
	resourceCfg ResourceConfig,
	binaryPath string,
	tmpDir string,
) ConfigResult {
	result := ConfigResult{
		Config:    resourceCfg,
		TotalRuns: config.Runs,
	}

	// Create a workspace directory for this configuration
	workspaceDir := filepath.Join(tmpDir, resourceCfg.DirName())
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		result.Error = fmt.Sprintf("failed to create workspace directory: %v", err)
		return result
	}

	// Create output directory for this configuration
	outputDir := filepath.Join(config.OutputDir, resourceCfg.DirName())
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		result.Error = fmt.Sprintf("failed to create output directory: %v", err)
		return result
	}

	fmt.Printf("  Starting container with %d CPUs, %d GB RAM...\n", resourceCfg.CPUs, resourceCfg.Memory)

	// Create container with resource limits
	container, err := dockerClient.CreateContainer(ctx, ContainerConfig{
		Image:     config.Image,
		CPUs:      resourceCfg.CPUs,
		Memory:    resourceCfg.Memory,
		MountPath: workspaceDir,
	})
	if err != nil {
		result.Error = fmt.Sprintf("failed to create container: %v", err)
		return result
	}

	// Ensure container is stopped and removed when done
	defer func() {
		fmt.Printf("  Stopping and removing container...\n")
		if err := container.Stop(ctx); err != nil {
			fmt.Printf("  Warning: failed to stop container: %v\n", err)
		}
	}()

	fmt.Printf("  Container started: %s\n", container.ID[:12])

	// Clone repository
	fmt.Printf("  Cloning repository: %s\n", config.RepoURL)
	cloneResult, err := container.ExecShell(ctx, fmt.Sprintf("git clone --depth 1 %s /workspace/repo", config.RepoURL), "/workspace")
	if err != nil {
		result.Error = fmt.Sprintf("failed to execute git clone: %v", err)
		return result
	}
	if cloneResult.ExitCode != 0 {
		result.Error = fmt.Sprintf("git clone failed (exit code %d): %s", cloneResult.ExitCode, cloneResult.Stderr)
		return result
	}
	fmt.Printf("  Repository cloned successfully\n")

	// Copy the ci-benchmark binary to the container
	fmt.Printf("  Copying ci-benchmark binary to container...\n")
	if err := container.CopyFileToContainer(ctx, binaryPath, "/workspace/ci-benchmark"); err != nil {
		result.Error = fmt.Sprintf("failed to copy binary to container: %v", err)
		return result
	}

	// Make the binary executable
	chmodResult, err := container.ExecShell(ctx, "chmod +x /workspace/ci-benchmark", "/workspace")
	if err != nil || chmodResult.ExitCode != 0 {
		result.Error = fmt.Sprintf("failed to make binary executable: %v", err)
		return result
	}

	// Construct benchmark command (prefix with repo name)
	repoName := config.RepoName()
	benchmarkName := fmt.Sprintf("%s_%s", repoName, resourceCfg.DirName())
	warmupFlag := ""
	if config.SkipWarmup {
		warmupFlag = "--no-warmup"
	}

	benchmarkCmd := fmt.Sprintf(
		"/workspace/ci-benchmark --runs %d --command %q --output-dir /workspace/results --name %s %s",
		config.Runs,
		config.Command,
		benchmarkName,
		warmupFlag,
	)

	fmt.Printf("  Running benchmark: %s\n", config.Command)
	fmt.Printf("  Number of runs: %d\n\n", config.Runs)

	// Create results directory in container
	mkdirResult, err := container.ExecShell(ctx, "mkdir -p /workspace/results", "/workspace")
	if err != nil || mkdirResult.ExitCode != 0 {
		result.Error = fmt.Sprintf("failed to create results directory: %v", err)
		return result
	}

	// Run the benchmark
	startTime := time.Now()
	benchResult, err := container.ExecShell(ctx, benchmarkCmd, "/workspace/repo")
	duration := time.Since(startTime)

	if err != nil {
		result.Error = fmt.Sprintf("failed to execute benchmark: %v", err)
		return result
	}

	// Print benchmark output
	if benchResult.Stdout != "" {
		fmt.Print(benchResult.Stdout)
	}
	if benchResult.Stderr != "" && benchResult.ExitCode != 0 {
		fmt.Printf("  Stderr: %s\n", benchResult.Stderr)
	}

	fmt.Printf("\n  Total time for configuration: %s\n", duration.Round(time.Second))

	// Copy results from container
	fmt.Printf("  Copying results from container...\n")
	if err := container.CopyDirFromContainer(ctx, "/workspace/results", outputDir); err != nil {
		result.Error = fmt.Sprintf("failed to copy results from container: %v", err)
		return result
	}

	// Parse the JSON results to extract statistics
	jsonPath := filepath.Join(outputDir, fmt.Sprintf("%s.json", benchmarkName))
	if err := parseResultsJSON(jsonPath, &result); err != nil {
		// Not a fatal error, just warn
		fmt.Printf("  Warning: failed to parse results JSON: %v\n", err)
		if benchResult.ExitCode != 0 {
			result.Error = fmt.Sprintf("benchmark failed (exit code %d)", benchResult.ExitCode)
			return result
		}
	}

	result.Success = true
	return result
}

// parseResultsJSON reads the benchmark JSON file and extracts statistics
func parseResultsJSON(jsonPath string, result *ConfigResult) error {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}

	var jsonResult struct {
		Summary struct {
			TotalRuns   int     `json:"totalRuns"`
			Successful  int     `json:"successful"`
			SuccessRate float64 `json:"successRate"`
		} `json:"summary"`
		Statistics struct {
			N      int     `json:"n"`
			Mean   float64 `json:"mean"`
			Median float64 `json:"median"`
			StdDev float64 `json:"stdDev"`
			Min    float64 `json:"min"`
			Max    float64 `json:"max"`
			P90    float64 `json:"p90"`
			P95    float64 `json:"p95"`
		} `json:"statistics"`
	}

	if err := json.Unmarshal(data, &jsonResult); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	result.TotalRuns = jsonResult.Summary.TotalRuns
	result.SuccessRuns = jsonResult.Summary.Successful
	result.SuccessRate = jsonResult.Summary.SuccessRate
	result.Mean = jsonResult.Statistics.Mean
	result.Median = jsonResult.Statistics.Median
	result.StdDev = jsonResult.Statistics.StdDev
	result.Min = jsonResult.Statistics.Min
	result.Max = jsonResult.Statistics.Max
	result.P90 = jsonResult.Statistics.P90
	result.P95 = jsonResult.Statistics.P95

	return nil
}

// BuildStaticBinary builds a static binary for Linux that can run in Docker containers
func BuildStaticBinary(outputPath string) error {
	fmt.Printf("Building static binary for Linux...\n")

	// Get the module root directory
	modRoot, err := getModuleRoot()
	if err != nil {
		return fmt.Errorf("failed to get module root: %w", err)
	}

	// Build command for static Linux binary
	cmd := exec.Command("go", "build",
		"-o", outputPath,
		"-ldflags", "-s -w",
		".",
	)
	cmd.Dir = modRoot
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS=linux",
		"GOARCH=amd64",
	)

	// Check if we're on ARM Mac and need to cross-compile
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		// Keep GOARCH=amd64 for x86_64 containers, or use arm64 for ARM containers
		// For now, default to amd64 as most Docker images are x86_64
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to build binary: %w\nOutput: %s", err, string(output))
	}

	fmt.Printf("Static binary built: %s\n", outputPath)
	return nil
}

// getModuleRoot finds the root directory of the Go module
func getModuleRoot() (string, error) {
	// Start from the executable's directory or current directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Walk up the directory tree looking for go.mod
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("could not find go.mod in parent directories")
}

// ExtractRepoName extracts the repository name from a URL
func ExtractRepoName(repoURL string) string {
	// Remove trailing .git if present
	repoURL = strings.TrimSuffix(repoURL, ".git")

	// Get the last part of the path
	parts := strings.Split(repoURL, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return "repo"
}
