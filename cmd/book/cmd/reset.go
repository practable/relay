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
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/ory/viper"
	"github.com/spf13/cobra"
	apiclient "github.com/practable/relay/internal/bc/client"
	"github.com/practable/relay/internal/bc/client/admin"
	"github.com/practable/relay/internal/bc/client/login"
)

// resetCmd represents the reset command
var resetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Delete all activities, pools and groups in the booking server.",
	Long: `Set the details of the server using environment variables. This
command should be used with extreme care, usually during testing only,
so manual confirmation is required.

export BOOKRESET_TOKEN=${your_admin_login_token}
export BOOKRESET_HOST=localhost
export_BOOKRESET_SCHEME=http
book reset 
`,
	Run: func(cmd *cobra.Command, args []string) {
		viper.SetEnvPrefix("BOOKRESET")
		viper.AutomaticEnv()
		viper.SetDefault("host", "localhost:8008")
		viper.SetDefault("scheme", "http")

		host := viper.GetString("host")
		scheme := viper.GetString("scheme")
		token := viper.GetString("token")

		if token == "" {
			fmt.Println("BOOK_ADMINTOKEN not set")
			os.Exit(1)
		}

		fmt.Printf("Do you really want to reset all activities,\npools and groups at %s://%s? [yes/NO]\n", scheme, host)
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		// convert CRLF to LF
		text = strings.Replace(text, "\n", "", -1)

		if strings.Compare("yes", strings.ToLower(text)) != 0 {
			fmt.Println("wise choice, aborting")
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

		err = bc.Admin.DeletePoolStore(
			admin.NewDeletePoolStoreParams().
				WithTimeout(timeout),
			auth)

		if !strings.HasPrefix(err.Error(), "[DELETE /admin/poolstore][404]") {
			fmt.Printf("Error: failed to reset book server because %s\n", err.Error())
			os.Exit(1)
		}

		fmt.Println("reset complete")
		os.Exit(0)

	},
}

func init() {
	rootCmd.AddCommand(resetCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// resetCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// resetCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
