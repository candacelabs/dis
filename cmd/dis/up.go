package main

import (
	"github.com/spf13/cobra"
)

var upCmd = &cobra.Command{
	Use:   "up",
	Short: "Start the dev container",
	Long:  "Build the image (if needed) and start the dev container.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := mustResolveConfig()
		mgr := mustCreateManager(cfg)
		return mgr.Up(cmd.Context())
	},
}
