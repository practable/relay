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
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/timdrysdale/relay/pkg/access"
	"github.com/timdrysdale/relay/pkg/relay"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var bufferSize int64
var host *url.URL
var audience, cfgFile, cpuprofile, listen, logFile, secret string
var development bool

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
export RELAY_RELAYPORT=10003
export RELAY_RELAYFQDN=wss://relay-access.example.io
export RELAY_SECRET=somesecret
export RELAY_DEVELOPMENT=true
shell relay
`,
	Run: func(cmd *cobra.Command, args []string) {

		viper.SetEnvPrefix("RELAY")
		viper.AutomaticEnv()

		viper.SetDefault("accessport", 8082)
		viper.SetDefault("relayport", 8083)

		accessPort := viper.GetInt("accessport")
		relayPort := viper.GetInt("relayport")
		development := viper.GetBool("development")
		secret := viper.GetString("secret")
		accessFQDN := viper.GetString("accessfqdn")
		relayFQDN := viper.GetString("relayfqdn")

		if development {
			// development environment
			fmt.Println("Development mode - logging output to stdout")
			fmt.Printf("Access port: %d for %s\nRelay port: %d for %s\n", accessPort, accessFQDN, relayPort, relayFQDN)
			log.SetFormatter(&log.TextFormatter{})
			log.SetLevel(log.TraceLevel)
			log.SetOutput(os.Stdout)

		} else {

			//production environment
			log.SetFormatter(&log.JSONFormatter{})
			log.SetLevel(log.WarnLevel)

			file, err := os.OpenFile("crossbar.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err == nil {
				log.SetOutput(file)
			} else {
				log.Info("Failed to log to file, using default stderr")
			}

		}

		var wg sync.WaitGroup

		closed := make(chan struct{})

		c := make(chan os.Signal, 1)

		signal.Notify(c, os.Interrupt)

		go func() {
			for _ = range c {
				close(closed)
				wg.Wait()
				os.Exit(0)
			}
		}()

		audience := accessFQDN + ":" + strconv.Itoa(accessPort)
		target := relayFQDN + ":" + strconv.Itoa(relayPort)

		wg.Add(1)

		go relay.Relay(closed, &wg, accessPort, relayPort, audience, secret, target, access.Options{})

		wg.Wait()

	},
}

func init() {
	rootCmd.AddCommand(relayCmd)
}
