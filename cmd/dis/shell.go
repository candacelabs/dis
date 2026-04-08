package main

import (
	"github.com/candacelabs/dis/internal/proxy"
	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Open an interactive shell in the dev container",
	Long:  "Open an interactive shell session inside the dev container.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := mustResolveConfig()
		mgr := mustCreateManager(cfg)
		err := proxy.InteractiveShell(cmd.Context(), mgr, cfg.Shell)
		if err != nil {
			handleExitCode(err)
			return err
		}
		return nil
	},
}
