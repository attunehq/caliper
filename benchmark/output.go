package benchmark

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"
)

// PrintConsole outputs the benchmark results to the console in a formatted table
func PrintConsole(result *Result) {
	fmt.Printf("\n")
	fmt.Printf("Benchmark Results\n")
	fmt.Printf("=================\n\n")

	// Summary information
	fmt.Printf("Command:        %s\n", result.Config.Command)
	fmt.Printf("Total Runs:     %d\n", result.Config.Runs)
	fmt.Printf("Successful:     %d\n", result.Stats.N)
	fmt.Printf("Failed:         %d\n", result.Config.Runs-result.Stats.N)
	fmt.Printf("Success Rate:   %.1f%%\n", result.SuccessRate)
	fmt.Printf("Total Duration: %v\n\n", result.TotalDuration.Round(time.Millisecond))

	// Statistics table
	if result.Stats.N > 0 {
		fmt.Printf("Statistics (successful runs only)\n")
		fmt.Printf("---------------------------------\n\n")

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "Metric\tValue\n")
		fmt.Fprintf(w, "------\t-----\n")
		fmt.Fprintf(w, "N\t%d\n", result.Stats.N)
		fmt.Fprintf(w, "Mean\t%s\n", formatDuration(result.Stats.Mean))
		fmt.Fprintf(w, "Median\t%s\n", formatDuration(result.Stats.Median))
		fmt.Fprintf(w, "Std Dev\t%s\n", formatDuration(result.Stats.StdDev))
		fmt.Fprintf(w, "Min\t%s\n", formatDuration(result.Stats.Min))
		fmt.Fprintf(w, "Max\t%s\n", formatDuration(result.Stats.Max))
		fmt.Fprintf(w, "P90\t%s\n", formatDuration(result.Stats.P90))
		fmt.Fprintf(w, "P95\t%s\n", formatDuration(result.Stats.P95))
		w.Flush()
	} else {
		fmt.Printf("No successful runs to calculate statistics.\n")
	}
}

// SaveJSON saves the benchmark results as JSON
func SaveJSON(result *Result, filename string) error {
	// Create a serializable version of the result
	output := map[string]interface{}{
		"config": map[string]interface{}{
			"command":    result.Config.Command,
			"runs":       result.Config.Runs,
			"name":       result.Config.Name,
			"outputDir":  result.Config.OutputDir,
		},
		"summary": map[string]interface{}{
			"totalRuns":     result.Config.Runs,
			"successful":    result.Stats.N,
			"failed":        result.Config.Runs - result.Stats.N,
			"successRate":   result.SuccessRate,
			"startTime":     result.StartTime.Format(time.RFC3339),
			"endTime":       result.EndTime.Format(time.RFC3339),
			"totalDuration": result.TotalDuration.Seconds(),
		},
		"statistics": map[string]interface{}{
			"n":      result.Stats.N,
			"mean":   result.Stats.Mean,
			"median": result.Stats.Median,
			"stdDev": result.Stats.StdDev,
			"min":    result.Stats.Min,
			"max":    result.Stats.Max,
			"p90":    result.Stats.P90,
			"p95":    result.Stats.P95,
		},
		"runs": result.Runs,
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

// SaveCSV saves the benchmark results as CSV
func SaveCSV(result *Result, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"Run", "Success", "Duration (seconds)", "Error"}); err != nil {
		return err
	}

	// Write individual run results
	for _, run := range result.Runs {
		record := []string{
			fmt.Sprintf("%d", run.RunNumber),
			fmt.Sprintf("%t", run.Success),
			fmt.Sprintf("%.6f", run.Duration.Seconds()),
			run.Error,
		}
		if err := writer.Write(record); err != nil {
			return err
		}
	}

	// Write summary statistics
	writer.Write([]string{})
	writer.Write([]string{"Summary Statistics"})
	writer.Write([]string{"Metric", "Value"})
	writer.Write([]string{"N", fmt.Sprintf("%d", result.Stats.N)})
	writer.Write([]string{"Mean (seconds)", fmt.Sprintf("%.6f", result.Stats.Mean)})
	writer.Write([]string{"Median (seconds)", fmt.Sprintf("%.6f", result.Stats.Median)})
	writer.Write([]string{"Std Dev (seconds)", fmt.Sprintf("%.6f", result.Stats.StdDev)})
	writer.Write([]string{"Min (seconds)", fmt.Sprintf("%.6f", result.Stats.Min)})
	writer.Write([]string{"Max (seconds)", fmt.Sprintf("%.6f", result.Stats.Max)})
	writer.Write([]string{"P90 (seconds)", fmt.Sprintf("%.6f", result.Stats.P90)})
	writer.Write([]string{"P95 (seconds)", fmt.Sprintf("%.6f", result.Stats.P95)})
	writer.Write([]string{"Success Rate (%)", fmt.Sprintf("%.1f", result.SuccessRate)})

	return nil
}

// SaveMarkdown saves the benchmark results as a Markdown report
func SaveMarkdown(result *Result, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var md strings.Builder

	// Header
	md.WriteString("# CI Benchmark Report\n\n")
	md.WriteString(fmt.Sprintf("**Generated:** %s\n\n", result.EndTime.Format(time.RFC1123)))

	// Configuration
	md.WriteString("## Configuration\n\n")
	md.WriteString(fmt.Sprintf("- **Command:** `%s`\n", result.Config.Command))
	md.WriteString(fmt.Sprintf("- **Benchmark Name:** %s\n", result.Config.Name))
	md.WriteString(fmt.Sprintf("- **Total Runs:** %d\n", result.Config.Runs))
	md.WriteString(fmt.Sprintf("- **Start Time:** %s\n", result.StartTime.Format(time.RFC1123)))
	md.WriteString(fmt.Sprintf("- **End Time:** %s\n", result.EndTime.Format(time.RFC1123)))
	md.WriteString(fmt.Sprintf("- **Total Duration:** %s\n\n", result.TotalDuration.Round(time.Millisecond)))

	// Summary
	md.WriteString("## Summary\n\n")
	md.WriteString(fmt.Sprintf("- **Successful Runs:** %d\n", result.Stats.N))
	md.WriteString(fmt.Sprintf("- **Failed Runs:** %d\n", result.Config.Runs-result.Stats.N))
	md.WriteString(fmt.Sprintf("- **Success Rate:** %.1f%%\n\n", result.SuccessRate))

	// Statistics
	if result.Stats.N > 0 {
		md.WriteString("## Statistics\n\n")
		md.WriteString("Statistics calculated from successful runs only:\n\n")
		md.WriteString("| Metric | Value |\n")
		md.WriteString("|--------|-------|\n")
		md.WriteString(fmt.Sprintf("| N | %d |\n", result.Stats.N))
		md.WriteString(fmt.Sprintf("| Mean | %s |\n", formatDuration(result.Stats.Mean)))
		md.WriteString(fmt.Sprintf("| Median | %s |\n", formatDuration(result.Stats.Median)))
		md.WriteString(fmt.Sprintf("| Std Dev | %s |\n", formatDuration(result.Stats.StdDev)))
		md.WriteString(fmt.Sprintf("| Min | %s |\n", formatDuration(result.Stats.Min)))
		md.WriteString(fmt.Sprintf("| Max | %s |\n", formatDuration(result.Stats.Max)))
		md.WriteString(fmt.Sprintf("| P90 | %s |\n", formatDuration(result.Stats.P90)))
		md.WriteString(fmt.Sprintf("| P95 | %s |\n", formatDuration(result.Stats.P95)))
		md.WriteString("\n")
	}

	// Individual runs
	md.WriteString("## Individual Runs\n\n")
	md.WriteString("| Run | Status | Duration | Error |\n")
	md.WriteString("|-----|--------|----------|-------|\n")
	for _, run := range result.Runs {
		status := "✓"
		if !run.Success {
			status = "✗"
		}
		errorMsg := ""
		if run.Error != "" {
			errorMsg = run.Error
		}
		md.WriteString(fmt.Sprintf("| %d | %s | %s | %s |\n",
			run.RunNumber,
			status,
			run.Duration.Round(time.Millisecond),
			errorMsg))
	}

	_, err = file.WriteString(md.String())
	return err
}

// formatDuration formats a duration in seconds to a human-readable string
func formatDuration(seconds float64) string {
	duration := time.Duration(seconds * float64(time.Second))

	// Round to milliseconds for readability
	duration = duration.Round(time.Millisecond)

	if duration >= time.Minute {
		return fmt.Sprintf("%s (%.3fs)", duration, seconds)
	}

	return fmt.Sprintf("%s (%.3fs)", duration, seconds)
}
