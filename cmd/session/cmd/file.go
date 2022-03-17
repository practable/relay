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

0. PREQUISITES
--------------

If you are connecting to a session relay, then you will need a JWT token 
that is valid for the relay you are connecting to. This
will probably be supplied by the administrator of the relay (using some
commands similar to this example:

export ACCESSTOKEN_LIFETIME=86400
export ACCESSTOKEN_ROLE=client
export ACCESSTOKEN_SECRET=$($HOME/secret/session_secret.sh)
export ACCESSTOKEN_TOPIC=spin35-data
export ACCESSTOKEN_CONNECTIONTYPE=session
export ACCESSTOKEN_AUDIENCE=https://relay-access.practable.io
export SESSION_CLIENT_TOKEN=$(session token)

1. MODES OF OPERATION
---------------------

Session client file supports three main modes of operation. For all of them, you can see 
addition logging information to stdout by setting development mode. This might be useful
while building familiarity with the tool. It does not show the content of incoming messages
but reports their size.
export SESSION_CLIENT_FILE_DEVELOPMENT=true

1.1 LOG 

You must specify a session to connect to, and a file to log to
export SESSION_CLIENT_SESSION=$ACCESSTOKEN_AUDIENCE/$ACCESSTOKEN_CONNECTIONTYPE/$ACCESSTOKEN_TOPIC
export SESSION_CLIENT_FILE_LOG=/var/log/session/spin35-data.log
session client file


1.2 PLAY

You can optionally also log any messages you get back, useful for running equipment checks.

export SESSION_CLIENT_SESSION=$ACCESSTOKEN_AUDIENCE/$ACCESSTOKEN_CONNECTIONTYPE/$ACCESSTOKEN_TOPIC
export SESSION_CLIENT_FILE_LOG=/var/log/session/spin35-data.log
export SESSION_CLIENT_FILE_PLAY=/etc/practable/spin35-check.play
session client file

If you are sending messages that with a short timeout, or a very 
long timeout, on conditions, then you 
may wish to specify a custom timeout interval (default is 1s)
export SESSION_CLIENT_FILE_INTERVAL=10ms

If your play file has errors but you want to play it anyway, then 
export SESSION_CLIENT_FILE_FORCE=true

More information on the play file format is given below.


1.3 CHECK  

If you just want to check the syntax in your playfile, without 
connecting to a relay, you can check it with

export SESSION_CLIENT_FILE_CHECK_ONLY=true
export SESSION_CLIENT_FILE_PLAY=/etc/practable/spin35-check.play
session client file


2. PLAYFILE FORMAT
------------------

The playfile lets you specify messages to send, as well as control
when they are sent, either by specifying delays, or messages to 
await reception of. You can also control which messages are logged
to file, by setting filters. Note that logging ends when the playfile
has finished, so if setting the filter for a long term logging session
make sure you add a final wait command with a long duration, longer
than you plan to log for (e.g. you might set this to 8760h for one year.

2.1 DURATIONS

A duration string is an unsigned decimal number with optional fraction 
and a mandatory unit suffix, such as "300ms", "1.5h" or "2m45s". 
Valid time units are "ms", "s", "m", "h".

2.2 COMMANDS

The format of the play file is one message per line. Each line can 
have one command. The available commands are:

- comment
- wait
- send/now
- send/delay
- send/condition
- filter setting

2.3 EXAMPLES

Some example lines to show what is possible include:

# comment
#+ comment that is echoed to log file
# send/now
{"some":"msg"}
# send/delay
[1.2s] {"some":"msg"}
# send command after receiving sufficient messages matching a pattern, or timing out
# collect 5 "is running" messages, or wait 10s, whichever comes first, then send message
<'\"is\"\s*:\s*\"running\"',5,10s> {"stop":"motor"}
# wait for 100ms, without sending a message
[100ms]
# Add an accept filter
|+> ^\s*{
# Add a deny filter
|-> "hb"
# reset the filter to pass everything
|r>

2.3.1 COMMENTS

Comments come in three formats
# non-echo
#- non-echo (just more explicit)
#+ echo to log file (not sent to relay, helpful for delimiting tests)

Any line starting with # is ignored as a comment. If you want to send
a message starting with # then you simply need to prepend a delay, e.g.

# This won't be sent because it is a comment
[] # This WILL be sent (not that I'd recommend it)

2.3.1 WAIT 

If you specify a duration in square brackets on its own, no message
is sent, but playback pauses for the duration specified. E.g. pause
for two seconds:

[2s]

2.3.2: SEND/NOW

Any message that is not of a defined type, is considered a message 
to be sent right away, e.g. 

{"some":"message"}
set foo=bar

Your message format doesn't have to be JSON

2.3.3 SEND/DELAY

Each line can be prepended by an optional delay within square brackets, e.g.

[0.1s] {"some":"msg"}

A regular expression is used to separate the optional delay from 
the message, such that the message transmitted starts with the 
first non-white space character occuring after the delay. There are
no further assumptions made about the message format, so as to 
support testing with malformed JSON or other message formats as may
be required to interact with hardware or utilities attached to 
session hosts. 
Therefore it is only possible to send a message consisting of only
whitespace, if you omit the delay. Such a message may be 
useful for testing the rejection of non-messages, where the lack of
a delay is less likely to be relevant.

For readability, it may be useful to pad the delay value inside the
brackets with whitespace. It is also acceptable to have a zero delay, 
or an empty delay (for no delay)
[] valid, zero delay
[0s] valid, zero delay
[  10ms ] valid, delay of 10ms
[ 0.1s ] valid, delay of 100ms

It is not recommended to specify delays of less than one millisecond 
(0.001) because these are unlikely to be faithfully respected. Note
that all delays are simply that - an additional delay on top of any
overhead already incurred in sending the previous message. They 
should not be treated as accurate inter-message spacings.

2.3.4 SEND/CONDITION

These are intended to speed up scripts that are waiting for some
sort of variable duration task, like collecting a certain number
of results, or waiting for a calibration to finish. There is no
conditional processing of outputs i.e. there is no scripting 
available. If you require complicated behaviours, then you should 
consider making a direct connection to the relay from a tool you've 
written in your favourite method. 

Since there is no conditional processing, if the condition is not
met, then it times out, and the command is sent anyway. The format 
of the conditional is 
 
<'REGEXP',COUNT,TIMEOUT>

All arguments are mandatory. Regexp is in golang format, so note that
some lookahead options are not supported (an error will be thrown during 
the check if the regexp does not compile).

Your regexp should be enclosed in single quotes. You can include single 
quotes in your regexp if they are escaped with backslash, e.g. a 
conditional to find 'foo' looks like this:

<'\'foo\'',1,10>

Of course, if you just want to look for foo, that's simpler:
<'foo',1,10>

3. LOG ROTATION

If you are using logrotate(8), then send SIGHUP in your postscript. 
Do NOT restart the service, as you will lose any messages sent during 
the brief window that access is renegotiated (typically fast, but 
theoretically you could miss a message).

See github.com/practable/relay/scripts/examples/log-only.sh for a
demonstration of log rotation using sighup, which goes a bit like this:

# environment variable setup omitted for brevity
session client file
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

		viper.SetEnvPrefix("SESSION_CLIENT")
		viper.AutomaticEnv()

		viper.SetDefault("interval", "1s")

		session := viper.GetString("session")
		token := viper.GetString("token")
		development := viper.GetBool("file_development")
		interval := viper.GetDuration("interval")
		check := viper.GetBool("check_only")
		force := viper.GetBool("force")
		logfilename := viper.GetString("file_log")
		playfilename := viper.GetString("file_play")

		if (len(logfilename) + len(playfilename)) < 1 {
			fmt.Println("you must specify at least one filename. File(s) are specified via environment variables only. For details run `session client file -h`")
			os.Exit(1)
		}

		if session == "" && !check {
			fmt.Println("SESSION_CLIENT_SESSION not set")
			os.Exit(1)
		}

		if token == "" && !check {
			fmt.Println("SESSION_CLIENT_TOKEN not set")
			os.Exit(1)
		}

		if development {
			// development environment
			fmt.Println("Development mode - logging output to stdout")
			fmt.Printf("Session: %s\nToken: %s\nLog: %s\nPlay: %s\nCheck: %s\nForce: %s\n",
				session,
				token,
				logfilename,
				playfilename,
				strconv.FormatBool(check),
				strconv.FormatBool(check))
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

		err := file.Run(ctx, sighup, session, token, logfilename, playfilename, interval, check, force)
		if err != nil {
			fmt.Println(err.Error())
			log.Errorf("Stopping due to error: %s", err.Error())
			os.Exit(1)
		}

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
