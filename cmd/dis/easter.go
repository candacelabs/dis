package main

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var nutsCmd = &cobra.Command{
	Use:    "nuts",
	Short:  "🥜",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🥜 dis nuts")
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

var asterCmd = &cobra.Command{
	Use:    "aster",
	Short:  "💥",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := mustResolveConfig()
		mgr := mustCreateManager(cfg)
		return mgr.Logs(cmd.Context(), false, "50")
	},
}
