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
	"encoding/json"
	"fmt"
	"os"
	"time"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/ory/viper"
	apiclient "github.com/practable/relay/internal/bc/client"
	"github.com/practable/relay/internal/bc/client/admin"
	"github.com/practable/relay/internal/bc/client/login"
	"github.com/spf13/cobra"
)

// setstatusCmd represents the setstatus command
var setstatusCmd = &cobra.Command{
	Use:   "setstatus",
	Short: "Set the lock status and message of the day",
	Long: `Set server details with environment variables 
and pass settings as arguments. 
For example:
export BOOKSTATUS_BASE=/book/api/v1
export BOOKSTATUS_HOST=core.prac.io
export BOOKSTATUS_SCHEME=https
export BOOKSTATUS_TOKEN=$secret
book setstatus unlock "Bookings are open"
`,
	Run: func(cmd *cobra.Command, args []string) {

		viper.SetEnvPrefix("BOOKSTATUS")
		viper.AutomaticEnv()
		viper.SetDefault("host", "book.practable.io")
		viper.SetDefault("scheme", "https")
		viper.SetDefault("base", "/api/v1")

		base := viper.GetString("base")
		host := viper.GetString("host")
		scheme := viper.GetString("scheme")
		token := viper.GetString("token")

		if token == "" {
			fmt.Println("BOOKSTATUS_TOKEN not set")
			os.Exit(1)
		}

		if len(os.Args) < 4 {
			fmt.Println("usage: book setstatus [lock/unlock] message")
			os.Exit(1)
		}

		lockword := os.Args[2]
		lock := false

		if lockword == "lock" {
			lock = true
		} else if lockword != "unlock" {
			fmt.Printf("lock status of %s not understood\n", lockword)
			fmt.Println("usage: book setstatus [lock/unlock] message")
			os.Exit(1)
		}

		message := os.Args[3]

		cfg := apiclient.DefaultTransportConfig().WithHost(host).WithSchemes([]string{scheme}).WithBasePath(base)
		loginAuth := httptransport.APIKeyAuth("Authorization", "header", token)
		bc := apiclient.NewHTTPClientWithConfig(nil, cfg)
		timeout := 10 * time.Second
		params := login.NewLoginParams().WithTimeout(timeout)
		resp, err := bc.Login.Login(params, loginAuth)
		if err != nil {
			fmt.Printf("Error: failed to log in because %s\n", err.Error())
			os.Exit(1)
		}

		auth := httptransport.APIKeyAuth("Authorization", "header", *resp.GetPayload().Token)

		slparams := admin.NewSetLockParams().WithTimeout(timeout).WithLock(lock).WithMsg(&message)
		status, err := bc.Admin.SetLock(slparams, auth)
		if err != nil {
			fmt.Printf("Error: failed to log in because %s\n", err.Error())
			os.Exit(1)
		}

		pretty, err := json.MarshalIndent(status.Payload, "", "\t")
		if err != nil {
			fmt.Printf("Error: failed to setstatus because %s\n", err.Error())
			os.Exit(1)
		}
		fmt.Println(string(pretty))
		os.Exit(0)

	},
}

func init() {
	rootCmd.AddCommand(setstatusCmd)
}
