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

	// Benchmark type
	if result.Config.Type != "" {
		md.WriteString(fmt.Sprintf("**Benchmark Type:** %s\n\n", result.Config.Type))
	}

	// Configuration
	md.WriteString("## Configuration\n\n")
	md.WriteString(fmt.Sprintf("- **Docker Image:** `%s`\n", result.Config.Image))
	md.WriteString(fmt.Sprintf("- **Repository:** %s\n", result.Config.RepoURL))
	md.WriteString(fmt.Sprintf("- **Command:** `%s`\n", result.Config.Command))
	md.WriteString(fmt.Sprintf("- **Runs per Config:** %d\n", result.Config.Runs))

	// Type-specific configuration
	switch result.Config.Type {
	case BenchmarkTypeSweepCPU:
		md.WriteString(fmt.Sprintf("- **Fixed RAM:** %d GB\n", result.Config.FixedRAM))
		md.WriteString(fmt.Sprintf("- **CPU Values Tested:** %s\n", formatIntList(result.Config.CPUList)))
	case BenchmarkTypeSweepRAM:
		md.WriteString(fmt.Sprintf("- **Fixed CPU:** %d\n", result.Config.FixedCPU))
		md.WriteString(fmt.Sprintf("- **RAM Values Tested:** %s GB\n", formatIntList(result.Config.RAMList)))
	case BenchmarkTypeAll:
		md.WriteString(fmt.Sprintf("- **CPU Values Tested:** %s\n", formatIntList(result.Config.CPUList)))
		md.WriteString(fmt.Sprintf("- **RAM Values Tested:** %s GB\n", formatIntList(result.Config.RAMList)))
	}

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

	// Add graphs section
	md.WriteString("## Graphs\n\n")
	graphStr := generateGraphsMarkdown(result)
	md.WriteString(graphStr)

	_, err = file.WriteString(md.String())
	return err
}

// generateGraphsMarkdown generates ASCII graphs as markdown code blocks
func generateGraphsMarkdown(result *MatrixResult) string {
	var sb strings.Builder

	switch result.Config.Type {
	case BenchmarkTypeAll:
		// Generate CPU sweep graphs (one per RAM value) and RAM sweep graphs (one per CPU value)
		cpuSet := make(map[int]bool)
		ramSet := make(map[int]bool)
		for _, r := range result.Results {
			cpuSet[r.Config.CPUs] = true
			ramSet[r.Config.Memory] = true
		}

		var cpus []int
		for cpu := range cpuSet {
			cpus = append(cpus, cpu)
		}
		sortInts(cpus)

		var rams []int
		for ram := range ramSet {
			rams = append(rams, ram)
		}
		sortInts(rams)

		// CPU sweep graphs
		for _, ram := range rams {
			graph := generateCPUSweepGraphString(result, ram)
			if graph != "" {
				sb.WriteString("```\n")
				sb.WriteString(graph)
				sb.WriteString("```\n\n")
			}
		}

		// RAM sweep graphs
		for _, cpu := range cpus {
			graph := generateRAMSweepGraphString(result, cpu)
			if graph != "" {
				sb.WriteString("```\n")
				sb.WriteString(graph)
				sb.WriteString("```\n\n")
			}
		}

	case BenchmarkTypeSweepCPU:
		graph := generateCPUSweepGraphString(result, result.Config.FixedRAM)
		if graph != "" {
			sb.WriteString("```\n")
			sb.WriteString(graph)
			sb.WriteString("```\n\n")
		}

	case BenchmarkTypeSweepRAM:
		graph := generateRAMSweepGraphString(result, result.Config.FixedCPU)
		if graph != "" {
			sb.WriteString("```\n")
			sb.WriteString(graph)
			sb.WriteString("```\n\n")
		}

	default:
		// Custom mode - show generic graph
		graph := generateGenericGraphString(result)
		if graph != "" {
			sb.WriteString("```\n")
			sb.WriteString(graph)
			sb.WriteString("```\n\n")
		}
	}

	return sb.String()
}

// generateCPUSweepGraphString generates a graph string for CPU sweep at fixed RAM
func generateCPUSweepGraphString(result *MatrixResult, fixedRAM int) string {
	var filtered []ConfigResult
	for _, r := range result.Results {
		if r.Success && r.Config.Memory == fixedRAM {
			filtered = append(filtered, r)
		}
	}

	if len(filtered) == 0 {
		return ""
	}

	title := fmt.Sprintf("Build Time vs CPU (%d GB RAM)", fixedRAM)
	return generateBarChartString(filtered, title, "cpu")
}

// generateRAMSweepGraphString generates a graph string for RAM sweep at fixed CPU
func generateRAMSweepGraphString(result *MatrixResult, fixedCPU int) string {
	var filtered []ConfigResult
	for _, r := range result.Results {
		if r.Success && r.Config.CPUs == fixedCPU {
			filtered = append(filtered, r)
		}
	}

	if len(filtered) == 0 {
		return ""
	}

	title := fmt.Sprintf("Build Time vs RAM (%d CPUs)", fixedCPU)
	return generateBarChartString(filtered, title, "ram")
}

// generateGenericGraphString generates a graph string for custom benchmark
func generateGenericGraphString(result *MatrixResult) string {
	var successful []ConfigResult
	for _, r := range result.Results {
		if r.Success {
			successful = append(successful, r)
		}
	}

	if len(successful) == 0 {
		return ""
	}

	return generateBarChartString(successful, "Build Time vs Configuration", "config")
}

// generateBarChartString generates a bar chart as a string
func generateBarChartString(results []ConfigResult, title string, labelType string) string {
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s\n", title))
	sb.WriteString(fmt.Sprintf("%s\n\n", strings.Repeat("=", len(title))))

	// Find max mean time for scaling
	maxMean := 0.0
	for _, r := range results {
		if r.Mean > maxMean {
			maxMean = r.Mean
		}
	}

	graphWidth := 50

	for _, r := range results {
		barWidth := int((r.Mean / maxMean) * float64(graphWidth))
		if barWidth < 1 {
			barWidth = 1
		}

		bar := strings.Repeat("█", barWidth)

		var label string
		switch labelType {
		case "cpu":
			label = fmt.Sprintf("%2d CPU", r.Config.CPUs)
		case "ram":
			label = fmt.Sprintf("%2d GB", r.Config.Memory)
		default:
			label = fmt.Sprintf("%2d CPU %2d GB", r.Config.CPUs, r.Config.Memory)
		}

		timeLabel := formatDuration(r.Mean)
		sb.WriteString(fmt.Sprintf("%s │%s %s\n", label, bar, timeLabel))
	}

	sb.WriteString(fmt.Sprintf("       └%s\n", strings.Repeat("─", graphWidth+10)))
	sb.WriteString(fmt.Sprintf("        0%s%s\n",
		strings.Repeat(" ", graphWidth/2-1),
		formatDuration(maxMean/2)))
	sb.WriteString(fmt.Sprintf("        %s%s\n",
		strings.Repeat(" ", graphWidth-len(formatDuration(maxMean))+8),
		formatDuration(maxMean)))
	sb.WriteString("\n")

	return sb.String()
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

// formatIntList formats a slice of ints as a comma-separated string
func formatIntList(ints []int) string {
	if len(ints) == 0 {
		return ""
	}
	strs := make([]string, len(ints))
	for i, v := range ints {
		strs[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(strs, ", ")
}

// PrintBuildTimeGraph prints an ASCII bar chart of build time based on benchmark type
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

	// Determine graph title based on benchmark type
	var title string
	switch result.Config.Type {
	case BenchmarkTypeSweepCPU:
		title = fmt.Sprintf("Build Time vs CPU (%d GB RAM)", result.Config.FixedRAM)
	case BenchmarkTypeSweepRAM:
		title = fmt.Sprintf("Build Time vs RAM (%d CPUs)", result.Config.FixedCPU)
	case BenchmarkTypeAll:
		// For "all" mode, use PrintAllGraphs instead
		title = "Build Time vs Configuration"
	default:
		title = "Build Time vs Configuration"
	}

	fmt.Printf("%s\n", title)
	fmt.Printf("%s\n\n", strings.Repeat("=", len(title)))

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

		// Format label based on benchmark type
		var label string
		switch result.Config.Type {
		case BenchmarkTypeSweepCPU:
			label = fmt.Sprintf("%2d CPU", r.Config.CPUs)
		case BenchmarkTypeSweepRAM:
			label = fmt.Sprintf("%2d GB", r.Config.Memory)
		default:
			label = fmt.Sprintf("%2d CPU %2d GB", r.Config.CPUs, r.Config.Memory)
		}

		// Format time label
		timeLabel := formatDuration(r.Mean)

		// Print the row
		fmt.Printf("%s │%s %s\n", label, bar, timeLabel)
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

// PrintAllGraphs prints multiple graphs for "all" benchmark mode
// One graph per RAM value showing CPU scaling, and one graph per CPU value showing RAM scaling
func PrintAllGraphs(result *MatrixResult) {
	if result.Config.Type != BenchmarkTypeAll {
		PrintBuildTimeGraph(result)
		return
	}

	// Get unique CPU and RAM values
	cpuSet := make(map[int]bool)
	ramSet := make(map[int]bool)
	for _, r := range result.Results {
		cpuSet[r.Config.CPUs] = true
		ramSet[r.Config.Memory] = true
	}

	// Convert to sorted slices
	var cpus []int
	for cpu := range cpuSet {
		cpus = append(cpus, cpu)
	}
	sortInts(cpus)

	var rams []int
	for ram := range ramSet {
		rams = append(rams, ram)
	}
	sortInts(rams)

	// Print CPU sweep graphs (one per RAM value)
	for _, ram := range rams {
		printCPUSweepGraph(result, ram)
	}

	// Print RAM sweep graphs (one per CPU value)
	for _, cpu := range cpus {
		printRAMSweepGraph(result, cpu)
	}
}

// printCPUSweepGraph prints a graph showing build time vs CPU for a fixed RAM value
func printCPUSweepGraph(result *MatrixResult, fixedRAM int) {
	// Filter results for this RAM value
	var filtered []ConfigResult
	for _, r := range result.Results {
		if r.Success && r.Config.Memory == fixedRAM {
			filtered = append(filtered, r)
		}
	}

	if len(filtered) == 0 {
		return
	}

	title := fmt.Sprintf("Build Time vs CPU (%d GB RAM)", fixedRAM)
	fmt.Printf("%s\n", title)
	fmt.Printf("%s\n\n", strings.Repeat("=", len(title)))

	printBarChart(filtered, "cpu")
}

// printRAMSweepGraph prints a graph showing build time vs RAM for a fixed CPU value
func printRAMSweepGraph(result *MatrixResult, fixedCPU int) {
	// Filter results for this CPU value
	var filtered []ConfigResult
	for _, r := range result.Results {
		if r.Success && r.Config.CPUs == fixedCPU {
			filtered = append(filtered, r)
		}
	}

	if len(filtered) == 0 {
		return
	}

	title := fmt.Sprintf("Build Time vs RAM (%d CPUs)", fixedCPU)
	fmt.Printf("%s\n", title)
	fmt.Printf("%s\n\n", strings.Repeat("=", len(title)))

	printBarChart(filtered, "ram")
}

// printBarChart prints a bar chart for the given results
// labelType is "cpu" or "ram" to determine axis labels
func printBarChart(results []ConfigResult, labelType string) {
	if len(results) == 0 {
		return
	}

	// Find max mean time for scaling
	maxMean := 0.0
	for _, r := range results {
		if r.Mean > maxMean {
			maxMean = r.Mean
		}
	}

	// Graph parameters
	graphWidth := 50

	// Print each bar
	for _, r := range results {
		barWidth := int((r.Mean / maxMean) * float64(graphWidth))
		if barWidth < 1 {
			barWidth = 1
		}

		bar := strings.Repeat("█", barWidth)

		var label string
		if labelType == "cpu" {
			label = fmt.Sprintf("%2d CPU", r.Config.CPUs)
		} else {
			label = fmt.Sprintf("%2d GB", r.Config.Memory)
		}

		timeLabel := formatDuration(r.Mean)
		fmt.Printf("%s │%s %s\n", label, bar, timeLabel)
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

// sortInts sorts a slice of ints in ascending order
func sortInts(a []int) {
	for i := 0; i < len(a)-1; i++ {
		for j := i + 1; j < len(a); j++ {
			if a[i] > a[j] {
				a[i], a[j] = a[j], a[i]
			}
		}
	}
}
