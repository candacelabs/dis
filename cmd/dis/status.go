package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show dev container status",
	Long:  "Show the current state of the dev container.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := mustResolveConfig()
		mgr := mustCreateManager(cfg)
		status, err := mgr.Status(cmd.Context())
		if err != nil {
			return err
		}

		fmt.Println("📦 dis status")
		fmt.Printf("   Container: %s\n", status.ContainerName)
		fmt.Printf("   State:     %s\n", status.State)

		if status.Image != "" {
			fmt.Printf("   Image:     %s\n", status.Image)
		}

		if len(status.Ports) > 0 {
			portStrs := make([]string, len(status.Ports))
			for i, p := range status.Ports {
				portStrs[i] = fmt.Sprintf("%d", p)
			}
			fmt.Printf("   Ports:     %s\n", strings.Join(portStrs, ", "))
		}

		if status.Uptime != "" {
			fmt.Printf("   Uptime:    %s\n", status.Uptime)
		}

		return nil
	},
}
