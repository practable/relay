/*
Copyright © 2021 Tim Drysdale <timothy.d.drysdale@gmail.com>

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
	"github.com/spf13/cobra"
)

// hostCmd represents the host command
var hostCmd = &cobra.Command{
	Use:   "switch",
	Short: "Set host rules based on GPIO",
	Long: `Watches a GPIO pin and updates the host rules when the pin changes stage. 
This is intended to provide a local control functionality for experiments e.g. via
a key switch. There are two rule sets, one is for the low state of the GPIO input, 
the other is for the high state. Another GPIO is used to indicate which rule set is
currently in force.  

export RELAY_SWITCH_PORT=8888
export RELAY_SWITCH_TOPIC_FEED=data
# connect to this topic to send the message
export RELAY_SWITCH_TOPIC_LOCAL=local
export RELAY_SWITCH_RULE_ID=1
export RELAY_SWITCH_DESTINATION=https://app.practable.io/some_instance/access/session/expt99-st-data
export RELAY_SWITCH_TOKEN=ey...
export RELAY_SWITCH_GPIO_INPUT==<some_pin>
export RELAY_SWITCH_GPIO_OUTPUT=<another_pin>
#connect feed to destination when GPIO input is true (false)
# else connect local to destination, which is where we're sending the message 
export RELAY_SWITCH_CONNECT_WHEN=true
export RELAY_SWITCH_MESSAGE=This experiment is under local control
export RELAY_SWITCH_MESSAGE_EVERY=1s

relay host`,
	Run: func(cmd *cobra.Command, args []string) {
		//todo add switch.Run() here
	},
}

func init() {
	rootCmd.AddCommand(hostCmd)

}
