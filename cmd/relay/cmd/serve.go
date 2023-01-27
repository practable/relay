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
	"net/http"
	_ "net/http/pprof" //ok in production https://medium.com/google-cloud/continuous-profiling-of-go-programs-96d4416af77b
	"os"
	"os/signal"
	"strconv"
	"strings"
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
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "websocket relay with topics",
	Long: `Relay is a websocket relay with topics set by the URL path, 
and can handle binary and text messages. Set parameters with environment
variables, for example:

export RELAY_ALLOW_NO_BOOKING_ID=true
export RELAY_AUDIENCE=https://example.org
export RELAY_LOG_LEVEL=warn
export RELAY_LOG_FORMAT=json
export RELAY_LOG_FILE=/var/log/relay/relay.log
export RELAY_PORT_ACCESS=3000
export RELAY_PORT_PROFILE=6061
export RELAY_PORT_RELAY=3001
export RELAY_PROFILE=true
export RELAY_SECRET=somesecret
export RELAY_TIDY_EVERY=5m 
export RELAY_URL=wss://example.io/relay 
relay serve 

Notes:
RELAY_URL tells access the FQDN for RELAY_PORT_RELAY; without it, access cannot redirect clients
RELAY_TIDY_EVERY is an optional tuning parameter that can safely be left at the default value

`,
	Run: func(cmd *cobra.Command, args []string) {

		viper.SetEnvPrefix("RELAY")
		viper.AutomaticEnv()

		viper.SetDefault("allow_no_booking_id", false) // default to most secure option; set true for backwards compatibility
		viper.SetDefault("audience", "")               //so we can check it's been provided
		viper.SetDefault("log_file", "/var/log/relay/relay.log")
		viper.SetDefault("log_format", "json")
		viper.SetDefault("log_level", "warn")
		viper.SetDefault("port_access", 3000)
		viper.SetDefault("port_relay", 3001)
		viper.SetDefault("profile", "true")
		viper.SetDefault("profile_port", 6061)
		viper.SetDefault("secret", "") //so we can check it's been provided
		viper.SetDefault("tidy_every", "5m")
		viper.SetDefault("url", "") //so we can check it's been provided

		allowNoBookingID := viper.GetBool("allow_no_booking_id")
		audience := viper.GetString("audience")
		logFile := viper.GetString("log_file")
		logFormat := viper.GetString("log_format")
		logLevel := viper.GetString("log_level")
		portAccess := viper.GetInt("port_access")
		portProfile := viper.GetInt("port_profile")
		portRelay := viper.GetInt("port_relay")
		profile := viper.GetBool("profile")
		secret := viper.GetString("secret")
		tidyEveryStr := viper.GetString("tidy_every")
		URL := viper.GetString("url")

		// Sanity checks
		ok := true

		if audience == "" {
			fmt.Println("You must set RELAY_AUDIENCE")
			ok = false
		}

		if secret == "" {
			fmt.Println("You must set RELAY_SECRET")
			ok = false
		}

		if URL == "" {
			fmt.Println("You must set RELAY_URL")
			ok = false
		}

		if !ok {
			os.Exit(1)
		}

		// parse durations

		tidyEvery, err := time.ParseDuration(tidyEveryStr)

		if err != nil {
			fmt.Print("cannot parse duration in RELAY_TIDY_EVERY=" + tidyEveryStr)
			os.Exit(1)
		}

		// set up logging
		switch strings.ToLower(logLevel) {
		case "trace":
			log.SetLevel(log.TraceLevel)
		case "debug":
			log.SetLevel(log.DebugLevel)
		case "info":
			log.SetLevel(log.InfoLevel)
		case "warn":
			log.SetLevel(log.WarnLevel)
		case "error":
			log.SetLevel(log.ErrorLevel)
		case "fatal":
			log.SetLevel(log.FatalLevel)
		case "panic":
			log.SetLevel(log.PanicLevel)
		default:
			fmt.Println("BOOK_LOG_LEVEL can be trace, debug, info, warn, error, fatal or panic but not " + logLevel)
			os.Exit(1)
		}

		switch strings.ToLower(logFormat) {
		case "json":
			log.SetFormatter(&log.JSONFormatter{})
		case "text":
			log.SetFormatter(&log.TextFormatter{})
		default:
			fmt.Println("BOOK_LOG_FORMAT can be json or text but not " + logLevel)
			os.Exit(1)
		}

		if strings.ToLower(logFile) == "stdout" {

			log.SetOutput(os.Stdout) //

		} else {

			file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err == nil {
				log.SetOutput(file)
			} else {
				log.Infof("Failed to log to %s, logging to default stderr", logFile)
			}
		}

		// Report useful info
		log.Infof("relay version: %s", versionString())
		log.Infof("Allow no booking ID: [%t]", allowNoBookingID)
		log.Infof("Audience: [%s]", audience)
		log.Infof("Log file: [%s]", logFile)
		log.Infof("Log format: [%s]", logFormat)
		log.Infof("Log level: [%s]", logLevel)
		log.Infof("Port for access: [%d]", portAccess)
		log.Infof("Port for profile: [%d]", portProfile)
		log.Infof("Port for relay: [%d]", portRelay)
		log.Infof("Profiling is on: [%t]", profile)
		log.Debugf("Secret: [%s...%s]", secret[:4], secret[len(secret)-4:])
		log.Infof("Tidy every: [%s]", tidyEvery)
		log.Infof("URL: [%s]", URL)

		// Optionally start the profiling server
		if profile {
			go func() {
				url := "localhost:" + strconv.Itoa(portProfile)
				err := http.ListenAndServe(url, nil)
				if err != nil {
					log.Errorf(err.Error())
				}
			}()
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

		wg.Add(1)

		config := relay.Config{
			AccessPort:       portAccess,
			RelayPort:        portRelay,
			Audience:         audience,
			Secret:           secret,
			Target:           URL,
			AllowNoBookingID: allowNoBookingID,
			PruneEvery:       tidyEvery,
		}

		go relay.Relay(closed, &wg, config) //accessPort, relayPort, audience, secret, target, allowNoBookingID)

		wg.Wait()

	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
