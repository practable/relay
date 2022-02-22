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
	"github.com/spf13/cobra"
	apiclient "github.com/practable/relay/internal/bc/client"
	"github.com/practable/relay/internal/bc/client/admin"
	"github.com/practable/relay/internal/bc/client/login"
)

// getstatusCmd represents the getstatus command
var getstatusCmd = &cobra.Command{
	Use:   "getstatus",
	Short: "Get the lock status and message of the day",
	Long: `Set server details with environment variables. F
For example:

export BOOKSTATUS_HOST=localhost:4000
export BOOKSTATUS_SCHEME=http
export BOOKSTATUS_TOKEN=$secret
book getstatus 
`,
	Run: func(cmd *cobra.Command, args []string) {

		viper.SetEnvPrefix("BOOKSTATUS")
		viper.AutomaticEnv()
		viper.SetDefault("host", "book.practable.io")
		viper.SetDefault("scheme", "https")

		host := viper.GetString("host")
		scheme := viper.GetString("scheme")
		token := viper.GetString("token")

		if token == "" {
			fmt.Println("BOOKSTATUS_TOKEN not set")
			os.Exit(1)
		}

		cfg := apiclient.DefaultTransportConfig().WithHost(host).WithSchemes([]string{scheme})
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

		sparams := admin.NewGetStoreStatusParams().WithTimeout(timeout)
		status, err := bc.Admin.GetStoreStatus(sparams, auth)
		if err != nil {
			fmt.Printf("Error: failed to log in because %s\n", err.Error())
			os.Exit(1)
		}

		pretty, err := json.MarshalIndent(status.Payload, "", "\t")
		if err != nil {
			fmt.Printf("Error: failed to getstatus because %s\n", err.Error())
			os.Exit(1)
		}
		fmt.Println(string(pretty))
		os.Exit(0)

	},
}

func init() {
	rootCmd.AddCommand(getstatusCmd)
}
