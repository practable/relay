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
	"os/signal"
	"sync"

	"github.com/ory/viper"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/practable/relay/internal/shellrelay"
)

// relayCmd represents the relay command
var relayCmd = &cobra.Command{
	Use:   "relay",
	Short: "shell relay connects shell clients to shell hosts",
	Long: `Set the operating paramters with environment variables, for example

export SHELLRELAY_ACCESSPORT=10001
export SHELLRELAY_ACCESSFQDN=https://access.example.io
export SHELLRELAY_RELAYPORT=10000
export SHELLRELAY_RELAYFQDN=wss://relay-access.example.io
export SHELLRELAY_SECRET=$your_secret
export SHELLRELAY_DEVELOPMENT=true
shell relay

It is expected that you will reverse proxy incoming connections (e.g. with nginx or apache). 
No provision is made for handling TLS in shell relay because this is more convenient
than separately managing certificates, especially when load balancing as may be required.
Note that load balancing takes place at the access phase, with the subsequent connection
being made by the associated relay. The FQDN of your relay access points must be distinct,
so that this affinity is maintained. The FQDN of the access points on the other hand, must
be the same, so that load balancing can be applied in your reverse proxy. All connections 
are individual so connections to a particular host can be simultaneously made in different 
shell relay instances, so long as the relay connection is made in the same instance which
handled the access (see comment above on setting target FQDN to be distinct, so that 
websocket connections are reverse proxied to the correct instance).
`,
	Run: func(cmd *cobra.Command, args []string) {

		viper.SetEnvPrefix("SHELLRELAY")
		viper.AutomaticEnv()

		viper.SetDefault("accessport", 8080)
		viper.SetDefault("relayport", 8081)

		accessPort := viper.GetInt("accessport")
		relayPort := viper.GetInt("relayport")
		development := viper.GetBool("development")
		secret := viper.GetString("secret")
		audience := viper.GetString("accessfqdn")
		target := viper.GetString("relayfqdn")

		if development {
			// development environment

			fmt.Println("Development mode - logging output to stdout")
			fmt.Printf("Access port: %d for %s\nRelay port: %d for %s\n", accessPort, audience, relayPort, target)
			log.SetReportCaller(true)
			log.SetFormatter(&log.TextFormatter{})
			log.SetLevel(log.InfoLevel)
			log.SetOutput(os.Stdout)

		} else {

			//production environment
			log.SetFormatter(&log.JSONFormatter{})
			log.SetLevel(log.WarnLevel)

		}

		// check inputs

		if secret == "" {
			fmt.Println("SHELLRELAY_SECRET not set")
			os.Exit(1)
		}
		if audience == "" {
			fmt.Println("SHELLRELAY_RELAYFQDN not set")
			os.Exit(1)
		}
		if target == "" {
			fmt.Println("SHELLRELAY_TARGETFQDN not set")
			os.Exit(1)
		}

		closed := make(chan struct{})

		var wg sync.WaitGroup

		c := make(chan os.Signal, 1)

		signal.Notify(c, os.Interrupt)

		go func() {
			for range c {
				close(closed)
				wg.Wait()
				os.Exit(0)
			}
		}()

		wg.Add(1)

		go shellrelay.Relay(closed, &wg, accessPort, relayPort, audience, secret, target)

		wg.Wait()

	},
}

func init() {
	rootCmd.AddCommand(relayCmd)
}
