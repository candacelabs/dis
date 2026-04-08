package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var downCmd = &cobra.Command{
	Use:   "down",
	Short: "Stop and remove the dev container",
	Long:  "Stop and remove the dev container. Idempotent — safe to run if already stopped.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := mustResolveConfig()
		mgr := mustCreateManager(cfg)
		if err := mgr.Down(cmd.Context()); err != nil {
			return err
		}
		fmt.Println("✅ Container stopped and removed.")
		return nil
	},
}
