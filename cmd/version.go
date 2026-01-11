package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set via ldflags during build
var Version string

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of gowatch",
	Long:  `Print the version number of gowatch.`,
	Run: func(cmd *cobra.Command, args []string) {
		version := Version
		if version == "" {
			version = "dev"
		}
		fmt.Println("gowatch", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
