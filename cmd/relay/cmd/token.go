/*
Copyright © 2020 Tim Drysdale <timothy.d.drysdale@gmail.com>

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
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/ory/viper"
	"github.com/practable/relay/internal/permission"
	"github.com/spf13/cobra"
)

// hostCmd represents the host command
var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "relay token generates a new token for authenticating to a relay",
	Long: `Set the operating paramters with environment variables, for example

export REALY_TOKEN_LIFETIME=3600
export REALY_TOKEN_READ=true
export REALY_TOKEN_WRITE=true
export REALY_TOKEN_SECRET=somesecret
export REALY_TOKEN_TOPIC=123
export REALY_TOKEN_AUDIENCE=https://relay-access.example.io
bearer=$(relay token)
`,

	Run: func(cmd *cobra.Command, args []string) {

		viper.SetEnvPrefix("RELAY_TOKEN")
		viper.AutomaticEnv()

		viper.SetDefault("connectionType", "session")
		viper.SetDefault("read", "true")
		viper.SetDefault("write", "true")

		lifetime := viper.GetInt64("lifetime")
		audience := viper.GetString("audience")
		secret := viper.GetString("secret")
		topic := viper.GetString("topic")
		connectionType := viper.GetString("connectionType")
		read := viper.GetBool("read")
		write := viper.GetBool("write")

		// check inputs

		if lifetime == 0 {
			fmt.Println("ACCESSTOKEN_LIFETIME not set")
			os.Exit(1)
		}
		if secret == "" {
			fmt.Println("ACCESSTOKEN_SECRET not set")
			os.Exit(1)
		}
		if topic == "" {
			fmt.Println("ACCESSTOKEN_TOPIC not set")
			os.Exit(1)
		}

		if connectionType == "" {
			fmt.Println("ACCESSTOKEN_CONNECTIONTYPE not set")
			os.Exit(1)
		}
		if audience == "" {
			fmt.Println("ACCESSTOKEN_AUDIENCE not set")
			os.Exit(1)
		}

		var scopes []string

		if write {
			scopes = append(scopes, "write")
		}

		if read {
			scopes = append(scopes, "read")
		}

		if !read && !write {
			fmt.Println("Neither read nor write scope, or both: no point in connecting.")
			os.Exit(1)
		}

		iat := time.Now().Unix() - 1 //ensure immediately usable
		nbf := iat
		exp := iat + lifetime

		var claims permission.Token
		claims.IssuedAt = jwt.NewNumericDate(time.Unix(iat, 0))
		claims.NotBefore = jwt.NewNumericDate(time.Unix(nbf, 0))
		claims.ExpiresAt = jwt.NewNumericDate(time.Unix(exp, 0))
		claims.Audience = jwt.ClaimStrings{audience}
		claims.Topic = topic
		claims.ConnectionType = connectionType // e.g. session
		claims.Scopes = scopes
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		bearer, err := token.SignedString([]byte(secret))

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

}
