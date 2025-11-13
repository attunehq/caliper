package benchmark

import (
	"math"
	"sort"
)

// Statistics holds calculated statistical metrics
type Statistics struct {
	N              int     // Number of successful runs
	Mean           float64 // Average duration in seconds
	Median         float64 // Median duration in seconds
	StdDev         float64 // Standard deviation in seconds
	Min            float64 // Minimum duration in seconds
	Max            float64 // Maximum duration in seconds
	P90            float64 // 90th percentile in seconds
	P95            float64 // 95th percentile in seconds
}

// CalculateStatistics computes all statistical metrics from duration data
func CalculateStatistics(durations []float64) Statistics {
	if len(durations) == 0 {
		return Statistics{}
	}

	stats := Statistics{
		N: len(durations),
	}

	// Sort durations for percentile calculations
	sorted := make([]float64, len(durations))
	copy(sorted, durations)
	sort.Float64s(sorted)

	// Calculate mean
	sum := 0.0
	for _, d := range durations {
		sum += d
	}
	stats.Mean = sum / float64(len(durations))

	// Calculate median
	stats.Median = percentile(sorted, 50)

	// Calculate standard deviation
	variance := 0.0
	for _, d := range durations {
		diff := d - stats.Mean
		variance += diff * diff
	}
	variance /= float64(len(durations))
	stats.StdDev = math.Sqrt(variance)

	// Min and Max
	stats.Min = sorted[0]
	stats.Max = sorted[len(sorted)-1]

	// Percentiles
	stats.P90 = percentile(sorted, 90)
	stats.P95 = percentile(sorted, 95)

	return stats
}

// percentile calculates the specified percentile from sorted data
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}

	if len(sorted) == 1 {
		return sorted[0]
	}

	// Use linear interpolation between closest ranks
	rank := (p / 100.0) * float64(len(sorted)-1)
	lowerIndex := int(math.Floor(rank))
	upperIndex := int(math.Ceil(rank))

	if lowerIndex == upperIndex {
		return sorted[lowerIndex]
	}

	// Interpolate between the two values
	weight := rank - float64(lowerIndex)
	return sorted[lowerIndex]*(1-weight) + sorted[upperIndex]*weight
}
