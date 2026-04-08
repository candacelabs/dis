package main

import (
	"github.com/spf13/cobra"
)

var rebuildCmd = &cobra.Command{
	Use:   "rebuild",
	Short: "Rebuild the dev container image from scratch",
	Long:  "Force-remove the current image and rebuild from the Dockerfile.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := mustResolveConfig()
		mgr := mustCreateManager(cfg)
		return mgr.Rebuild(cmd.Context())
	},
}
