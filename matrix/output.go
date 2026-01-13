package matrix

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// PrintSummaryTable prints a formatted summary table to the console
func PrintSummaryTable(result *MatrixResult) {
	fmt.Printf("\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Matrix Benchmark Summary\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	fmt.Printf("Image:      %s\n", result.Config.Image)
	fmt.Printf("Repository: %s\n", result.Config.RepoURL)
	fmt.Printf("Command:    %s\n", result.Config.Command)
	fmt.Printf("Runs:       %d per configuration\n\n", result.Config.Runs)

	// Create tabwriter for aligned output
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print header
	fmt.Fprintf(w, "CPUs\tRAM\tMean\tMedian\tStd Dev\tMin\tMax\tSuccess\n")
	fmt.Fprintf(w, "----\t---\t----\t------\t-------\t---\t---\t-------\n")

	// Print each result
	for _, r := range result.Results {
		if r.Success {
			fmt.Fprintf(w, "%d\t%d GB\t%s\t%s\t%s\t%s\t%s\t%.0f%%\n",
				r.Config.CPUs,
				r.Config.Memory,
				formatDuration(r.Mean),
				formatDuration(r.Median),
				formatDuration(r.StdDev),
				formatDuration(r.Min),
				formatDuration(r.Max),
				r.SuccessRate,
			)
		} else {
			fmt.Fprintf(w, "%d\t%d GB\tFAILED\t-\t-\t-\t-\t0%%\n",
				r.Config.CPUs,
				r.Config.Memory,
			)
		}
	}

	w.Flush()

	// Print failed configurations if any
	var failed []ConfigResult
	for _, r := range result.Results {
		if !r.Success {
			failed = append(failed, r)
		}
	}

	if len(failed) > 0 {
		fmt.Printf("\nFailed Configurations:\n")
		for _, r := range failed {
			fmt.Printf("  - %s: %s\n", r.Config.String(), r.Error)
		}
	}

	fmt.Printf("\n")
}

// SaveSummaryJSON saves the matrix results as JSON
func SaveSummaryJSON(result *MatrixResult, filename string) error {
	output := map[string]interface{}{
		"config": map[string]interface{}{
			"image":      result.Config.Image,
			"repoURL":    result.Config.RepoURL,
			"command":    result.Config.Command,
			"runs":       result.Config.Runs,
			"outputDir":  result.Config.OutputDir,
			"name":       result.Config.Name,
			"skipWarmup": result.Config.SkipWarmup,
		},
		"results": make([]map[string]interface{}, 0, len(result.Results)),
	}

	for _, r := range result.Results {
		resultMap := map[string]interface{}{
			"config": map[string]interface{}{
				"cpus":   r.Config.CPUs,
				"memory": r.Config.Memory,
			},
			"success":     r.Success,
			"error":       r.Error,
			"totalRuns":   r.TotalRuns,
			"successRuns": r.SuccessRuns,
			"successRate": r.SuccessRate,
		}

		if r.Success {
			resultMap["statistics"] = map[string]interface{}{
				"mean":   r.Mean,
				"median": r.Median,
				"stdDev": r.StdDev,
				"min":    r.Min,
				"max":    r.Max,
				"p90":    r.P90,
				"p95":    r.P95,
			}
		}

		output["results"] = append(output["results"].([]map[string]interface{}), resultMap)
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// SaveSummaryCSV saves the matrix results as CSV
func SaveSummaryCSV(result *MatrixResult, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"CPUs", "Memory (GB)", "Success",
		"Mean (s)", "Median (s)", "Std Dev (s)",
		"Min (s)", "Max (s)", "P90 (s)", "P95 (s)",
		"Success Rate (%)", "Total Runs", "Successful Runs", "Error",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Write each result
	for _, r := range result.Results {
		record := []string{
			fmt.Sprintf("%d", r.Config.CPUs),
			fmt.Sprintf("%d", r.Config.Memory),
			fmt.Sprintf("%t", r.Success),
			fmt.Sprintf("%.3f", r.Mean),
			fmt.Sprintf("%.3f", r.Median),
			fmt.Sprintf("%.3f", r.StdDev),
			fmt.Sprintf("%.3f", r.Min),
			fmt.Sprintf("%.3f", r.Max),
			fmt.Sprintf("%.3f", r.P90),
			fmt.Sprintf("%.3f", r.P95),
			fmt.Sprintf("%.1f", r.SuccessRate),
			fmt.Sprintf("%d", r.TotalRuns),
			fmt.Sprintf("%d", r.SuccessRuns),
			r.Error,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}

// SaveSummaryMarkdown saves the matrix results as Markdown
func SaveSummaryMarkdown(result *MatrixResult, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var md strings.Builder

	// Header
	md.WriteString("# Matrix Benchmark Report\n\n")
	md.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format(time.RFC1123)))

	// Configuration
	md.WriteString("## Configuration\n\n")
	md.WriteString(fmt.Sprintf("- **Docker Image:** `%s`\n", result.Config.Image))
	md.WriteString(fmt.Sprintf("- **Repository:** %s\n", result.Config.RepoURL))
	md.WriteString(fmt.Sprintf("- **Command:** `%s`\n", result.Config.Command))
	md.WriteString(fmt.Sprintf("- **Runs per Config:** %d\n", result.Config.Runs))
	if result.Config.SkipWarmup {
		md.WriteString("- **Warm-up:** Disabled\n")
	} else {
		md.WriteString("- **Warm-up:** Enabled (excluded from stats)\n")
	}
	md.WriteString("\n")

	// Summary table
	md.WriteString("## Results Summary\n\n")
	md.WriteString("| CPUs | RAM | Mean | Median | Std Dev | Min | Max | Success Rate |\n")
	md.WriteString("|------|-----|------|--------|---------|-----|-----|-------------|\n")

	for _, r := range result.Results {
		if r.Success {
			md.WriteString(fmt.Sprintf("| %d | %d GB | %s | %s | %s | %s | %s | %.0f%% |\n",
				r.Config.CPUs,
				r.Config.Memory,
				formatDuration(r.Mean),
				formatDuration(r.Median),
				formatDuration(r.StdDev),
				formatDuration(r.Min),
				formatDuration(r.Max),
				r.SuccessRate,
			))
		} else {
			md.WriteString(fmt.Sprintf("| %d | %d GB | FAILED | - | - | - | - | 0%% |\n",
				r.Config.CPUs,
				r.Config.Memory,
			))
		}
	}
	md.WriteString("\n")

	// Detailed statistics
	md.WriteString("## Detailed Statistics\n\n")
	for _, r := range result.Results {
		md.WriteString(fmt.Sprintf("### %s\n\n", r.Config.String()))

		if r.Success {
			md.WriteString("| Metric | Value |\n")
			md.WriteString("|--------|-------|\n")
			md.WriteString(fmt.Sprintf("| Mean | %s (%.3fs) |\n", formatDuration(r.Mean), r.Mean))
			md.WriteString(fmt.Sprintf("| Median | %s (%.3fs) |\n", formatDuration(r.Median), r.Median))
			md.WriteString(fmt.Sprintf("| Std Dev | %s (%.3fs) |\n", formatDuration(r.StdDev), r.StdDev))
			md.WriteString(fmt.Sprintf("| Min | %s (%.3fs) |\n", formatDuration(r.Min), r.Min))
			md.WriteString(fmt.Sprintf("| Max | %s (%.3fs) |\n", formatDuration(r.Max), r.Max))
			md.WriteString(fmt.Sprintf("| P90 | %s (%.3fs) |\n", formatDuration(r.P90), r.P90))
			md.WriteString(fmt.Sprintf("| P95 | %s (%.3fs) |\n", formatDuration(r.P95), r.P95))
			md.WriteString(fmt.Sprintf("| Success Rate | %.1f%% (%d/%d) |\n", r.SuccessRate, r.SuccessRuns, r.TotalRuns))
		} else {
			md.WriteString(fmt.Sprintf("**Status:** Failed\n\n"))
			md.WriteString(fmt.Sprintf("**Error:** %s\n", r.Error))
		}
		md.WriteString("\n")
	}

	// Failed configurations section if any
	var failed []ConfigResult
	for _, r := range result.Results {
		if !r.Success {
			failed = append(failed, r)
		}
	}

	if len(failed) > 0 {
		md.WriteString("## Failed Configurations\n\n")
		for _, r := range failed {
			md.WriteString(fmt.Sprintf("- **%s:** %s\n", r.Config.String(), r.Error))
		}
		md.WriteString("\n")
	}

	_, err = file.WriteString(md.String())
	return err
}

// formatDuration formats a duration in seconds to a human-readable string
func formatDuration(seconds float64) string {
	if seconds == 0 {
		return "0s"
	}

	duration := time.Duration(seconds * float64(time.Second))

	if duration >= time.Hour {
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		secs := int(duration.Seconds()) % 60
		return fmt.Sprintf("%dh%dm%ds", hours, minutes, secs)
	}

	if duration >= time.Minute {
		minutes := int(duration.Minutes())
		secs := int(duration.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", minutes, secs)
	}

	if duration >= time.Second {
		return fmt.Sprintf("%.1fs", seconds)
	}

	return fmt.Sprintf("%.0fms", seconds*1000)
}

// PrintBuildTimeGraph prints an ASCII bar chart of build time vs CPU count
func PrintBuildTimeGraph(result *MatrixResult) {
	// Filter successful results only
	var successful []ConfigResult
	for _, r := range result.Results {
		if r.Success {
			successful = append(successful, r)
		}
	}

	if len(successful) == 0 {
		return
	}

	fmt.Printf("Build Time vs CPU Count\n")
	fmt.Printf("=======================\n\n")

	// Find max mean time for scaling
	maxMean := 0.0
	for _, r := range successful {
		if r.Mean > maxMean {
			maxMean = r.Mean
		}
	}

	// Graph parameters
	graphWidth := 50 // Width of the bar area in characters

	// Print each bar
	for _, r := range successful {
		// Calculate bar width proportional to mean time
		barWidth := int((r.Mean / maxMean) * float64(graphWidth))
		if barWidth < 1 {
			barWidth = 1
		}

		// Create the bar
		bar := strings.Repeat("█", barWidth)

		// Format CPU label (right-aligned)
		cpuLabel := fmt.Sprintf("%2d CPU", r.Config.CPUs)

		// Format time label
		timeLabel := formatDuration(r.Mean)

		// Print the row
		fmt.Printf("%s │%s %s\n", cpuLabel, bar, timeLabel)
	}

	// Print x-axis
	fmt.Printf("       └%s\n", strings.Repeat("─", graphWidth+10))
	fmt.Printf("        0%s%s\n",
		strings.Repeat(" ", graphWidth/2-1),
		formatDuration(maxMean/2))
	fmt.Printf("        %s%s\n",
		strings.Repeat(" ", graphWidth-len(formatDuration(maxMean))+8),
		formatDuration(maxMean))

	fmt.Printf("\n")
}
