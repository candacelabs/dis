package main

import (
	"github.com/candacelabs/dis/internal/proxy"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:                "run [command...]",
	Short:              "Run a command inside the dev container",
	Long:               "Run a command inside the dev container. All flags after 'run' are passed through.",
	DisableFlagParsing: true, // CRITICAL: pass all args through so `dis run make test -v` works
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := mustResolveConfig()
		mgr := mustCreateManager(cfg)
		err := proxy.RunCommand(cmd.Context(), mgr, args)
		if err != nil {
			handleExitCode(err)
			return err
		}
		return nil
	},
}
