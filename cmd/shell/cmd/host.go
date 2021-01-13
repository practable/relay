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
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"

	"github.com/ory/viper"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/timdrysdale/relay/pkg/shellhost"
)

// hostCmd represents the host command
var hostCmd = &cobra.Command{
	Use:   "host",
	Short: "shell host connects a local shell login service to shell relay",
	Long: `Set the operating paramters with environment variables, for example
export SHELLHOST_LOCALPORT=22
export SHELLHOST_RELAYSESSION=https://access.example.io/shell/abc123
export SHELLHOST_TOKEN=ey...<snip>
export SHELLHOST_DEVELOPMENT=true
shell host
`,

	Run: func(cmd *cobra.Command, args []string) {

		viper.SetEnvPrefix("SHELLHOST")
		viper.AutomaticEnv()

		viper.SetDefault("localport", 22)
		localPort := viper.GetInt("localport")
		relaySession := viper.GetString("relaysession")
		token := viper.GetString("token")
		development := viper.GetBool("development")

		if development {
			// development environment
			fmt.Println("Development mode - logging output to stdout")
			fmt.Printf("Local port: %d for %s with %d-byte token\n", localPort, relaySession, len(token))
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

		if relaySession == "" {
			fmt.Println("SHELLHOST_RELAYSESSION not set")
			os.Exit(1)
		}
		if token == "" {
			fmt.Println("SHELLHOST_TOKEN not set")
			os.Exit(1)
		}

		ctx, cancel := context.WithCancel(context.Background())

		c := make(chan os.Signal, 1)

		signal.Notify(c, os.Interrupt)

		go func() {
			for range c {
				cancel()
				<-ctx.Done()
				os.Exit(0)
			}
		}()

		local := "localhost:" + strconv.Itoa(localPort)

		go shellhost.Host(ctx, local, relaySession, token)

		<-ctx.Done() //unlikely to exit this way, but block all the same
		os.Exit(0)

	},
}

func init() {
	rootCmd.AddCommand(hostCmd)
}
