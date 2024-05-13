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

// usbCmd represents the usb command
var usbCmd = &cobra.Command{
	Use:   "usb",
	Short: "Connect a usb port to a relay host",
	Long: `Connect a usb port to a relay host.

you must set the environment variables RELAY_USB_PORT, RELAY_USB_BAUD, RELAY_USB_TARGET

e.g. 
export RELAY_USB_PORT=/dev/ttyACM0
export RELAY_USB_BAUD=115200
export RELAY_USB_TARGET=ws://localhost:8888/ws/data
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("error: not implemented yet")
	},
}

func init() {
	rootCmd.AddCommand(usbCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// clientCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// clientCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
