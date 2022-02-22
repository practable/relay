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
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/ory/viper"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/practable/relay/internal/book"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the booking server",
	Long: `Book is a REST API for booking experiments. Set parameters with environment
variables, for example:

export BOOK_PORT=4000
export BOOK_FQDN=https://book.practable.io
export BOOK_LOGINTIME=3600
export BOOK_SECRET=somesecret
book serve
`,
	Run: func(cmd *cobra.Command, args []string) {

		viper.SetEnvPrefix("BOOK")
		viper.AutomaticEnv()

		viper.SetDefault("port", 8080)
		viper.SetDefault("maxtime", 5400)
		viper.SetDefault("logfile", "/var/log/book/book.log")

		development := viper.GetBool("development")
		fqdn := viper.GetString("fqdn")
		port := viper.GetInt("port")
		secret := viper.GetString("secret")
		logintime := viper.GetInt("logintime")
		logfile := viper.GetString("logfile")

		if development {
			// development environment
			fmt.Println("Development mode - logging output to stdout")
			fmt.Printf("Listening port: %d\n", port)
			fmt.Printf("Secret=[%s]\n", secret)
			log.SetFormatter(&log.TextFormatter{})
			log.SetLevel(log.TraceLevel)
			log.SetOutput(os.Stdout)

		} else {

			//production environment
			log.SetFormatter(&log.JSONFormatter{})
			log.SetLevel(log.WarnLevel)

			file, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
			if err == nil {
				log.SetOutput(file)
			} else {
				log.Infof("Failed to log to %s, using default stderr", logfile)
			}
		}

		c := make(chan os.Signal, 1)

		signal.Notify(c, os.Interrupt)

		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			for range c {
				cancel()
				<-ctx.Done()
				os.Exit(0)
			}
		}()

		book.Book(ctx, port, int64(logintime), fqdn, secret)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
