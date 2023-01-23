/*
   Crossbar is a websocket relay
   Copyright (C) 2019 Timothy Drysdale <timothy.d.drysdale@gmail.com>

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as
   published by the Free Software Foundation, either version 3 of the
   License, or (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/
package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/practable/relay/internal/relay"
	log "github.com/sirupsen/logrus"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

/* configuration

bufferSize
muxBufferLength (for main message queue into the mux)
clientBufferLength (for each client's outgoing channel)

*/

// rootCmd represents the base command when called without any subcommands
var relayCmd = &cobra.Command{
	Use:   "relay",
	Short: "websocket relay with topics",
	Long: `Relay is a websocket relay with topics set by the URL path, 
and can handle binary and text messages. Set parameters with environment
variables, for example:

export RELAY_ACCESSPORT=10002
export RELAY_ACCESSFQDN=https://access.example.io
export RELAY_ALLOWNOBOOKINGID=true
export RELAY_RELAYPORT=10003
export RELAY_RELAYFQDN=wss://relay-access.example.io
export RELAY_SECRET=somesecret
export RELAY_DEVELOPMENT=true
export RELAY_PRUNEEVERY=5m #optional, advanced tuning parameter for deny list maintenance
relay relay 

`,
	Run: func(cmd *cobra.Command, args []string) {

		viper.SetEnvPrefix("RELAY")
		viper.AutomaticEnv()

		viper.SetDefault("accessport", 8082)
		viper.SetDefault("relayport", 8083)
		viper.SetDefault("pruneevery", "5m")

		accessPort := viper.GetInt("accessport")
		allowNoBookingID := viper.GetBool("allownobookingid")
		relayPort := viper.GetInt("relayport")
		development := viper.GetBool("development")
		secret := viper.GetString("secret")
		accessFQDN := viper.GetString("accessfqdn")
		relayFQDN := viper.GetString("relayfqdn")
		pruneEveryStr := viper.GetString("pruneevery")

		pruneEvery, err := time.ParseDuration(pruneEveryStr)

		if err != nil {
			fmt.Print("cannot parse duration in RELAY_PRUNEEVERY=" + pruneEveryStr)
			os.Exit(1)
		}

		if development {
			// development environment
			fmt.Println("Development mode - logging output to stdout")
			fmt.Printf("Access port [%d] for FQDN[%s]\nRelay port [%d] for FQDN[%s]\nNo booking ID allowed [%t]\n", accessPort, accessFQDN, relayPort, relayFQDN, allowNoBookingID)
			log.SetFormatter(&log.TextFormatter{})
			log.SetLevel(log.TraceLevel)
			log.SetOutput(os.Stdout)

		} else {

			//production environment
			log.SetFormatter(&log.JSONFormatter{})
			log.SetLevel(log.WarnLevel)

		}

		var wg sync.WaitGroup

		closed := make(chan struct{})

		c := make(chan os.Signal, 1)

		signal.Notify(c, os.Interrupt)

		go func() {
			for range c {
				close(closed)
				wg.Wait()
				os.Exit(0)
			}
		}()

		audience := accessFQDN //+ ":" + strconv.Itoa(accessPort)
		target := relayFQDN    //+ ":" + strconv.Itoa(relayPort)

		wg.Add(1)

		config := relay.Config{
			AccessPort:       accessPort,
			RelayPort:        relayPort,
			Audience:         audience,
			Secret:           secret,
			Target:           target,
			AllowNoBookingID: allowNoBookingID,
			PruneEvery:       pruneEvery,
		}

		go relay.Relay(closed, &wg, config) //accessPort, relayPort, audience, secret, target, allowNoBookingID)

		wg.Wait()

	},
}

func init() {
	rootCmd.AddCommand(relayCmd)
}
