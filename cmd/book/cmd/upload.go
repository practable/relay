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
	"io/ioutil"
	"os"
	"time"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/ory/viper"
	"github.com/spf13/cobra"
	apiclient "github.com/practable/relay/internal/bc/client"
	"github.com/practable/relay/internal/bc/client/login"
	"github.com/practable/relay/internal/manifest"
	"gopkg.in/yaml.v2"
)

// uploadCmd represents the upload command
var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a manifest of activities, pools and groups to booking server",
	Long: `A yaml-format booking manifest is required.

export BOOKUPLOAD_TOKEN=${your_admin_login_token}
export BOOKUPLOAD_HOST=book.practable.io
export_BOOKUPLOAD_SCHEME=https
book upload your_manifest.yml
`,
	Run: func(cmd *cobra.Command, args []string) {
		viper.SetEnvPrefix("BOOKUPLOAD")
		viper.AutomaticEnv()
		viper.SetDefault("host", "book.practable.io")
		viper.SetDefault("scheme", "https")

		host := viper.GetString("host")
		scheme := viper.GetString("scheme")
		token := viper.GetString("token")

		if token == "" {
			fmt.Println("BOOKUPLOAD_TOKEN not set")
			os.Exit(1)
		}

		if len(os.Args) < 3 {
			fmt.Println("Specifiy manifest file as argument")
			os.Exit(1)
		}
		f := os.Args[2]
		mfest, err := ioutil.ReadFile(f)
		if err != nil {
			fmt.Printf("Error: failed to read manifest from file %s because %s\n", f, err.Error())
			os.Exit(1)
		}

		m := &manifest.Manifest{}

		err = yaml.Unmarshal(mfest, m)
		if err != nil {
			fmt.Printf("Error: failed to unmarshal manifest from file because %s\n", err.Error())
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

		status, err := manifest.UploadManifest(bc, auth, timeout, *m)
		if err != nil {
			fmt.Printf("Error: failed to upload manifest because %s\n", err.Error())
			os.Exit(1)
		}
		pretty, err := json.MarshalIndent(status, "", "\t")
		if err != nil {
			fmt.Printf("Error: failed to upmarshal status because %s\n", err.Error())
			os.Exit(1)
		}
		fmt.Println(string(pretty))
		os.Exit(0)

	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// uploadCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// uploadCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
