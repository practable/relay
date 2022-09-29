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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ory/viper"
	"github.com/practable/relay/internal/login"
	"github.com/spf13/cobra"
)

// tokenCmd represents the token command
var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "session token generates a new token for authenticating to book",
	Long: `Set the operating paramters with environment variables, for example

export BOOKTOKEN_LIFETIME=300
export BOOKTOKEN_SECRET=somesecret
export BOOKTOKEN_ADMIN=true
export BOOKTOKEN_AUDIENCE=https://book.example.io
export BOOKTOKEN_GROUPS="group1 group2 group3"
bearer=$(book token)

If you want to set a future NBF date, then specify the NBF in RFC3339 format
export BOOKTOKEN_NBF=2022-10-12T07:20:50Z
`,

	Run: func(cmd *cobra.Command, args []string) {

		viper.SetEnvPrefix("BOOKTOKEN")
		viper.AutomaticEnv()

		viper.SetDefault("lifetime", "600")
		viper.SetDefault("admin", "false")
		viper.SetDefault("audience", "https://book.practable.io")
		viper.SetDefault("groups", "everyone")
		viper.SetDefault("addscope", "")
		viper.SetDefault("nbf", "")

		lifetime := viper.GetInt64("lifetime")
		audience := viper.GetString("audience")
		secret := viper.GetString("secret")
		rawgroups := viper.GetString("groups")
		groups := strings.Split(rawgroups, " ")
		admin := viper.GetBool("admin")
		addscope := viper.GetString("addscope")
		startdate := viper.GetString("nbf")

		// check inputs

		if lifetime == 0 {
			fmt.Println("BOOKTOKEN_LIFETIME not set")
			os.Exit(1)
		}
		if secret == "" {
			fmt.Println("BOOKTOKEN_SECRET not set")
			os.Exit(1)
		}
		if rawgroups == "" {
			fmt.Println("BOOKTOKEN_GROUPS not set")
			os.Exit(1)
		}

		if audience == "" {
			fmt.Println("BOOKTOKEN_AUDIENCE not set")
			os.Exit(1)
		}

		iat := time.Now().Unix() - 1 // need immediately usable tokens for testing
		nbf := iat                   //update below if NBF is specified

		if startdate != "" {
			t, e := time.Parse(
				time.RFC3339,
				startdate)
			if e != nil {
				fmt.Printf("BOOKTOKEN_NBF time format error: %s\n", e.Error())
			}
			// ensure future date
			if t.After(time.Now()) {
				nbf = t.Unix()
			}
		}

		exp := nbf + lifetime

		var scopes []string

		if admin {
			scopes = []string{"login:admin"}
		} else {
			scopes = []string{"login:user"}
		}

		if addscope != "" {
			scopes = append(scopes, addscope)
		}

		token := login.NewToken(audience, groups, []string{}, scopes, iat, nbf, exp)
		bearer, err := login.Signed(token, secret)

		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println(bearer)
		os.Exit(0)

	},
}

func init() {
	rootCmd.AddCommand(tokenCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// tokenCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// tokenCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
