/*
Copyright © 2022 Tim Drysdale <timothy.d.drysdale@gmail.com>

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
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/practable/relay/internal/file"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// fileCmd represents the file command
var fileCmd = &cobra.Command{
	Use:   "file",
	Short: "Read/write data exchanged with relay from/to file",
	Long: `Read/write text data exchanged with relay from/to file.

You need to specify configuration information using environment variables, before
calling the executable. For example, to collect text data to stdout:

export RELAY_CLIENT_TOKEN=$(cat expt00-st-data.token)
export RELAY_CLIENT_SESSION=https://app.practable.io/ed0/access/session/expt00-st-data
relay client file

or to collect binary video stream to file:

export RELAY_CLIENT_TOKEN=$(cat expt00-st-video.token)
export RELAY_CLIENT_SESSION=https://app.practable.io/ed0/access/session/expt00-st-video
export RELAY_CLIENT_MODE=binary
export RELAY_CLIENT_FILE=./video/expt00.ts
relay client file


0. Authorisation
----------------

You will be connecting to a secured relay, so you need to specify the topic, and provide
a valid JWT token. If you can obtain them from your administrator, you'd specify them like 
this example (change server details etc to suit): 

export RELAY_CLIENT_TOKEN=$(cat expt00-st-data.token)
export RELAY_CLIENT_SESSION=https://app.practable.io/ed0/access/session/expt00-st-data

or if you have the relay secret, you can gerenate your own token when required, like this:

export RELAY_TOKEN_LIFETIME=86400
export RELAY_TOKEN_ROLE=client
export RELAY_TOKEN_SECRET=$(cat $HOME/secret/relay.pat)
export RELAY_TOKEN_TOPIC=expt00-st-data
export RELAY_TOKEN_CONNECTIONTYPE=session
export RELAY_TOKEN_AUDIENCE=https://app.practable.io/xx0/access
export RELAY_CLIENT_TOKEN=$(relay token)
export RELAY_CLIENT_SESSION=$ACCESSTOKEN_AUDIENCE/$ACCESSTOKEN_CONNECTIONTYPE/$ACCESSTOKEN_TOPIC

1. Binary vs Text Mode
----------------------

Relay client file supports two modes of operation, text (with timestamps) and binary (without).

You can see additional logging information on stdout by setting development mode. This might be useful
while building familiarity with the tool. It does not show the content of incoming messages
but reports their size. Do not use when streaming binary data to stdout as these timestamps will corrupt 
the binary data

Binary mode:
export RELAY_CLIENT_MODE=binary

Text mode (default):
export RELAY_CLIENT_MODE=text

If you do not set this variable, text mode is assumed.

2. Writing to File
------------------

Some long running log files might become large, indicating the need for log rotation.

If you are using logrotate(8), then send SIGHUP in your postscript. 
Do NOT restart the service, as you will lose any messages sent during 
the brief window that access is renegotiated (typically fast, but 
theoretically you could miss a message).

See github.com/practable/relay/scripts/examples/log-only.sh for a
demonstration of log rotation using sighup, which goes a bit like this:

# environment variable setup omitted for brevity
relay client file
export pid=$!
mv $logfile $logfile1
kill -SIGHUP pid
`,
	Run: func(cmd *cobra.Command, args []string) {

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("stacktrace from panic: \n" + string(debug.Stack()))
			}
		}()

		viper.SetEnvPrefix("RELAY_CLIENT")
		viper.AutomaticEnv()

		viper.SetDefault("interval", "1s")

		session := viper.GetString("session")
		token := viper.GetString("token")
		development := viper.GetBool("development")
		mode := viper.GetString("mode")
		filename := viper.GetString("file")

		binary := false
		mode = strings.TrimSpace(mode)
		mode = strings.ToLower(mode)

		if mode == "binary" {
			binary = true
		} else if !(mode == "text" || mode == "") {
			fmt.Println("RELAY_CLIENT_MODE must be 'text', 'binary' or unset (defaulting to text)")
			os.Exit(1)
		}

		if session == "" {
			fmt.Println("RELAY_CLIENT_SESSION not set")
			os.Exit(1)
		}

		if token == "" {
			fmt.Println("RELAY_CLIENT_TOKEN not set")
			os.Exit(1)
		}

		if development {
			// development environment
			fmt.Println("Development mode - logging output to stdout")
			fmt.Printf("Session: %s\nToken: %s\nFile: %s\nBinary: %s\n",
				session,
				token,
				filename,
				strconv.FormatBool(binary),
			)
			Formatter := new(log.TextFormatter)
			Formatter.TimestampFormat = time.RFC3339Nano
			log.SetFormatter(Formatter)
			log.SetLevel(log.TraceLevel)
			log.SetOutput(os.Stdout)

		} else {

			//production environment
			Formatter := new(log.JSONFormatter)
			Formatter.TimestampFormat = time.RFC3339Nano
			log.SetFormatter(Formatter)
			log.SetLevel(log.InfoLevel)

		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)

		go func() {
			for range c {
				cancel()
				log.Infof("Stopping normally due to Ctrl-C or SIGINT")
				os.Exit(0)
			}
		}()

		sighup := make(chan os.Signal, 1)
		signal.Notify(sighup, syscall.SIGHUP)

		connected := make(chan struct{})

		err := file.Run(ctx, sighup, connected, session, token, filename, binary)

		if err != nil {
			fmt.Println(err.Error())
			log.Errorf("Stopping due to error: %s", err.Error())
			os.Exit(1)
		}

		<-connected //wait for connection established

		log.Infof("Connected to session %s", session)

	},
}

func init() {
	clientCmd.AddCommand(fileCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// fileCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// fileCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
