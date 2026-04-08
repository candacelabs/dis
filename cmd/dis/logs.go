package main

import (
	"github.com/spf13/cobra"
)

var (
	logsFollow bool
	logsTail   string
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show dev container logs",
	Long:  "Stream logs from the dev container.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := mustResolveConfig()
		mgr := mustCreateManager(cfg)
		return mgr.Logs(cmd.Context(), logsFollow, logsTail)
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Follow log output")
	logsCmd.Flags().StringVar(&logsTail, "tail", "100", "Number of lines to show from the end of the logs")
}
