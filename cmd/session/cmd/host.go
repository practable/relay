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
	"github.com/practable/relay/internal/vw"
)

// hostCmd represents the host command
var hostCmd = &cobra.Command{
	Use:   "host",
	Short: "Host video and data sessions",
	Long: `Establishes a connection to the session relay, 
listening for local tcp and websocket connections to forward, 
according to rules submitted by to the RESTful-ish API.  There are sensible defaults
for all parameters, and these can be overridden with environment variables. The defaults
value are as follows:

export RELAYHOST_PORT=8888
export RELAYHOST_LOGLEVEL=PANIC
export RELAYHOST_MAXBUFFERLENGTH=10
export RELAYHOST_CLIENTBUFFERLENGTH=5
export RELAYHOST_CLIENTTIMEOUTMS=1000
export RELAYHOST_HTTPWAITMS=5000
export RELAYHOST_HTTPSFLUSHMS=5
export RELAYHOST_HTTPTIMEOUTMS=1000
export RELAYHOST_CPUPROFULE=
export RELAYHOST_API=

session host`,
	Run: func(cmd *cobra.Command, args []string) {
		vw.Stream()
	},
}

func init() {
	rootCmd.AddCommand(hostCmd)

}
