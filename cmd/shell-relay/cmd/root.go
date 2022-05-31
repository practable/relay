/*
Copyright © 2020 Tim Drysdale <timothy.d.drysdale@gmail.com>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var development bool
var logFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "shell",
	Short: "Shell is a set of services for relaying ssh connections",
	Long: `Shell is a set of services for relaying ssh connections.
Three services required in total for a single connection:
  host: the unattended remote machine 
  client: the attended local machine 
  relay: runs at a public IP address, shared out of band with the host and client

A relay can handle multiple connections. An administrator with multiple hosts to access, 
should start a separate client instance for each host they wish to connect to.
`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().BoolVar(&development, "dev", false, "development environment")
	rootCmd.PersistentFlags().StringVar(&logFile, "log", "", "log file (default is STDOUT)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
}
