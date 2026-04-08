package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "0.1.0"

var rootCmd = &cobra.Command{
	Use:     "dis",
	Short:   "Simple, plug-and-play dev containers",
	Long:    "dis — simple, plug-and-play dev containers.\nPart of Candace Labs.",
	Version: version,
}

func Execute() {
	// Override the default version template so `--version` prints "dis version X.Y.Z"
	rootCmd.SetVersionTemplate(fmt.Sprintf("dis version %s\n", version))

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(upCmd)
	rootCmd.AddCommand(downCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(shellCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(rebuildCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(nutsCmd)
	rootCmd.AddCommand(asterCmd)
}
