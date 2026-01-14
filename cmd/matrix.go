package cmd

import (
	"github.com/spf13/cobra"
)

var matrixCmd = &cobra.Command{
	Use:   "matrix",
	Short: "Run benchmarks across multiple CPU/RAM configurations",
	Long: `Run benchmarks across multiple CPU/RAM configurations in Docker containers.

This command allows you to test how a command performs with different resource
allocations, helping you understand scaling characteristics and resource requirements.

Available subcommands:
  custom      Run benchmarks with arbitrary CPU:RAM configuration pairs
  sweep-cpu   Run benchmarks varying CPU count with fixed RAM
  sweep-ram   Run benchmarks varying RAM with fixed CPU count
  all         Run benchmarks across a full CPU x RAM grid`,
}

func init() {
	rootCmd.AddCommand(matrixCmd)
}
