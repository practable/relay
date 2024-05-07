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
	"github.com/practable/relay/internal/vw"
	"github.com/spf13/cobra"
)

// hostCmd represents the host command
var hostCmd = &cobra.Command{
	Use:   "host",
	Short: "Host video and data sessions",
	Long: `Establishes a connection to the session relay, 
listening for local tcp and websocket connections to forward, 
according to rules submitted by to the RESTful-ish API. 

The configurable options can be adjusted by exporting the following variables

	VW_PORT                (default:"8888")
	VW_LOGLEVEL            (default:"PANIC")
	VW_MUXBUFFERLENGTH     (default:"10")
	VW_CLIENTBUFFERLENGTH  (default:"5")
	VW_CLIENTTIMEOUTMS     (default:"1000")
	VW_HTTPWAITMS          (default:"5000")
	VW_HTTPFLUSHMS         (default:"5")
	VW_HTTPTIMEOUTMS       (default:"1000")
	VW_CPUPROFILE          (default:"")
	VW_API                 (default:"")
 
relay host`,
	Run: func(cmd *cobra.Command, args []string) {
		vw.Stream()
	},
}

func init() {
	rootCmd.AddCommand(hostCmd)

}
