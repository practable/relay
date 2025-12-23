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
	_ "net/http/pprof" //ok in production https://medium.com/google-cloud/continuous-profiling-of-go-programs-96d4416af77b
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	"github.com/practable/relay/internal/monitor"
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
var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "monitor for websocket relay",
	Long: `Monitor triggers a shell command when the latency of live canary connection to a relay serve instance exceeds a threshold.
 Set parameters with environment
variables, for example:

# these env var are as for the relay serve instance being monitored
export RELAY_MONITOR_AUDIENCE=https://app.practable.io/ed0/access
export RELAY_MONITOR_SECRET=somesecret

# these env var are specific to the monitor
export RELAY_MONITOR_LOG_LEVEL=warn
export RELAY_MONITOR_LOG_FORMAT=json
export RELAY_MONITOR_LOG_FILE=/var/log/relay/monitor.log
export RELAY_MONITOR_THRESHOLD=100ms
export RELAY_MONITOR_INTERVAL=1s
export RELAY_MONITOR_NO_RETRIGGER_WITHIN=60s
export RELAY_MONITOR_TRIGGER_AFTER_MISSES=3
export RELAY_MONITOR_TOPIC=canary-st-data
# in production, this might be a script that sends an alert
# or kills the relay process for systemd to restart it
export RELAY_MONITOR_COMMAND="echo 'latency exceeded'"
relay monitor 

Notes on configuration of the relay serve instance being monitored:

1/ A typical FQDN for a canary channel might be:

https://app.practable.io/ed0/access/session/canary-st-data

which can be constructed as follows
${RELAY_AUDIENCE}/session/${RELAY_MONITOR_TOPIC}

2/ sudo pkill 


`,
	Run: func(cmd *cobra.Command, args []string) {

		//runtime.SetBlockProfileRate(1) // https://pkg.go.dev/runtime#SetBlockProfileRate

		viper.SetEnvPrefix("RELAY_MONITOR")
		viper.AutomaticEnv()

		// set blank defaults to ensure we can check critical parameters
		// of which instance we are monitoring are explicitly set
		viper.SetDefault("audience", "")
		viper.SetDefault("secret", "")

		// set sensible defaults for logging to make configuration easier
		viper.SetDefault("log_level", "warn")
		viper.SetDefault("log_format", "json")
		viper.SetDefault("log_file", "") //blank log_file will log to stdout

		// set sensible defaults for monitor operation
		viper.SetDefault("threshold", "200ms")
		viper.SetDefault("interval", "1s")
		viper.SetDefault("reconnect_every", "24h")
		viper.SetDefault("trigger_after_misses", 3)
		viper.SetDefault("no_retrigger_within", "60s") //give relay time to restart and on-board connections
		viper.SetDefault("topic", "canary-st-data")

		// set a safe default action, overridable by user for something useful in production, obvs
		viper.SetDefault("command", "echo 'latency exceeded'")

		// read configuration
		audience := viper.GetString("audience")
		secret := viper.GetString("secret")
		logLevel := viper.GetString("log_level")
		logFormat := viper.GetString("log_format")
		logFile := viper.GetString("log_file")
		thresholdStr := viper.GetString("threshold")
		intervalStr := viper.GetString("interval")
		reconnectEveryStr := viper.GetString("reconnect_every")
		triggerAfterMisses := viper.GetInt("trigger_after_misses")
		noRetriggerWithinStr := viper.GetString("no_retrigger_within")
		topic := viper.GetString("topic")
		command := viper.GetString("command")

		// Sanity checks
		ok := true
		if audience == "" {
			fmt.Println("You must set RELAY_MONITOR_AUDIENCE")
			ok = false
		}
		if secret == "" {
			fmt.Println("You must set RELAY_MONITOR_SECRET")
			ok = false
		}

		// parse durations
		threshold, err := time.ParseDuration(thresholdStr)

		if err != nil {
			fmt.Print("cannot parse duration in RELAY_MONITOR_THRESHOLD=" + thresholdStr)
			ok = false
		}

		interval, err := time.ParseDuration(intervalStr)
		if err != nil {
			fmt.Print("cannot parse duration in RELAY_MONITOR_THRESHOLD=" + intervalStr)
			ok = false
		}

		noRetriggerWithin, err := time.ParseDuration(noRetriggerWithinStr)
		if err != nil {
			fmt.Print("cannot parse duration in RELAY_MONITOR_NO_RETRIGGER_WITHIN=" + noRetriggerWithinStr)
			ok = false
		}

		reconnectEvery, err := time.ParseDuration(reconnectEveryStr)
		if err != nil {
			fmt.Print("cannot parse duration in RELAY_MONITOR_RECONNECT_EVERY=" + reconnectEveryStr)
			ok = false
		}

		if !ok {
			// exit if we are missing something critical
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
			fmt.Println("RELAY_MONITOR_LOG_LEVEL can be trace, debug, info, warn, error, fatal or panic but not " + logLevel)
			os.Exit(1)
		}

		switch strings.ToLower(logFormat) {
		case "json":
			log.SetFormatter(&log.JSONFormatter{})
		case "text":
			log.SetFormatter(&log.TextFormatter{})
		default:
			fmt.Println("RELAY_MONITOR can be json or text but not " + logLevel)
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
		log.Infof("Audience: [%s]", audience)
		log.Debugf("Secret: [%s...%s]", secret[:4], secret[len(secret)-4:])

		log.Infof("Log file: [%s]", logFile)
		log.Infof("Log format: [%s]", logFormat)
		log.Infof("Log level: [%s]", logLevel)

		log.Infof("Threshold: [%s]", threshold)
		log.Infof("Interval: [%s]", interval)
		log.Infof("No retrigger within: [%s]", noRetriggerWithin)
		log.Infof("Reconnect every: [%s]", reconnectEvery)
		log.Infof("Topic: [%s]", topic)
		log.Infof("Trigger after misses: [%d]", triggerAfterMisses)
		log.Infof("Command: [%s]", command)

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

		config := monitor.Config{
			Command:            command,
			Interval:           interval,
			LatencyThreshold:   threshold,
			NoRetriggerWithin:  noRetriggerWithin,
			ReconnectEvery:     reconnectEvery,
			RelayAudience:      audience,
			RelaySecret:        secret,
			Topic:              topic,
			TriggerAfterMisses: triggerAfterMisses,
		}

		go monitor.Monitor(closed, &wg, config) //pass waitgroup to allow graceful shutdown

		wg.Wait()

	},
}

func init() {
	rootCmd.AddCommand(monitorCmd)
}
