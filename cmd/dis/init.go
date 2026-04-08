package main

import (
	"os"

	"github.com/candacelabs/dis/internal/scaffold"
	"github.com/spf13/cobra"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize dis in the current project",
	Long:  "Detect project type, generate .dis/Dockerfile and Justfile.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		return scaffold.Init(cwd, scaffold.InitOptions{Force: initForce})
	},
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite existing files")
}
