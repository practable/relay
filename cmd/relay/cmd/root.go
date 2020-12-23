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
	"runtime/pprof"
	"sync"

	log "github.com/sirupsen/logrus"

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
var rootCmd = &cobra.Command{
	Use:   "crossbar",
	Short: "websocket relay with topics",
	Long: `Crossbar is a websocket relay with topics set by the URL path, 
and can handle binary and text messages.`,

	Run: func(cmd *cobra.Command, args []string) {

		addr := viper.GetString("listen")
		development := viper.GetBool("development")

		if development {
			// development environment
			fmt.Println("Development mode - logging output to stdout")
			fmt.Printf("Listening on %v\n", addr)
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

		if cpuprofile != "" {
			f, err := os.Create(cpuprofile)
			if err != nil {
				log.Fatal(err)
			}
			pprof.StartCPUProfile(f)
			defer pprof.StopCPUProfile()
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

		config := Config{
			Addr:     viper.GetString("listen"),
			Secret:   viper.GetString("secret"),
			Audience: viper.GetString("audience"),
		}

		wg.Add(1)

		go crossbar(config, closed, &wg)

		wg.Wait()

	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// init - specify args/flags
func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&listen, "listen", "127.0.0.1:8080", "<ip>:<port> to listen on (default is 127.0.0.1:8080)")
	rootCmd.PersistentFlags().Int64Var(&bufferSize, "buffer", 32768, "bufferSize in bytes (default is 32,768)")
	rootCmd.PersistentFlags().StringVar(&logFile, "log", "", "log file (default is STDOUT)")
	rootCmd.PersistentFlags().StringVar(&cpuprofile, "cpuprofile", "", "write cpu profile to file")
	rootCmd.PersistentFlags().BoolVar(&development, "dev", false, "development environment")
	rootCmd.PersistentFlags().StringVar(&secret, "secret", "", "set a secret to enable jwt authentication")
	rootCmd.PersistentFlags().StringVar(&audience, "https://localhost", "", "set the root FQDN we use to check the jwt audience (n.b. aud must contain the routing too)")
}

// initConfig - no config file; use ENV variables where available e.g. export CROSSBAR_LISTEN=127.0.0.1:8081
func initConfig() {
	viper.SetEnvPrefix("CROSSBAR")
	viper.AutomaticEnv() // read in environment variables that match
}
