/*
Copyright © 2022 Tim Drysdale <timothy.d.drysdale@gmail.com>

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

	"github.com/spf13/cobra"
)

// clientCmd represents the client command
/* Several interfaces are intended to be offered, via subcommands:
file
usb (not implemented yet)
ws-listen (not implemented yet)
http-listen (not implemented yet)
*/
var clientCmd = &cobra.Command{
	Use:   "client",
	Short: "Connect to a session relay",
	Long: `Connect to a session relay to exchange text data with a session host.

you must set the environment variables SESSION_CLIENT_SESSION and SESSION_CLIENT_TOKEN

e.g. 
export SESSION_CLIENT_SESSION=https://relay-access.practable.io/session/govn05-data
export SESSION_CLIENT_TOKEN=ey... #include complete JWT token
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("error: you must specify an interface e.g. file, or -h (help)")
	},
}

func init() {
	rootCmd.AddCommand(clientCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// clientCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// clientCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
