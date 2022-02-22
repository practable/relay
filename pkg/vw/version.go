package vw

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version represents the current version
var (
	Version   = "Version"
	BuildTime = "BuildTime"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of vw",
	Long:  `All software has versions. This is vw's`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s %s\n", Version, BuildTime)
	},
}
